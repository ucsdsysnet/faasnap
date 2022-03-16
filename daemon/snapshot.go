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
	"container/list"
	"context"
	"errors"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"sort"
	"sync"
	"syscall"

	"github.com/ucsdsysnet/faasnap/models"
	"github.com/ucsdsysnet/faasnap/restapi/operations"
	"go.opencensus.io/trace"
	"golang.org/x/sys/unix"
)

type Snapshot struct {
	sync.Mutex
	Function            string `json:"function"`
	MemFilePath         string `json:"memFilePath"`
	loadOnce            *sync.Once
	records             []uint64
	mincoreLayers       []int
	mincoreCurrentLayer int
	nonZero             []bool
	overlayRegions      map[int]int // offset->length
	wsRegions           [][]int     // [[offset, length]...]
	WsFile              string      `json:"wsFile"`
	Size                int         `json:"size"`
	BlockSize           int         `json:"blockSize"`
	SnapshotBase        string      `json:"snapshotBase"`
	SnapshotType        string      `json:"snapshotType"`
	SnapshotId          string      `json:"snapshotId"`
	SnapshotPath        string      `json:"snapshotPath"`
	Version             string      `json:"functionVersion"`
}

type SnapshotManager struct {
	sync.Mutex
	Snapshots map[string]*Snapshot `json:"snapshots"`
	config    *Config
}

func NewSnapshotManager(config *Config) *SnapshotManager {
	return &SnapshotManager{
		Mutex:     sync.Mutex{},
		Snapshots: map[string]*Snapshot{},
		config:    config,
	}
}

func (sm *SnapshotManager) CopySnapshot(ctx context.Context, src, memFilePath string) (*models.Snapshot, error) {
	oldSnap, ok := sm.Snapshots[src]
	if !ok {
		log.Println("snapshot not exists")
		return nil, errors.New("snapshot not exists")
	}

	newSsId := "ss_" + RandStringRunes(8)
	newSnap := &Snapshot{
		Function:            oldSnap.Function,
		MemFilePath:         memFilePath,
		loadOnce:            new(sync.Once),
		records:             oldSnap.records,
		mincoreLayers:       oldSnap.mincoreLayers,
		mincoreCurrentLayer: oldSnap.mincoreCurrentLayer,
		nonZero:             oldSnap.nonZero,
		overlayRegions:      oldSnap.overlayRegions,
		wsRegions:           oldSnap.wsRegions,
		WsFile:              oldSnap.WsFile,
		Size:                oldSnap.Size,
		BlockSize:           oldSnap.BlockSize,
		SnapshotBase:        oldSnap.SnapshotBase,
		SnapshotType:        oldSnap.SnapshotType,
		SnapshotId:          newSsId,
		SnapshotPath:        oldSnap.SnapshotPath,
		Version:             oldSnap.Version,
	}

	if err := CopyFile(newSnap.MemFilePath, oldSnap.MemFilePath); err != nil {
		return nil, err
	}

	if oldSnap.WsFile != "" {
		newSnap.WsFile = oldSnap.WsFile + "." + newSsId
		if err := CopyFile(newSnap.WsFile, oldSnap.WsFile); err != nil {
			return nil, err
		}
	}

	sm.Lock()
	sm.Snapshots[newSsId] = newSnap
	sm.Unlock()
	vmId := ""
	return &models.Snapshot{SsID: newSsId, MemFilePath: newSnap.MemFilePath, VMID: &vmId}, nil
}

func (sm *SnapshotManager) RegisterSnapshot(snapshot *Snapshot) error {
	f, err := os.OpenFile(snapshot.MemFilePath, os.O_RDONLY, 0644)
	if err != nil {
		log.Println("open", snapshot.MemFilePath, "failed", err)
		return err
	}
	defer f.Close()
	fi, err := f.Stat()
	if err != nil {
		log.Println(err)
		return err
	}
	snapshot.Size = int(fi.Size())
	sm.Lock()
	sm.Snapshots[snapshot.SnapshotId] = snapshot
	sm.Unlock()
	return nil
}

