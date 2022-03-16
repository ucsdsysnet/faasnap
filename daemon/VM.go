// MIT License
//
// Copyright (c) 2022 Lixiang Ao
//
// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction, including without limitation the rights
// to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in all
// copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
// SOFTWARE.

package daemon

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/ucsdsysnet/faasnap/models"
	"github.com/ucsdsysnet/faasnap/reap"
	"go.opencensus.io/trace"
)

type BootSource struct {
	KernelImagePath string `json:"kernel_image_path"`
	BootArgs        string `json:"boot_args"`
}

type Drive struct {
	DriveId      string `json:"drive_id"`
	PathOnHost   string `json:"path_on_host"`
	IsRootDevice bool   `json:"is_root_device"`
	IsReadOnly   bool   `json:"is_read_only"`
}

type MachineConfig struct {
	VcpuCount       int  `json:"vcpu_count"`
	MemSizeMib      int  `json:"mem_size_mib"`
	HtEnabled       bool `json:"ht_enabled"`
	TrackDirtyPages bool `json:"track_dirty_pages"`
}

type Network struct {
	namespace   string
	HostDevName string `json:"host_dev_name"`
	IfaceId     string `json:"iface_id"`
	GuestMac    string `json:"guest_mac"`
	guestAddr   string
	uniqueAddr  string
}

type VmConfig struct {
	BootSource    BootSource    `json:"boot-source"`
	Drives        []Drive       `json:"drives"`
	MachineConfig MachineConfig `json:"machine-config"`
	Networks      []Network     `json:"network-interfaces"`
}

type VM struct {
	sync.Mutex
	VmId        string    `json:"vmId"`
	Function    string    `json:"function"`
	State       string    `json:"state"`
	Socket      string    `json:"socket"`
	VMNetwork   *Network  `json:"net"`
	VmConf      *VmConfig `json:"vmConf"`
	VmPath      string    `json:"vmPath"`
	MincoreSize int       `json:"mincoreSize"`
	ReapId      string    `json:"reapId"`
	process     *os.Process
	httpc       *http.Client
	Snapshot    *Snapshot
}

func (vm *VM) Dial() error {
	vm.Lock()
	defer vm.Unlock()
	if vm.httpc == nil {
		vm.httpc = &http.Client{
			Transport: &http.Transport{
				DialContext: func(_ context.Context, _, _ string) (net.Conn, error) {
					return net.Dial("unix", vm.Socket)
				},
			},
		}
	}
	var err error
	req, _ := http.NewRequest("GET", "http://localhost/", nil)
	req.Header.Add("Accept", "application/json")
	req.Header.Add("Content-Type", "application/json")
	var resp *http.Response
	for delay := 1; delay < 32; delay *= 2 { // total 32*100-1 ms
		for i := 0; i < 100; i++ {
			resp, err = vm.httpc.Do(req)
			if err != nil {
				time.Sleep(time.Duration(delay) * time.Millisecond)
			} else {
				goto connected
			}
		}
	}
	log.Printf("dial %s timeout: %s\n", vm.VmId, err.Error())
	return err
connected:
	if resp.StatusCode > 299 {
		log.Printf("%s err response: %v\n", vm.VmId, resp)
		return fmt.Errorf("%s err response: %v", vm.VmId, resp)
	}
	return nil
}

type VMController struct {
	sync.Mutex
	config   *Config
	BasePath string              `json:"basePath"`
	Machines map[string]*VM      `json:"machines"`
	Networks map[string]*Network `json:"netInterfaces"`
	VMMPool  map[string]*VM      `json:"vmmPool"`
}

func NewVMController(config *Config) *VMController {
	vc := new(VMController)
	vc.Mutex = sync.Mutex{}
	vc.config = config
	vc.BasePath = config.BasePath
	vc.Machines = make(map[string]*VM)
	vc.Networks = make(map[string]*Network)
	vc.VMMPool = make(map[string]*VM)
	return vc
}

