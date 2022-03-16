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
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"math/rand"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/ucsdsysnet/faasnap/models"
	"github.com/ucsdsysnet/faasnap/reap"
	"github.com/ucsdsysnet/faasnap/restapi/operations"

	"contrib.go.opencensus.io/exporter/prometheus"
	"go.opencensus.io/stats/view"
	"go.opencensus.io/trace"
)

type Config struct {
	LogLevel    string            `json:"log_level"`
	BasePath    string            `json:"base_path"`
	Images      map[string]string `json:"images"`
	Kernels     map[string]string `json:"kernels"`
	Executables map[string]string `json:"executables"`
	RedisHost   string            `json:"redis_host"`
	RedisPasswd string            `json:"redis_passwd"`
}

type DaemonState struct {
	FnManager       *FunctionManager `json:"functionManager"`
	VmController    *VMController    `json:"vmController"`
	SnapshotManager *SnapshotManager `json:"snapshotManager"`
	Config          *Config          `json:"config"`
}

var fnManager *FunctionManager
var vmController *VMController
var ssManager *SnapshotManager

func registerPrometheus() *prometheus.Exporter {
	pe, err := prometheus.NewExporter(prometheus.Options{Namespace: "daemon"})
	if err != nil {
		log.Fatalf("Failed to create Prometheus exporter: %v", err)
	}
	view.RegisterExporter(pe)
	return pe
}

func Setup(s *http.Server, scheme, addr string) *DaemonState {

	var configFile = "/etc/faasnap.json"

	convertAbs := func(rel *string) {
		if !filepath.IsAbs(*rel) {
			abspath, err := filepath.Abs(*rel)
			if err != nil {
				log.Fatalf("Failed to convert path %v to absolute", *rel)
			}
			*rel = abspath
		}
	}

	convertAbs(&configFile)

	if _, err := os.Stat(configFile); os.IsNotExist(err) {
		log.Fatalf("config file %v not found", configFile)
	}
	bytes, err := ioutil.ReadFile(configFile)
	if err != nil {
		log.Fatalf("Failed to read config file at %v: %v", configFile, err)
	}
	var config Config
	json.Unmarshal(bytes, &config)

	verifyResource := func(rtype string, collection *map[string]string) {
		for alias, path := range *collection {
			if _, err := os.Stat(path); os.IsNotExist(err) {
				log.Fatalf("could not find %v %v at path %v when loading config", rtype, alias, path)
			}
			if !filepath.IsAbs(path) {
				log.Fatalf("all %v paths must be absolute: %v - %v", rtype, alias, path)
			}
		}
	}

	verifyResource("images", &config.Images)
	verifyResource("kernels", &config.Kernels)
	verifyResource("executables", &config.Executables)

	rand.Seed(time.Now().UnixNano())

	fnManager = NewFunctionManager(&config)
	vmController = NewVMController(&config)
	ssManager = NewSnapshotManager(&config)

	state := &DaemonState{
		FnManager:       fnManager,
		VmController:    vmController,
		SnapshotManager: ssManager,
	}

	// pe := registerPrometheus()
	// registerZipkin(zipkinHost, port)
	// mux := http.NewServeMux()

	// mux.HandleFunc("/invoke", invokeFunction)
	// mux.HandleFunc("/dmesg", getDmesg)
	// mux.HandleFunc("/create", createFunction)
	// mux.HandleFunc("/start", startVM)
	// mux.HandleFunc("/stop", stopVM)
	// mux.HandleFunc("/snapshot", takeSnapshot)
	// mux.HandleFunc("/load", loadSnapshot)
	// mux.HandleFunc("/prewarm", preWarm)
	// http.HandleFunc("/ui/data", handleUiData(state))
	// http.Handle("/ui/", http.StripPrefix("/ui/", http.FileServer(http.Dir("./ui/build/"))))
	// http.Handle("/metrics", pe)
	// mux.Handle("/", http.RedirectHandler("/ui/", http.StatusMovedPermanently))

	// h := &ochttp.Handler{Handler: mux}
	// if err := view.Register(ochttp.DefaultServerViews...); err != nil {
	// 	log.Fatal("Failed to register ochttp.DefaultServerViews")
	// }
	// address := fmt.Sprintf(":%v", port)
	// log.Printf("Server listening at %v! ...", address)
	// log.Fatal(http.ListenAndServe(address, h))
	reap.Setup()

	return state
}