func (sm *SnapshotManager) GetMincore(r *http.Request, src string) (*operations.GetSnapshotsSsIDMincoreOKBody, error) {
	source, ok := sm.Snapshots[src]
	if !ok {
		log.Println("snapshot", src, "not exists")
		return nil, errors.New("snapshot not exists")
	}
	if source.mincoreLayers == nil {
		log.Println("mincore for", src, "does not exist")
		return nil, errors.New("mincore does not exist")
	}
	nzRegionSize := 0
	for _, length := range source.overlayRegions {
		nzRegionSize += length
	}
	wsRegionSize := 0
	for _, item := range source.wsRegions {
		wsRegionSize += item[1]
	}
	return &operations.GetSnapshotsSsIDMincoreOKBody{
		Nlayers:      int64(source.mincoreCurrentLayer),
		NNzRegions:   int64(len(source.overlayRegions)),
		NzRegionSize: int64(nzRegionSize),
		NWsRegions:   int64(len(source.wsRegions)),
		WsRegionSize: int64(wsRegionSize),
	}, nil
}

func (sm *SnapshotManager) CopyMincore(r *http.Request, dst string, src string) error {
	source, ok := sm.Snapshots[src]
	if !ok {
		log.Println("snapshot", src, "not exists")
		return errors.New("snapshot not exists")
	}
	dest, ok := sm.Snapshots[dst]
	if !ok {
		log.Println("snapshot", dst, "not exists")
		return errors.New("snapshot not exists")
	}
	source.Lock()
	defer source.Unlock()
	dest.Lock()
	defer dest.Unlock()
	if len(source.mincoreLayers) > 0 {
		dest.mincoreLayers = make([]int, len(source.mincoreLayers))
		copy(dest.mincoreLayers, source.mincoreLayers)
	}
	dest.mincoreCurrentLayer = source.mincoreCurrentLayer
	// sum := func(list []int) int {
	// 	ret := 0
	// 	for _, v := range list {
	// 		ret += v
	// 	}
	// 	return ret
	// }
	// log.Println("src:", sum(source.mincoreLayers))
	// log.Println("dst:", sum(dest.mincoreLayers))
	return nil
}

func (sm *SnapshotManager) AddMincoreLayer(req *http.Request, ssID string, position int, fromDiff string) error {
	snapshot, ok := sm.Snapshots[ssID]
	if !ok {
		log.Println("snapshot", ssID, "not exists")
		return errors.New("snapshot not exists")
	}
	other, ok := sm.Snapshots[fromDiff]
	if !ok {
		log.Println("snapshot", fromDiff, "not exists")
		return errors.New("snapshot not exists")
	}

	fa, err := os.OpenFile(snapshot.MemFilePath, os.O_RDWR, 0644)
	if err != nil {
		log.Println("open", snapshot.MemFilePath, "failed", err)
		return err
	}
	defer fa.Close()
	fb, err := os.OpenFile(other.MemFilePath, os.O_RDWR, 0644)
	if err != nil {
		log.Println("open", other.MemFilePath, "failed", err)
		return err
	}
	defer fb.Close()
	statA, err := fa.Stat()
	if err != nil {
		log.Println(err)
		return err
	}
	mmapA, err := unix.Mmap(int(fa.Fd()), 0, int(statA.Size()), unix.PROT_READ, unix.MAP_PRIVATE)
	if err != nil {
		log.Println(err)
		return err
	}
	defer unix.Munmap(mmapA)
	statB, err := fb.Stat()
	if err != nil {
		log.Println(err)
		return err
	}
	mmapB, err := unix.Mmap(int(fb.Fd()), 0, int(statB.Size()), unix.PROT_READ, unix.MAP_PRIVATE)
	if err != nil {
		log.Println(err)
		return err
	}
	defer unix.Munmap(mmapB)
	pagesize := os.Getpagesize()
	size := statA.Size()
	result := make([]bool, (size+int64(pagesize)-1)/int64(pagesize))
	zeros := make([]byte, pagesize)
	true_count := 0
	for cur := 0; cur < len(result); cur += 1 {
		upper := (cur + 1) * pagesize
		if int(size) < upper {
			upper = int(size)
		}
		if bytes.Equal(mmapA[cur*pagesize:upper], mmapB[cur*pagesize:upper]) &&
			!bytes.Equal(mmapA[cur*pagesize:upper], zeros[0:upper-cur*pagesize]) {
			result[cur] = true
			true_count += 1
		}
	}
	log.Println("inserting", true_count, "pages to layer", position)

	return snapshot.InsertMincoreLayer(result, position)
}

