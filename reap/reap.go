// MIT License
//
// Copyright (c) 2020 Dmitrii Ustiugov, Plamen Petrov and EASE lab
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

package reap

import (
	"context"
	"net/http"
	"runtime"

	"go.opencensus.io/trace"
	// ctrdlog "github.com/containerd/containerd/log"
	// fccdcri "github.com/ease-lab/vhive/cri"
	// ctriface "github.com/ease-lab/vhive/ctriface"
	// hpb "github.com/ease-lab/vhive/examples/protobuf/helloworld"
	// pb "github.com/ease-lab/vhive/proto"
	log "github.com/sirupsen/logrus"
	// "google.golang.org/grpc"
)

// func invokeFunction(script string, function string) {
// 	bashExecutable, _ := exec.LookPath("bash")
// 	invokeCommand := &exec.Cmd{
// 		Path:   bashExecutable,
// 		Args:   []string{bashExecutable, script, function},
// 		Stdout: os.Stdout,
// 		Stderr: os.Stderr,
// 	}
// 	if err := invokeCommand.Run(); err != nil {
// 		fmt.Println("Error:", err)
// 	}
// }

var mmanager *MemoryManager

func Setup() {
	runtime.GOMAXPROCS(16)
	mmCfg := MemoryManagerCfg{
		MetricsModeOn: false,
	}
	mmanager = NewMemoryManager(mmCfg)

}

func Deactivate(id string) ([]uint64, error) {
	return mmanager.Deactivate(id)
	// mmanager.DumpUPFPageStats(vmID, "fn1", mmanager.instances[vmID].MetricsPath)
}

func Register(ctx context.Context, ssId string, baseDir string, vmmStatePath string, guestMemPath string, memSize int, wsFileDirectIO, wsSingleRead bool) (string, error) {
	_, span := trace.StartSpan(ctx, "reap.Register")
	defer span.End()

	return mmanager.RegisterVM(ssId, vmmStatePath, guestMemPath, baseDir, memSize, wsFileDirectIO, wsSingleRead)
}

func ClearCache(ctx context.Context, ssId string) error {
	return mmanager.ClearCache(ssId)
}

func Activate(req *http.Request, id string) error {
	log.Println("reap.Activate")
	_, span := trace.StartSpan(req.Context(), "reap.Activate")
	defer span.End()
	if err := mmanager.FetchState(req.Context(), id); err != nil {
		return err
	}
	return mmanager.Activate(req.Context(), id)
}