// func (vc *VMController) createNetwork(id int) (*network, error) {
// 	addr := fmt.Sprintf("172.16.0.%d", id+2)
// 	mac := fmt.Sprintf("AA:FC:00:00:00:%02d", id+1)
// 	device := fmt.Sprintf("tap%d", id)
// 	return &network{Address: addr, Mac: mac, Device: device}, nil
// }

func (vc *VMController) AddNetwork(req *http.Request, namespace, hostDevName, ifaceId, guestMac, guestAddr, uniqueAddr string) error {
	// TODO: verify
	vc.Networks[namespace] = &Network{namespace: namespace, HostDevName: hostDevName, IfaceId: ifaceId, GuestMac: guestMac, guestAddr: guestAddr, uniqueAddr: uniqueAddr}
	return nil
}

func (vc *VMController) StartVM(ctx *context.Context, function, kernel, image, namespace string, vcpu, memSize int) (string, error) {
	_, span := trace.StartSpan(*ctx, "startVM_setup")
	netIface, ok := vc.Networks[namespace]
	if !ok {
		return "", fmt.Errorf("network %s not found", namespace)
	}
	conf := &VmConfig{
		BootSource: BootSource{
			KernelImagePath: kernel,
			BootArgs:        "reboot=k panic=1 pci=off random.trust_cpu=on i8042.nokbd i8042.noaux",
		},
		Drives: []Drive{{
			DriveId:      "rootfs",
			PathOnHost:   image,
			IsRootDevice: true,
			IsReadOnly:   true,
		}},
		MachineConfig: MachineConfig{
			VcpuCount:       vcpu,
			MemSizeMib:      memSize,
			HtEnabled:       false,
			TrackDirtyPages: false,
		},
		Networks: []Network{*netIface},
	}

	id := RandStringRunes(8)
	vmPath := vc.BasePath + "/" + id
	if err := os.MkdirAll(vmPath, 0755); err != nil {
		log.Println(err)
		return "", err
	}

	jsonConf, err := json.Marshal(conf)
	if err != nil {
		log.Println(err)
		return "", err
	}

	configFile := vmPath + "/vm_config.json"
	if err = ioutil.WriteFile(configFile, jsonConf, 0644); err != nil {
		log.Println(err)
		return "", err
	}

	ip := "/bin/ip"
	firecracker := fnManager.config.Executables["vanilla"]

	apiSock := vmPath + "/firecracker.sock"
	outFile, err := os.Create(vmPath + "/stdout")
	if err != nil {
		log.Println(err)
		return "", err
	}
	errFile, err := os.Create(vmPath + "/stderr")
	if err != nil {
		log.Println(err)
		return "", err
	}
	span.End()

	cmd := &exec.Cmd{
		Path: ip,
		Args: []string{
			ip,
			"netns",
			"exec",
			netIface.namespace,
			firecracker,
			"--api-sock", apiSock,
			"--config-file", configFile,
		},
		Stdout: outFile,
		Stderr: errFile,
	}

	log.Println("running vm", id, " with command:", *cmd)

	_, span = trace.StartSpan(*ctx, "firecracker_start_vm")
	defer span.End()
	if err := cmd.Start(); err != nil {
		log.Println(err)
		return "", err
	}

	log.Println("vmID:", id, "Started")

	newVM := &VM{
		VmId:      id,
		Function:  function,
		State:     "uninitialized",
		Socket:    apiSock,
		VMNetwork: netIface,
		VmConf:    conf,
		VmPath:    vmPath,
		process:   cmd.Process,
	}

	vc.Lock()
	vc.Machines[id] = newVM
	vc.Unlock()

	go func(vm *VM) {
		if err := cmd.Wait(); err != nil {
			log.Println(err)
		}
		log.Println("vmID:", vm.VmId, "Stopped")
		vc.Lock()
		delete(vc.Machines, vm.VmId)
		vc.Unlock()
	}(newVM)
	return id, nil
}