// func GetDmesg(w http.ResponseWriter, req *http.Request) {
// 	q := req.URL.Query()
// 	vmid := q.Get("")
// 	vm, ok := vmController.Machines[vmid]
// 	if !ok {
// 		w.WriteHeader(500)
// 		w.Write([]byte(fmt.Sprintf("vm %v not found", vmid)))
// 		log.Println("vm ", vm, " not found")
// 		return
// 	}

// 	dmesg, err := vm.getDmesg(req.Context())
// 	if err != nil {
// 		w.WriteHeader(500)
// 		w.Write([]byte("Failed to get dmesg " + err.Error()))
// 		return
// 	}
// 	w.Write(dmesg)
// }

func CreateFunction(params operations.PostFunctionsParams) error {
	return fnManager.CreateFunction(*params.Function.FuncName, params.Function.Kernel, params.Function.Image, int(params.Function.Vcpu), int(params.Function.MemSize))
}

func StartVM(req *http.Request, name, ssId, namespace string) (string, error) {
	if ssId == "" {
		return DoStartVM(req.Context(), name, namespace)
	} else {
		return "", nil // TODO
	}
}

func DoStartVM(ctx context.Context, function, namespace string) (string, error) {
	_, span := trace.StartSpan(ctx, fmt.Sprintf("doStartVM_%v", function))
	defer span.End()
	if fn, ok := fnManager.Functions[function]; ok {
		if id, err := vmController.StartVM(&ctx, fn.Name, fn.Kernel, fn.Image, namespace, fn.Vcpu, fn.MemSize); err != nil {
			return "", err
		} else {
			return id, nil
		}
	} else {
		log.Println("function", function, "not exist")
		return "", errors.New("function not exists")
	}
}

func StopVM(req *http.Request, vmID string) error {
	return vmController.StopVM(req, vmID)
}

func StartVMM(ctx context.Context, enableReap bool, namespace string) (string, error) {
	_, span := trace.StartSpan(ctx, "start_vmm")
	defer span.End()
	return vmController.StartVMM(ctx, enableReap, namespace)
}

func TakeSnapshot(req *http.Request, vmID string, snapshotType string, snapshotPath string, memFilePath string, version string, recordRegions bool, sizeThreshold, intervalThreshold int) (string, error) {
	vmController.Lock()
	vm, ok := vmController.Machines[vmID]
	vmController.Unlock()
	if !ok {
		log.Println("vmID not exists: ", vmID)
		return "", errors.New("vmID not exists")
	}
	if snapshotType == "" || snapshotPath == "" || memFilePath == "" || version == "" {
		return "", errors.New("snapshot configs incomplete")
	}

	ssId := "ss_" + RandStringRunes(8)
	snap := &Snapshot{
		SnapshotId:     ssId,
		Function:       vm.Function,
		SnapshotBase:   ssManager.config.BasePath + "/" + ssId,
		SnapshotType:   snapshotType,
		MemFilePath:    memFilePath,
		SnapshotPath:   snapshotPath,
		Version:        version,
		overlayRegions: map[int]int{},
		wsRegions:      [][]int{},
		loadOnce:       new(sync.Once),
	}

	var err error
	if err = vmController.TakeSnapshot(req, vmID, snap); err != nil {
		log.Println("snapshot failed: ", err)
		return "", err
	}

	if err := ssManager.RegisterSnapshot(snap); err != nil {
		return "", err
	}
	log.Println("snap.SnapshotId:", snap.SnapshotId)

	if recordRegions {
		if err = snap.RecordRegions(req.Context(), sizeThreshold, intervalThreshold); err != nil {
			log.Println("RecordRegions failed: ", err)
			return "", err
		}
	}

	return snap.SnapshotId, nil
}

