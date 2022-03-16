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
	"errors"
	"fmt"
	"io"
	"log"
	"math/rand"
	"os"
	"strconv"
	"strings"
	"time"
	"unsafe"

	"golang.org/x/sys/unix"
)

var letterRunes = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ1234567890")

func RandStringRunes(n int) string {
	b := make([]rune, n)
	for i := range b {
		b[i] = letterRunes[rand.Intn(len(letterRunes))]
	}
	return string(b)
}

func CopyFile(dst, src string) error {
	source, err := os.Open(src)
	if err != nil {
		return err
	}
	defer source.Close()

	destination, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer destination.Close()
	_, err = io.Copy(destination, source)
	return err
}

func FileMincore(f *os.File, size int64) ([]bool, error) {
	// borrowed from https://github.com/tobert/pcstat/blob/master/mincore.go
	//skip could not mmap error when the file size is 0
	if int(size) == 0 {
		return nil, nil
	}
	// mmap is a []byte
	mmap, err := unix.Mmap(int(f.Fd()), 0, int(size), unix.PROT_NONE, unix.MAP_SHARED)
	if err != nil {
		return nil, fmt.Errorf("could not mmap: %v", err)
	}
	// TODO: check for MAP_FAILED which is ((void *) -1)
	// but maybe unnecessary since it looks like errno is always set when MAP_FAILED

	// one byte per page, only LSB is used, remainder is reserved and clear
	vecsz := (size + int64(os.Getpagesize()) - 1) / int64(os.Getpagesize())
	vec := make([]byte, vecsz)

	// get all of the arguments to the mincore syscall converted to uintptr
	mmap_ptr := uintptr(unsafe.Pointer(&mmap[0]))
	size_ptr := uintptr(size)
	vec_ptr := uintptr(unsafe.Pointer(&vec[0]))

	// use Go's ASM to submit directly to the kernel, no C wrapper needed
	// mincore(2): int mincore(void *addr, size_t length, unsigned char *vec);
	// 0 on success, takes the pointer to the mmap, a size, which is the
	// size that came from f.Stat(), and the vector, which is a pointer
	// to the memory behind an []byte
	// this writes a snapshot of the data into vec which a list of 8-bit flags
	// with the LSB set if the page in that position is currently in VFS cache
	ret, _, err := unix.Syscall(unix.SYS_MINCORE, mmap_ptr, size_ptr, vec_ptr)
	if ret != 0 {
		return nil, fmt.Errorf("syscall SYS_MINCORE failed: %v", err)
	}
	defer unix.Munmap(mmap)

	mc := make([]bool, vecsz)

	// there is no bitshift only bool
	for i, b := range vec {
		if b%2 == 1 {
			mc[i] = true
		} else {
			mc[i] = false
		}
	}

	return mc, nil
}

func ScanFileMincore(f *os.File, size int64, startLayer int, interval int, stop chan bool) ([]int, int, error) {
	// borrowed from https://github.com/tobert/pcstat/blob/master/mincore.go
	//skip could not mmap error when the file size is 0
	if int(size) == 0 {
		return nil, 0, nil
	}
	// mmap is a []byte
	mmap, err := unix.Mmap(int(f.Fd()), 0, int(size), unix.PROT_NONE, unix.MAP_SHARED)
	if err != nil {
		return nil, 0, fmt.Errorf("could not mmap: %v", err)
	}
	defer unix.Munmap(mmap)

	// TODO: check for MAP_FAILED which is ((void *) -1)
	// but maybe unnecessary since it looks like errno is always set when MAP_FAILED

	// one byte per page, only LSB is used, remainder is reserved and clear
	vecsz := (size + int64(os.Getpagesize()) - 1) / int64(os.Getpagesize())
	vec := make([]byte, vecsz)

	// get all of the arguments to the mincore syscall converted to uintptr
	mmap_ptr := uintptr(unsafe.Pointer(&mmap[0]))
	size_ptr := uintptr(size)
	vec_ptr := uintptr(unsafe.Pointer(&vec[0]))

	// use Go's ASM to submit directly to the kernel, no C wrapper needed
	// mincore(2): int mincore(void *addr, size_t length, unsigned char *vec);
	// 0 on success, takes the pointer to the mmap, a size, which is the
	// size that came from f.Stat(), and the vector, which is a pointer
	// to the memory behind an []byte
	// this writes a snapshot of the data into vec which a list of 8-bit flags
	// with the LSB set if the page in that position is currently in VFS cache

	mc := make([]int, vecsz)
	nlayers := 0
	var running bool = true
	for running {
		select {
		case _, ok := <-stop:
			if ok {
				log.Println("stopping mincore")
			} else {
				log.Println("stop channel close!")
			}
			running = false
		case <-time.After(time.Duration(interval) * time.Millisecond):
		}
		ret, _, err := unix.Syscall(unix.SYS_MINCORE, mmap_ptr, size_ptr, vec_ptr)
		nlayers += 1
		if ret != 0 {
			return nil, 0, fmt.Errorf("syscall SYS_MINCORE failed: %v", err)
		}
		for i, b := range vec {
			if int(b%2) > 0 && mc[i] == 0 {
				mc[i] = nlayers + startLayer
			}
		}
	}
	return mc, nlayers + startLayer, nil
}