func (vc *VMController) StopVM(req *http.Request, vmID string) error {
	vc.Lock()
	vm, ok := vc.Machines[vmID]
	vc.Unlock()
	if ok {
		if vm.ReapId != "" {
			log.Println("Deactivating Reap...")
			records, err := reap.Deactivate(vm.ReapId)
			if err != nil {
				log.Println("Deactivate Reap:", err)
				return err
			}
			vm.Snapshot.records = make([]uint64, len(records))
			copy(vm.Snapshot.records, records)
		}
		if err := vm.process.Signal(syscall.SIGTERM); err != nil {
			log.Println("Error calling Signal:", err)
			log.Println("Not critical if 'process already finished' because userpagefault already deactivated")
		}
		return nil
	} else {
		log.Println("vmID", vmID, "not exists")
		return errors.New("vmID not exists")
	}
}

func (vc *VMController) TakeSnapshot(r *http.Request, vmID string, snap *Snapshot) error {
	vc.Lock()
	vm, ok := vc.Machines[vmID]
	vc.Unlock()
	if !ok {
		log.Println("vmID", vmID, "not exists")
		return fmt.Errorf("vmID %v not exists", vmID)
	}

	if err := os.MkdirAll(snap.SnapshotBase, 0755); err != nil {
		log.Println(err)
		return err
	}
	_, span := trace.StartSpan(r.Context(), "vm_dial")
	vm.Dial()
	span.End()

	data := "{\"state\": \"Paused\"}"
	req, err := http.NewRequest("PATCH", "http://localhost/vm", strings.NewReader(data))
	if err != nil {
		log.Println(err)
		return err
	}
	req.Header.Add("Accept", "application/json")
	req.Header.Add("Content-Type", "application/json")
	_, span = trace.StartSpan(r.Context(), "vm_pause")
	resp, err := vm.httpc.Do(req)
	span.End()
	if err != nil {
		log.Println(err)
		return err
	}
	if resp.StatusCode < 300 {
		log.Println(vmID, "paused")
	} else {
		log.Println("pausing", vmID, "response:", resp)
		return errors.New("pausing failed")
	}

	params := struct {
		SnapshotType string `json:"snapshot_type"`
		SnapshotPath string `json:"snapshot_path"`
		MemFilePath  string `json:"mem_file_path"`
		Version      string `json:"version"`
	}{
		SnapshotType: snap.SnapshotType,
		SnapshotPath: snap.SnapshotPath,
		MemFilePath:  snap.MemFilePath,
		Version:      snap.Version,
	}
	dataBytes, err := json.Marshal(params)
	if err != nil {
		log.Println(err)
		return err
	}
	req, err = http.NewRequest("PUT", "http://localhost/snapshot/create", strings.NewReader(string(dataBytes)))
	if err != nil {
		log.Println(err)
		return err
	}
	req.Header.Add("Accept", "application/json")
	req.Header.Add("Content-Type", "application/json")
	_, span = trace.StartSpan(r.Context(), "vm_snapshot")
	resp, err = vm.httpc.Do(req)
	span.End()
	if err != nil {
		log.Println(err)
		return err
	}
	if resp.StatusCode < 300 {
		log.Println(vmID, "snapshotted")
	} else {
		log.Println("snapshot", vmID, "response:", resp)
		return errors.New("snapshotting failed")
	}

	data = "{\"state\": \"Resumed\"}"
	req, err = http.NewRequest("PATCH", "http://localhost/vm", strings.NewReader(data))
	if err != nil {
		log.Println(err)
		return err
	}
	req.Header.Add("Accept", "application/json")
	req.Header.Add("Content-Type", "application/json")
	_, span = trace.StartSpan(r.Context(), "vm_resume")
	resp, err = vm.httpc.Do(req)
	span.End()
	if err != nil {
		log.Println(err)
		return err
	}
	if resp.StatusCode < 300 {
		log.Println(vmID, "resumed")
	} else {
		log.Println("resuming", vmID, "response:", resp)
		return errors.New("resuming failed")
	}

	return nil
}