func LoadSnapshot(req *http.Request, invoc *models.Invocation, reapId string) (string, error) {
	snapshot, ok := ssManager.Snapshots[invoc.SsID]
	if !ok {
		log.Println("snapshot not exists")
		return "", errors.New("snapshot not exists")
	}
	vmID, err := vmController.LoadSnapshot(req, snapshot, invoc, reapId)
	if err != nil {
		log.Println("load snapshot failed")
		return "", err
	}

	return vmID, nil
}

func ChangeSnapshot(req *http.Request, ssID string, digHole, loadCache, dropCache bool) error {
	log.Println("ChangeSnapshot", ssID, digHole, loadCache, dropCache)
	snapshot, ok := ssManager.Snapshots[ssID]
	if !ok {
		log.Println("snapshot not exists")
		return errors.New("snapshot not exists")
	}
	return snapshot.UpdateCacheState(digHole, loadCache, dropCache)
}

func CopySnapshot(ctx context.Context, fromSnapshot, memFilePath string) (*models.Snapshot, error) {
	return ssManager.CopySnapshot(ctx, fromSnapshot, memFilePath)
}

func PutNetwork(req *http.Request, namespace, hostDevName, ifaceId, guestMac, guestAddr, uniqueAddr string) error {
	return vmController.AddNetwork(req, namespace, hostDevName, ifaceId, guestMac, guestAddr, uniqueAddr)
}

func InvokeFunction(req *http.Request, invoc *models.Invocation) (string, string, string, error) {
	var vm string
	var snapshot *Snapshot
	var finished chan bool
	var scan bool
	var reapId string
	span := trace.FromContext(req.Context())
	traceId := span.SpanContext().TraceID.String()

	switch {
	case invoc.VMID != "":
		// warm start
		var ok bool
		vmController.Lock()
		_, ok = vmController.Machines[invoc.VMID]
		vmController.Unlock()
		if !ok {
			log.Println("VM not exists")
			return "", "", traceId, errors.New("VM not exists")
		}
		vm = invoc.VMID
	case invoc.SsID != "":
		// snapshot start
		var err error
		resultChan := make(chan error)
		snapshot, ok := ssManager.Snapshots[invoc.SsID]
		if !ok {
			log.Println("Snapshot not exists")
			return "", "", traceId, errors.New("Snapshot not exists")
		}

		if invoc.EnableReap {
			reapId, err = reap.Register(req.Context(), invoc.SsID, snapshot.SnapshotBase, snapshot.SnapshotPath, snapshot.MemFilePath, snapshot.Size, invoc.WsFileDirectIo, invoc.WsSingleRead)
			if err != nil {
				log.Println("Register REAP failed", err.Error())
				return "", "", traceId, err
			}
			go func() {
				err := reap.Activate(req, reapId)
				if err != nil {
					log.Println("Activate REAP failed", err.Error())
				}
				resultChan <- err
			}()
			if vm, err = LoadSnapshot(req, invoc, reapId); err != nil {
				log.Println("Snapshot start invocation failed")
				return "", "", traceId, err
			}
			if err := <-resultChan; err != nil {
				return "", "", traceId, err
			}
			vmController.Machines[vm].ReapId = reapId
		} else {
			if vm, err = LoadSnapshot(req, invoc, ""); err != nil {
				log.Println("Snapshot start invocation failed")
				return "", "", traceId, err
			}
		}
	default:
		// cold start
		var err error
		if vm, err = DoStartVM(req.Context(), *invoc.FuncName, invoc.Namespace); err != nil {
			log.Println("Cold start invocation failed")
			return "", "", traceId, err
		}
	}

	if invoc.SsID != "" && (*invoc.Mincore >= 0 || invoc.MincoreSize > 0) {
		if *invoc.Mincore >= 0 && invoc.MincoreSize > 0 {
			log.Println("both mincore modes specified")
			return "", "", traceId, errors.New("both mincore modes specified")
		}
		snapshot = ssManager.Snapshots[invoc.SsID]
		if snapshot.mincoreLayers == nil {
			scan = true
		}
	}

	if scan {
		finished = make(chan bool)
		go snapshot.ScanMincore(req, vmController.Machines[vm].process.Pid, int(*invoc.Mincore), int(invoc.MincoreSize), finished)
		defer func() {
			go func() {
				finished <- true
			}()
		}()
	}

	resp, err := vmController.InvokeFunction(req, vm, *invoc.FuncName, invoc.Params)
	if err != nil {
		return "", "", traceId, err
	}

	// if enableReap {
	// 	go func() {
	// 		if err := reap.Deactivate(req, reapId); err != nil {
	// 			log.Println("Deactivate REAP failed", err)
	// 		}
	// 	}()
	// }

	return resp, vm, traceId, nil
}