func (snapshot *Snapshot) InsertMincoreLayer(layer []bool, position int) error {
	if position < 1 {
		return errors.New("position must >= 1")
	}
	if snapshot.mincoreLayers == nil {
		snapshot.mincoreLayers = make([]int, len(layer))
	}
	for i, v := range layer {
		if v {
			switch {
			case snapshot.mincoreLayers[i] == 0:
				snapshot.mincoreLayers[i] = position
			case snapshot.mincoreLayers[i] > position:
				snapshot.mincoreLayers[i] = position
			default:
			}
		} else {
			if snapshot.mincoreLayers[i] >= position {
				snapshot.mincoreLayers[i] += 1
			}
		}
	}
	snapshot.mincoreCurrentLayer += 1
	return nil
}

func (snapshot *Snapshot) UpdateCacheState(digHole, loadCache, dropCache bool) error {
	if digHole {
		fallocate, err := exec.LookPath("fallocate")
		if err != nil {
			log.Println("can't find fallocate:", err)
			return err
		}
		cmd := &exec.Cmd{
			Path: fallocate,
			Args: []string{
				"fallocate",
				"-d",
				snapshot.MemFilePath,
			},
		}
		if err := cmd.Start(); err != nil {
			log.Println("start", cmd, "failed:", err)
			return err
		}
		if err := cmd.Wait(); err != nil {
			log.Println(cmd, "error:", err)
			return err
		}
	}

	f, err := os.OpenFile(snapshot.MemFilePath, os.O_RDWR, 0644)
	if err != nil {
		log.Println("open", snapshot.MemFilePath, "failed", err)
		return err
	}
	defer f.Close()

	if loadCache {
		blackHole, err := os.OpenFile("/dev/null", os.O_RDWR, 0644)
		if err != nil {
			log.Println("open /dev/null failed", err)
			return err
		}
		if _, err := io.Copy(blackHole, f); err != nil {
			return err
		}
		if err := blackHole.Close(); err != nil {
			return err
		}
	}
	if dropCache {
		log.Println("dropping cache of", snapshot.MemFilePath)
		return unix.Fadvise(int(f.Fd()), 0, int64(snapshot.Size), unix.FADV_DONTNEED)
	}
	return nil
}

// func (snapshot *Snapshot) loadMincore(r *http.Request) error {
// 	_, span := trace.StartSpan(r.Context(), "load_mincore")
// 	defer span.End()

// 	snapshot.Lock()
// 	if snapshot.mmap == nil {
// 		f, _ := os.OpenFile(snapshot.MemFilePath, os.O_RDWR, 0644)
// 		defer f.Close()
// 		fi, err := f.Stat()
// 		if err != nil {
// 			log.Println(err)
// 			return err
// 		}
// 		snapshot.Size = int(fi.Size())