func ScanFileMincoreBySize(f *os.File, size int64, startLayer int, pid, sizeIncr int, stop chan bool) ([]int, int, error) {
	// borrowed from https://github.com/tobert/pcstat/blob/master/mincore.go
	//skip could not mmap error when the file size is 0
	if int(size) == 0 {
		return nil, 0, nil
	}
	// mmap is a []byte
	mmap, err := unix.Mmap(int(f.Fd()), 0, int(size), unix.PROT_NONE, unix.MAP_SHARED)
	if err != nil {
		return nil, 0, fmt.Errorf("could not mmap: %v", err)
	}
	defer unix.Munmap(mmap)

	// TODO: check for MAP_FAILED which is ((void *) -1)
	// but maybe unnecessary since it looks like errno is always set when MAP_FAILED

	// one byte per page, only LSB is used, remainder is reserved and clear
	vecsz := (size + int64(os.Getpagesize()) - 1) / int64(os.Getpagesize())
	vec := make([]byte, vecsz)

	// get all of the arguments to the mincore syscall converted to uintptr
	mmap_ptr := uintptr(unsafe.Pointer(&mmap[0]))
	size_ptr := uintptr(size)
	vec_ptr := uintptr(unsafe.Pointer(&vec[0]))

	// use Go's ASM to submit directly to the kernel, no C wrapper needed
	// mincore(2): int mincore(void *addr, size_t length, unsigned char *vec);
	// 0 on success, takes the pointer to the mmap, a size, which is the
	// size that came from f.Stat(), and the vector, which is a pointer
	// to the memory behind an []byte
	// this writes a snapshot of the data into vec which a list of 8-bit flags
	// with the LSB set if the page in that position is currently in VFS cache

	mc := make([]int, vecsz)
	nlayers := 0
	var running bool = true
	reachSize := make(chan bool)

	getSize := func() (int, error) {
		data, err := os.ReadFile("/proc/" + strconv.Itoa(pid) + "/smaps_rollup")
		if err != nil {
			log.Println("read prof file", err)
			return 0, err
		}
		for _, line := range strings.Split(strings.TrimSuffix(string(data), "\n"), "\n") {
			if strings.HasPrefix(line, "Rss:") {
				fields := strings.Fields(line)
				intVar, err := strconv.Atoi(fields[len(fields)-2])
				if err != nil {
					log.Println("parsing integer", fields[len(fields)-2], err)
					return 0, err
				}
				return intVar, nil
			}
		}
		return 0, errors.New("rss not found")
	}

	go func() {
		cur := 0
		for running {
			rssSize, err := getSize()
			if err != nil {
				log.Println("get size error")
				close(reachSize)
				break
			}
			// log.Println("new size: ", rssSize/4)
			if rssSize/4 >= cur+sizeIncr {
				log.Println("oversize: ", rssSize/4-cur-sizeIncr)
				cur = rssSize / 4
				reachSize <- true
			} else {
				time.Sleep(4 * time.Millisecond)
			}
		}
	}()

	for running {
		select {
		case _, ok := <-stop:
			if ok {
				log.Println("stopping mincore")
			} else {
				log.Println("stop channel close!")
			}
			running = false
		case _, ok := <-reachSize:
			if ok {
				// log.Println("reaching size, go")
			} else {
				log.Println("reachSize channel close!")
				running = false
			}
		}
		ret, _, err := unix.Syscall(unix.SYS_MINCORE, mmap_ptr, size_ptr, vec_ptr)
		nlayers += 1
		if ret != 0 {
			return nil, 0, fmt.Errorf("syscall SYS_MINCORE failed: %v", err)
		}
		for i, b := range vec {
			if int(b%2) > 0 && mc[i] == 0 {
				mc[i] = nlayers + startLayer
			}
		}
	}

	// time.Sleep(1000 * time.Millisecond)
	final, err := getSize()
	if err != nil {
		log.Println("quited!")
	}
	log.Println("final size: ", final/4)
	return mc, nlayers + startLayer, nil
}