func (vc *VMController) StartVMM(ctx context.Context, enableReap bool, namespace string) (string, error) {
	var fcExecutable string
	if enableReap {
		fcExecutable = vc.config.Executables["uffd"]
	} else {
		fcExecutable = vc.config.Executables["vanilla"]
	}
	vm, err := vc.startVMM(ctx, fcExecutable, namespace)
	if err != nil {
		return "", err
	}
	vc.Lock()
	defer vc.Unlock()
	if _, ok := vc.Machines[vm.VmId]; ok {
		vc.VMMPool[vm.VmId] = vm
		return vm.VmId, nil
	}
	return "", fmt.Errorf("VMM for %s failed", vm.VmId)
}

func (vc *VMController) LoadSnapshot(r *http.Request, snapshot *Snapshot, invoc *models.Invocation, reapId string) (string, error) {
	var (
		vmId string
		vm   *VM
		err  error
	)

	if snapshot.mincoreLayers != nil {
		go func() {
			if invoc.UseWsFile {
				if true {
					snapshot.loadOnce.Do(func() {
						if err := snapshot.loadWsFile(r.Context()); err != nil {
							log.Println(err)
						}
					})
				}
			} else {
				snapshot.loadOnce.Do(func() {
					if err := snapshot.loadMincore(r.Context(), invoc.LoadMincore, false); err != nil {
						log.Println(err)
					}
				})
			}
		}()
	}

	vc.Lock()
	for k := range vc.VMMPool { // get the 1st vm from pool
		vmId = k
		vm = vc.VMMPool[vmId]
		delete(vc.VMMPool, k) // remove from pool
		break
	}
	vc.Unlock()

	if vmId == "" {
		_, span := trace.StartSpan(r.Context(), "start_vmm")
		var fcExecutable string
		if invoc.EnableReap {
			fcExecutable = vc.config.Executables["uffd"]
		} else {
			fcExecutable = vc.config.Executables["vanilla"]
		}
		vm, err = vc.startVMM(r.Context(), fcExecutable, invoc.Namespace)
		if err != nil {
			return "", err
		}
		span.End()
	}
	vm.Function = snapshot.Function

	_, span := trace.StartSpan(r.Context(), "vm_dial")
	vm.Dial()
	span.End()

	log.Println("VM pid:", vm.process.Pid)
	// time.Sleep(20 * time.Second)
	return vc.loadSnapshot(r.Context(), vm, snapshot, invoc, reapId)

}