// 		snapshot.mmap, err = unix.Mmap(int(f.Fd()), 0, int(fi.Size()), unix.PROT_READ, unix.MAP_PRIVATE)
// 		if err != nil {
// 			log.Println(err)
// 			return err
// 		}
// 		if err := unix.Madvise(snapshot.mmap, syscall.MADV_NORMAL); err != nil {
// 			log.Println(err)
// 			return err
// 		}

// 	}
// 	snapshot.Unlock()
// 	defer func() { // race condition
// 		unix.Munmap(snapshot.mmap)
// 		snapshot.mmap = nil
// 	}()

// 	pagesize := os.Getpagesize()

// 	if snapshot.mincore != nil {
// 		vecsize := len(snapshot.mincore)
// 		value := byte(0)
// 		for cur := 0; cur < vecsize; cur++ {
// 			if snapshot.mincore[cur] {
// 				if cur*pagesize >= snapshot.Size || int64(cur*pagesize) < 0 {
// 					log.Fatal("cur*pagesize out of range: ", cur*pagesize)
// 				}
// 				value ^= snapshot.mmap[cur*pagesize]
// 			}
// 		}
// 		log.Printf("value: %d\n", value)
// 		return nil
// 	} else if snapshot.mincoreLayers != nil { // mincore layers
// 		value := byte(0)
// 		vecsize := len(snapshot.mincoreLayers)
// 		_, span2 := trace.StartSpan(r.Context(), "load_mincoreLayers")
// 		count := 0
// 		defer span2.End()
// 		for layer := snapshot.MincoreNlayers; layer > 0; layer-- {
// 			layerCount := 0
// 			for cur := 0; cur < vecsize; cur++ {
// 				if snapshot.mincoreLayers[cur] == layer {
// 					if cur*pagesize >= snapshot.Size || int64(cur*pagesize) < 0 {
// 						log.Fatal("cur*pagesize out of range: ", cur*pagesize)
// 					}
// 					value ^= snapshot.mmap[cur*pagesize]
// 					count += 1
// 					layerCount += 1
// 				}
// 			}
// 			log.Printf("layer %d: %d pages\n", layer, layerCount)
// 		}
// 		log.Printf("value: %d, nlayers: %d, pages: %d\n", value, snapshot.MincoreNlayers, count)
// 		return nil
// 	} else {
// 		log.Println("mincore and mincoreLayers not exist!")
// 		return errors.New("mincore and mincoreLayers not exist")
// 	}
// }

func (snapshot *Snapshot) EmulateMincore(ctx context.Context, mincoreSize int) error {
	var (
		curLayer  int
		layerSize int
	)
	if len(snapshot.mincoreLayers) != 0 || len(snapshot.records) == 0 {
		log.Println("EmulateMincore: mincore exists or records do not exist")
		return errors.New("mincore exists or records do not exist")
	}
	pagesize := os.Getpagesize()
	layers := make([]int, snapshot.Size/pagesize)
	if mincoreSize > 0 {
		layerSize = mincoreSize
	} else {
		layerSize = 1024
	}
	for i, addr := range snapshot.records {
		curLayer = i/layerSize + 1
		layers[addr/uint64(pagesize)] = curLayer
	}
	for i := 1; i <= curLayer; i += 1 {
		count := 0
		for _, l := range layers {
			if l == i {
				count += 1
			}
		}
		log.Println("layers:", i, "count:", count)
	}
	snapshot.mincoreLayers = layers
	snapshot.mincoreCurrentLayer = curLayer
	log.Println("Emulated mincore, layers:", curLayer)
	return nil
}