func ChangeReapCacheState(req *http.Request, ssID string, cache bool) error {
	if !cache {
		if err := reap.ClearCache(req.Context(), ssID); err != nil {
			log.Println("reap.ClearCache error", err)
			return err
		}
		return nil
	} else {
		return errors.New("not implemented")
	}
}

func ChangeMincoreState(ctx context.Context, ssID string, fromRecordSize int, trimRegions bool, toWsFile string, inactiveWs, zeroWs bool, sizeThreshold, intervalThreshold int, nlayers []int64, dropWsCache bool) error {
	log.Println("ChangeMincoreState", nlayers, trimRegions)
	snapshot, ok := ssManager.Snapshots[ssID]
	if !ok {
		log.Println("snapshot", ssID, "not exists")
		return errors.New("snapshot not exists")
	}
	if fromRecordSize > 0 {
		if err := snapshot.EmulateMincore(ctx, fromRecordSize); err != nil {
			return err
		}
	}
	if trimRegions {
		if err := snapshot.TrimMincoreRegions(ctx); err != nil {
			return err
		}
	}
	if toWsFile != "" {
		if err := snapshot.createWsFile(ctx, toWsFile, inactiveWs, zeroWs, sizeThreshold, intervalThreshold); err != nil {
			return err
		}
	}
	if len(nlayers) > 0 {
		return snapshot.PreWarmMincore(ctx, nlayers)
	}
	if dropWsCache {
		if err := snapshot.dropWsCache(ctx); err != nil {
			return err
		}
	}
	return nil
}

func GetMincore(req *http.Request, ssID string) (*operations.GetSnapshotsSsIDMincoreOKBody, error) {
	return ssManager.GetMincore(req, ssID)
}

func CopyMincore(req *http.Request, ssID string, source string) error {
	return ssManager.CopyMincore(req, ssID, source)
}

func AddMincoreLayer(req *http.Request, ssID string, position int, fromDiff string) error {
	return ssManager.AddMincoreLayer(req, ssID, position, fromDiff)
}

// func getDmesg(w http.ResponseWriter, req *http.Request) {
// 	q := req.URL.Query()
// 	vmid := q.Get(vmID)
// 	vm, ok := vmController.Machines[vmid]
// 	if !ok {
// 		w.WriteHeader(500)
// 		w.Write([]byte(fmt.Sprintf("vm %v not found", vmid)))
// 		log.Println("vm ", vm, " not found")
// 		return
// 	}

// 	dmesg, err := vm.getDmesg(req.Context())
// 	if err != nil {
// 		w.WriteHeader(500)
// 		w.Write([]byte("Failed to get dmesg " + err.Error()))
// 		return
// 	}
// 	w.Write(dmesg)
// }