func (vc *VMController) loadSnapshot(ctx context.Context, vm *VM, snapshot *Snapshot, invoc *models.Invocation, reapId string) (string, error) {
	var (
		err       error
		span      *trace.Span
		dataBytes []byte
	)

	if false {
		params := struct {
			SnapshotPath         string      `json:"snapshot_path"`
			MemFilePath          string      `json:"mem_file_path"`
			EnableDiffSnapshots  bool        `json:"enable_diff_snapshots"`
			EnableUserPageFaults bool        `json:"enable_user_page_faults"`
			SockFilePath         string      `json:"sock_file_path"`
			OverlayRegions       map[int]int `json:"overlay_regions"`
			WsFile               string      `json:"ws_file"`
			WsRegions            map[int]int `json:"ws_regions"`
			LoadWsFile           bool        `json:"load_ws"`
		}{
			SnapshotPath:         snapshot.SnapshotPath,
			MemFilePath:          snapshot.MemFilePath,
			EnableDiffSnapshots:  false,
			EnableUserPageFaults: true,
			SockFilePath:         vc.config.BasePath + "/" + snapshot.SnapshotId + "/uffd-" + reapId + ".sock",
			OverlayRegions:       snapshot.overlayRegions,
			WsFile:               snapshot.WsFile,
			LoadWsFile:           invoc.VmmLoadWs,
		}
		dataBytes, err = json.Marshal(params)
		if err != nil {
			log.Println(err)
			return "", err
		}
	} else {
		params := struct {
			SnapshotPath         string      `json:"snapshot_path"`
			MemFilePath          string      `json:"mem_file_path"`
			EnableDiffSnapshots  bool        `json:"enable_diff_snapshots"`
			EnableUserPageFaults bool        `json:"enable_user_page_faults"`
			SockFilePath         string      `json:"sock_file_path"`
			OverlayFilePath      string      `json:"overlay_file_path"`
			OverlayRegions       map[int]int `json:"overlay_regions"`
			WsFilePath           string      `json:"ws_file_path"`
			WsRegions            [][]int     `json:"ws_regions"`
			LoadWsFile           bool        `json:"load_ws"`
			Fadvise              string      `json:"fadvise"`
		}{
			SnapshotPath:        snapshot.SnapshotPath,
			EnableDiffSnapshots: false,
			OverlayRegions:      map[int]int{},
			WsRegions:           [][]int{},
			LoadWsFile:          invoc.VmmLoadWs,
		}
		if invoc.EnableReap {
			params.EnableUserPageFaults = true
			params.SockFilePath = vc.config.BasePath + "/" + snapshot.SnapshotId + "/uffd-" + reapId + ".sock"
		}
		if invoc.UseMemFile {
			params.MemFilePath = snapshot.MemFilePath
		}
		if invoc.OverlayRegions {
			params.OverlayFilePath = snapshot.MemFilePath
			params.OverlayRegions = snapshot.overlayRegions
		}
		if invoc.UseWsFile {
			params.WsFilePath = snapshot.WsFile
			params.WsRegions = snapshot.wsRegions
		}

		dataBytes, err = json.Marshal(params)
		if err != nil {
			log.Println(err)
			return "", err
		}
	}

	req, _ := http.NewRequest("PUT", "http://localhost/snapshot/load", strings.NewReader(string(dataBytes)))
	req.Header.Add("Accept", "application/json")
	req.Header.Add("Content-Type", "application/json")
	_, span = trace.StartSpan(ctx, "request_load_snapshot")
	var resp *http.Response
	resp, err = vm.httpc.Do(req)
	if err != nil {
		log.Println("http://localhost/snapshot/load", vm.VmId, err)
		span.End()
		return "", err
	}
	span.End()
	if err != nil {
		log.Println(err)
		return "", err
	}
	if resp.StatusCode > 299 {
		log.Println(resp)
		return "", errors.New("loading snapshot failed")
	}

	data := "{\"state\": \"Resumed\"}"
	req, err = http.NewRequest("PATCH", "http://localhost/vm", strings.NewReader(data))
	if err != nil {
		log.Println(err)
		return "", err
	}
	req.Header.Add("Accept", "application/json")
	req.Header.Add("Content-Type", "application/json")
	_, span = trace.StartSpan(ctx, "vm_resume")
	resp, err = vm.httpc.Do(req)
	span.End()
	if err != nil {
		log.Println(err)
		return "", err
	}
	if resp.StatusCode < 300 {
		log.Println(vm.VmId, "resumed")
	} else {
		log.Println("resuming", vm.VmId, "response:", resp)
		return "", errors.New("resuming failed")
	}
	vm.Snapshot = snapshot
	return vm.VmId, nil
}