func (snapshot *Snapshot) ScanMincore(r *http.Request, pid, scanInterval, sizeIncr int, finished chan bool) error {
	var (
		mincore []int
		cur     int
		err     error
	)
	_, span := trace.StartSpan(r.Context(), "scan_mincore")
	defer span.End()
	f, _ := os.OpenFile(snapshot.MemFilePath, os.O_RDWR, 0644)
	defer f.Close()
	fi, err := f.Stat()
	if err != nil {
		log.Println(err)
		return err
	}
	if scanInterval > 0 {
		mincore, cur, err = ScanFileMincore(f, fi.Size(), snapshot.mincoreCurrentLayer, scanInterval, finished)
	} else if sizeIncr > 0 {
		mincore, cur, err = ScanFileMincoreBySize(f, fi.Size(), snapshot.mincoreCurrentLayer, pid, sizeIncr, finished)
	}
	if err != nil {
		log.Println(err)
		return err
	}
	log.Println(cur-snapshot.mincoreCurrentLayer, "layers scanned")
	snapshot.mincoreLayers = mincore
	count := 0
	for _, b := range snapshot.mincoreLayers {
		if b > 0 {
			count += 1
		}
	}
	log.Println("scanned mincore size:", count)
	snapshot.mincoreCurrentLayer = cur
	log.Println("snapshot.mincoreCurrentLayer:", snapshot.mincoreCurrentLayer)
	return nil
}

func (snapshot *Snapshot) loadMincore(ctx context.Context, layers []int64, retainMmap bool) error {
	_, span := trace.StartSpan(ctx, "load_mincore")
	defer span.End()
	log.Println("prewarming", snapshot.SnapshotId, "by", layers, "layers")

	f, err := os.OpenFile(snapshot.MemFilePath, os.O_RDWR, 0644)
	if err != nil {
		log.Println("OpenFile mem file", err)
		return err
	}
	defer f.Close()

	mmap, err := unix.Mmap(int(f.Fd()), 0, int(snapshot.Size), unix.PROT_READ, unix.MAP_PRIVATE)
	if err != nil {
		log.Println(err)
		return err
	}
	if err := unix.Madvise(mmap, syscall.MADV_NORMAL); err != nil {
		log.Println(err)
		return err
	}
	if !retainMmap {
		defer unix.Munmap(mmap)
	}

	pagesize := os.Getpagesize()

	if snapshot.mincoreLayers != nil { // mincore layers
		value := byte(0)
		vecsize := len(snapshot.mincoreLayers)
		_, span2 := trace.StartSpan(ctx, "load_mincoreLayers")
		count := 0
		defer span2.End()
		for _, layer := range layers {
			layerCount := 0
			for cur := 0; cur < vecsize; cur++ {
				if snapshot.mincoreLayers[cur] == int(layer) {
					if (cur*pagesize) >= snapshot.Size || cur*pagesize < 0 {
						log.Fatal("cur*pagesize out of range: ", cur*pagesize)
					}
					value ^= mmap[cur*pagesize]
					count += 1
					layerCount += 1
				}
			}
			log.Printf("layer %d: %d pages\n", layer, layerCount)
		}
		log.Printf("loaded nlayers: %d, pages: %d; value: %d; \n", len(layers), count, value)
		return nil
	} else {
		log.Println("mincore and mincoreLayers not exist!")
		return errors.New("mincore and mincoreLayers not exist")
	}
}

func (snapshot *Snapshot) PreWarmMincore(ctx context.Context, nlayers []int64) error {
	snapshot.UpdateCacheState(false, false, true)
	return snapshot.loadMincore(ctx, nlayers, false)
}

func (snapshot *Snapshot) RecordRegions(ctx context.Context, sizeThreshold, intervalThreshold int) error {
	_, span := trace.StartSpan(ctx, "record_regions")
	defer span.End()
	snapshot.Lock()
	defer snapshot.Unlock()

	f, err := os.OpenFile(snapshot.MemFilePath, os.O_RDWR, 0644)
	if err != nil {
		log.Println("OpenFile:", err)
		return err
	}
	defer f.Close()
	statfs := unix.Statfs_t{}
	if err = unix.Statfs(snapshot.MemFilePath, &statfs); err != nil {
		log.Println("Statfs:", err)
		return err
	}
	snapshot.BlockSize = int(statfs.Bsize)

	mmap, err := unix.Mmap(int(f.Fd()), 0, int(snapshot.Size), unix.PROT_READ, unix.MAP_PRIVATE)
	if err != nil {
		log.Println("Mmap failed:", err)
		return err
	}
	defer unix.Munmap(mmap)

	snapshot.GetNonZeroRegions(mmap, snapshot.BlockSize, sizeThreshold, intervalThreshold)
	// (*snapshot.Regions)[0] = snapshot.Size

	total := 0
	count := 0
	for _, v := range snapshot.overlayRegions {
		count += 1
		total += v
	}
	log.Println("non zero regions:", count, "total pages:", total)
	return nil
}

func (snapshot *Snapshot) GetNonZeroRegions(buf []byte, blockSize, sizeThreshold, intervalThreshold int) {
	nz := []bool{}
	regions := map[int]int{}

	// kind 0: zero block, 1: non-zero block
	kind := func(block []byte) int {
		for _, v := range block {
			if v != 0 {
				return 1
			}
		}
		return 0
	}
	// create a linked list of all regions. each element has (type, start, length)
	type Region struct {
		kind   int
		start  int
		length int
	}
	l := list.New()
	l.PushBack(&Region{kind: -1, start: 0, length: 0})
	for offset := 0; offset < len(buf); offset += blockSize { // traverse all blocks
		newKind := kind(buf[offset : offset+blockSize])
		if newKind != l.Back().Value.(*Region).kind { // new kind of block
			l.PushBack(&Region{newKind, offset / blockSize, 1})
		} else { // same kind, extend the last one
			l.Back().Value.(*Region).length += 1
		}
		nz = append(nz, newKind == 1)
	}

	l.Remove(l.Front())

	// first remove small intervals
	if l.Front().Next() != nil {
		for e := l.Front().Next(); e != l.Back() && e != nil; {
			r := e.Value.(*Region)
			if r.kind == 0 && r.length < intervalThreshold { // a small zero region to be dropped
				e.Prev().Value.(*Region).length += r.length + e.Next().Value.(*Region).length
				l.Remove(e.Next())
				next := e.Next()
				l.Remove(e)
				e = next
			} else {
				e = e.Next()
			}
		}
	}

	// first merge regions separated by small intervals
	if l.Len() > 1 {
		for e := l.Front().Next(); e != nil && e != l.Back(); {
			r := e.Value.(*Region)
			if r.kind == 0 && r.length < intervalThreshold { // a small zero region to be dropped
				e.Prev().Value.(*Region).length += r.length + e.Next().Value.(*Region).length
				l.Remove(e.Next())
				next := e.Next()
				l.Remove(e)
				e = next
			} else {
				e = e.Next()
			}
		}
	}

	// then drop small regions
	for e := l.Front(); e != nil; {
		r := e.Value.(*Region)
		next := e.Next()
		if r.kind == 1 && r.length < sizeThreshold {
			l.Remove(e)
		}
		e = next
	}

	for e := l.Front(); e != nil; e = e.Next() {
		r := e.Value.(*Region)
		if r.kind == 1 {
			regions[r.start] = r.length
		}
	}
	snapshot.nonZero = nz
	snapshot.overlayRegions = regions
}

func (snapshot *Snapshot) TrimMincoreRegions(ctx context.Context) error {
	// count := 0
	// for _, b := range snapshot.mincoreLayers {
	// 	if b > 0 {
	// 		count += 1
	// 	}
	// }
	// log.Println("Before trim:", count)
	// newLayers := make([]int, len(snapshot.mincoreLayers))
	// for offset, length := range snapshot.overlayRegions {
	// 	start := offset
	// 	end := offset + length
	// 	copy(newLayers[start:end], snapshot.mincoreLayers[start:end])
	// }
	// snapshot.mincoreLayers = newLayers
	// count = 0
	// for _, b := range snapshot.mincoreLayers {
	// 	if b > 0 {
	// 		count += 1
	// 	}
	// }
	// log.Println("After trim:", count)
	return nil
}