func (vc *VMController) startVMM(ctx context.Context, fcExecutable, namespace string) (*VM, error) {
	id := RandStringRunes(8)
	vmPath := vc.BasePath + "/" + id
	if err := os.MkdirAll(vmPath, 0755); err != nil {
		log.Println(err)
		return nil, err
	}

	// firecracker, err := exec.LookPath("firecracker")
	// if err != nil {
	// 	log.Println(err)
	// 	return nil, err
	// }
	netIface, ok := vc.Networks[namespace]
	if !ok {
		return nil, fmt.Errorf("network %s not found", namespace)
	}

	apiSock := vmPath + "/firecracker.sock"
	outFile, err := os.Create(vmPath + "/stdout")
	if err != nil {
		log.Println(err)
		return nil, err
	}
	errFile, err := os.Create(vmPath + "/stderr")
	if err != nil {
		log.Println(err)
		return nil, err
	}
	logFile, err := os.Create(vmPath + "/log")
	if err != nil {
		log.Println(err)
		return nil, err
	}
	logFile.Close()

	ip := "/bin/ip"
	cmd := &exec.Cmd{
		Path: ip,
		Args: []string{
			ip,
			"netns",
			"exec",
			netIface.namespace,
			fcExecutable,
			"--api-sock", apiSock,
			"--level", vc.config.LogLevel,
			"--log-path", vmPath + "/log",
		},
		Stdout: outFile,
		Stderr: errFile,
	}

	log.Println("starting vmm: ", cmd)
	_, span := trace.StartSpan(ctx, "start firecracker")
	if err := cmd.Start(); err != nil {
		log.Println("running", cmd, "failed")
		log.Println(err)
		return nil, err
	}
	span.End()

	log.Println("vmID:", id, "Started")

	newVM := &VM{
		VmId:      id,
		Function:  "",
		State:     "uninitialized",
		Socket:    apiSock,
		VMNetwork: netIface,
		VmConf:    nil,
		VmPath:    vmPath,
		process:   cmd.Process,
	}

	vc.Lock()
	vc.Machines[id] = newVM
	vc.Unlock()

	go func(vm *VM) {
		if err := cmd.Wait(); err != nil {
			log.Println(err)
		}
		log.Println("vmID:", vm.VmId, "Stopped")
		vc.Lock()
		delete(vc.Machines, vm.VmId)
		delete(vc.VMMPool, vm.VmId)
		vc.Unlock()
	}(newVM)

	return newVM, nil
}

func (vc *VMController) InvokeFunction(r *http.Request, vmID string, function string, params string) (string, error) {
	vc.Lock()
	vm, ok := vc.Machines[vmID]
	vc.Unlock()
	if !ok {
		log.Println("vmID ", vmID, " not exists")
		return "", errors.New("vmID not exists")
	}

	client := &http.Client{}
	url := fmt.Sprintf("%s://%s/invoke?function=%s&redishost=%s&redispasswd=%s", "http", vm.VMNetwork.uniqueAddr+":5000", function, vc.config.RedisHost, vc.config.RedisPasswd)
	log.Println("requesting ", url, " with params: ", params)
	newReq, err := http.NewRequest("POST", url, bytes.NewReader([]byte(params)))
	newReq.Header.Set("Content-Type", "application/json; charset=utf-8")
	if err != nil {
		log.Println(err)
		return "", err
	}
	_, span := trace.StartSpan(r.Context(), "invoke_"+function)
	resp, err := client.Do(newReq)
	span.End()
	if err != nil {
		log.Println(err)
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 300 {
		log.Println(function, "invoked")
		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			log.Println("read body failed", err)
		}
		return string(body), nil
	} else {
		log.Println("invoking", function, "response:", resp)
		return "", errors.New("invoking failed")
	}
}

func (vm *VM) getDmesg(context context.Context) ([]byte, error) {
	client := &http.Client{}
	url := fmt.Sprintf("%s://%s/%s", "http", vm.VMNetwork.uniqueAddr+":5000", "dmesg")
	newReq, err := http.NewRequest("GET", url, bytes.NewReader([]byte{}))
	if err != nil {
		log.Println(err)
		return nil, err
	}
	_, span := trace.StartSpan(context, "invoke_dmesg")
	resp, err := client.Do(newReq)
	span.End()
	if err != nil {
		log.Println(err)
		return nil, err
	}
	if resp.StatusCode > 300 {
		log.Println("invoking dmesg failed. response:", resp)
		return nil, fmt.Errorf("invoking dmesg failed for vm %v: %v", vm.VmId, resp.StatusCode)

	}
	dmesgResp, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Println("Failed to read body of dmesg ", vm.VmId, err)
		return nil, err
	}
	log.Println("dmesg invoked for ", vm.VmId)
	return dmesgResp, nil
}