func (snapshot *Snapshot) createWsRegions(ctx context.Context, withInactive, withZero bool, sizeThreshold, intervalThreshold int) {
	type Region struct {
		start   int
		len     int
		layer   int
		include bool
	}
	result := make([]Region, 0)
	// inRegion := false
	// regionStart := 0
	// regionLayer := 0
	// var i int
	// for i = 0; i < len(snapshot.mincoreLayers); i += 1 { // traverse all pages
	// 	if inRegion {
	// 		if (!withInactive && snapshot.mincoreLayers[i] == 0) || (withInactive && snapshot.mincoreLayers[i] == 0 && !snapshot.nonZero[i]) { // region stops
	// 			inRegion = false
	// 			result = append(result, Region{regionStart, i - regionStart, regionLayer})
	// 		} else {
	// 			if snapshot.mincoreLayers[i] > regionLayer {
	// 				regionLayer = snapshot.mincoreLayers[i]
	// 			}
	// 		}
	// 	} else {
	// 		if snapshot.mincoreLayers[i] != 0 || (withInactive && snapshot.nonZero[i]) { // region starts
	// 			inRegion = true
	// 			regionStart = i
	// 			regionLayer = snapshot.mincoreLayers[i]
	// 		}
	// 	}
	// }
	// if inRegion { // last page
	// 	result = append(result, Region{regionStart, i - regionStart, regionLayer})
	// }
	include := func(i int) bool {
		if withInactive {
			if withZero {
				return snapshot.mincoreLayers[i] > 0 || snapshot.nonZero[i]
			} else {
				return snapshot.nonZero[i]
			}
		} else {
			if withZero {
				return snapshot.mincoreLayers[i] > 0
			} else {
				return snapshot.mincoreLayers[i] > 0 && snapshot.nonZero[i]
			}
		}
	}
	l := list.New()
	l.PushBack(&Region{start: 0, len: 1, layer: snapshot.mincoreLayers[0], include: include(0)})
	for i := 1; i < len(snapshot.mincoreLayers); i += 1 { // traverse all pages
		last := l.Back().Value.(*Region)
		if include(i) == last.include {
			last.len += 1
			if snapshot.mincoreLayers[i] > 0 && snapshot.mincoreLayers[i] < last.layer {
				last.layer = snapshot.mincoreLayers[i] // elevate layer
			}
		} else { // a change
			l.PushBack(&Region{start: i, len: 1, layer: snapshot.mincoreLayers[i], include: include(i)})
		}
	}
	l.Remove(l.Front())

	// first merge regions separated by small intervals
	if l.Len() > 1 {
		for e := l.Front().Next(); e != nil && e != l.Back(); {
			r := e.Value.(*Region)
			if !r.include && r.len < intervalThreshold { // a small non-include region to be dropped
				e.Prev().Value.(*Region).len += r.len + e.Next().Value.(*Region).len
				if e.Next().Value.(*Region).layer > e.Prev().Value.(*Region).layer {
					e.Prev().Value.(*Region).layer = e.Next().Value.(*Region).layer // elevate layer
				}
				l.Remove(e.Next())
				next := e.Next()
				l.Remove(e)
				e = next
			} else {
				e = e.Next()
			}
		}
	}

	// then drop small regions
	for e := l.Front(); e != nil; {
		r := e.Value.(*Region)
		next := e.Next()
		if r.include && r.len < sizeThreshold {
			l.Remove(e)
		}
		e = next
	}

	for e := l.Front(); e != nil; e = e.Next() {
		r := e.Value.(*Region)
		if r.include {
			result = append(result, *r)
		}
	}

	// sort and store
	sort.Slice(result, func(i, j int) bool {
		if result[i].layer == result[j].layer {
			return result[i].start < result[j].start
		}
		if result[i].layer == 0 {
			return false
		}
		if result[j].layer == 0 {
			return true
		}
		return result[i].layer < result[j].layer
	})
	snapshot.wsRegions = make([][]int, 0)
	for _, region := range result {
		snapshot.wsRegions = append(snapshot.wsRegions, []int{region.start, region.len})
	}
	log.Println("created", len(snapshot.wsRegions), "Ws Regions")
}

func (snapshot *Snapshot) createWsFile(ctx context.Context, wsFilePath string, withInactive, withZero bool, sizeThreshold, intervalThreshold int) error {
	snapshot.createWsRegions(ctx, withInactive, withZero, sizeThreshold, intervalThreshold)
	f, err := os.OpenFile(snapshot.MemFilePath, os.O_RDONLY, 0644)
	if err != nil {
		log.Println("OpenFile", snapshot.MemFilePath, "failed:", err)
		return err
	}
	defer f.Close()
	mmSrc, err := unix.Mmap(int(f.Fd()), 0, snapshot.Size, unix.PROT_READ, unix.MAP_PRIVATE)
	if err != nil {
		log.Println("Mmap failed:", err)
		return err
	}
	defer unix.Munmap(mmSrc)

	wsFile, err := os.OpenFile(wsFilePath, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0644)
	if err != nil {
		log.Println("OpenFile", wsFilePath, "failed:", err)
		return err
	}
	defer wsFile.Close()

	page_size := os.Getpagesize()
	pageCount := 0
	for _, region := range snapshot.wsRegions {
		if _, err := wsFile.Write(mmSrc[region[0]*page_size : (region[0]+region[1])*page_size]); err != nil {
			log.Println("Write failed:", err)
			return err
		}
		pageCount += region[1]
	}
	snapshot.WsFile = wsFilePath
	log.Println("wsfile created, pages:", pageCount, ", bytes:", pageCount*page_size)
	return nil
}

func (snapshot *Snapshot) loadWsFile(ctx context.Context) error {
	_, span := trace.StartSpan(ctx, "load_ws_file")
	defer span.End()
	f, err := os.OpenFile(snapshot.WsFile, os.O_RDWR, 0644)
	if err != nil {
		log.Println("OpenFile:", err)
		return err
	}
	defer f.Close()
	fi, err := f.Stat()
	if err != nil {
		log.Println("Stat ws file:", err)
		return err
	}
	if false {
		if err := unix.Fadvise(int(f.Fd()), 0, fi.Size(), unix.FADV_SEQUENTIAL); err != nil {
			log.Println("Fadvise:", err)
			return err
		}
	}
	mm, err := unix.Mmap(int(f.Fd()), 0, int(fi.Size()), unix.PROT_READ, unix.MAP_PRIVATE)
	if err != nil {
		log.Println("Mmap:", err)
		return err
	}
	defer unix.Munmap(mm)

	if false {
		if err := unix.Mlock(mm); err != nil {
			log.Println("unix.Mlock:", err)
			return err
		}
	} else {
		pagesize := os.Getpagesize()
		value := byte(0)
		count := 0
		for cur := 0; cur < int(fi.Size()); cur += pagesize {
			count += 1
			value ^= mm[cur]
		}
		log.Println("ws file pages loaded:", count, "; value", value)
	}
	return nil
}

func (snapshot *Snapshot) dropWsCache(ctx context.Context) error {
	if snapshot.WsFile == "" {
		return nil
	}
	f, err := os.OpenFile(snapshot.WsFile, os.O_RDWR, 0644)
	if err != nil {
		log.Println("OpenFile:", err)
		return err
	}
	defer f.Close()
	fi, err := f.Stat()
	if err != nil {
		log.Println("Stat ws file:", err)
		return err
	}
	return unix.Fadvise(int(f.Fd()), 0, fi.Size(), unix.FADV_DONTNEED)
}
