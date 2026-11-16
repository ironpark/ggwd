// Copyright 2026 The ggwd Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package remote

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"unsafe"

	"golang.org/x/sys/windows"
)

// On Windows the shared memory is a file mapping backed by the paging file. A unix socket passes
// no handles there, so the SharedMemory message carries the mapping's name, which is random and
// in the session's Local namespace; the viewer opens it by name and closes its handle once mapped.
// The mapping is gone when the last handle and view are, so it outlives neither process.

var procOpenFileMappingW = windows.NewLazySystemDLL("kernel32.dll").NewProc("OpenFileMappingW")

// NewSharedMemory makes a file mapping of size bytes for the app's batches, mapped for writing;
// the viewer gets its name as a SharedMemory message.
func NewSharedMemory(size int) (*SharedMemory, []byte, error) {
	if size <= 0 {
		return nil, nil, errors.New("remote: shared memory of no size")
	}
	var r [16]byte
	if _, err := rand.Read(r[:]); err != nil {
		return nil, nil, err
	}
	name := `Local\ggwd-remote-` + hex.EncodeToString(r[:])
	n, err := windows.UTF16PtrFromString(name)
	if err != nil {
		return nil, nil, err
	}
	s := uint64(size)
	h, err := windows.CreateFileMapping(windows.InvalidHandle, nil, windows.PAGE_READWRITE, uint32(s>>32), uint32(s), n)
	if err != nil {
		if h != 0 {
			// ERROR_ALREADY_EXISTS: someone else's, which this must not write to.
			_ = windows.CloseHandle(h)
		}
		return nil, nil, fmt.Errorf("remote: CreateFileMapping: %w", err)
	}
	b, err := mapView(h, windows.FILE_MAP_WRITE, size)
	if err != nil {
		_ = windows.CloseHandle(h)
		return nil, nil, err
	}
	return &SharedMemory{Size: int64(size), Name: name, handle: uintptr(h)}, b, nil
}

// Map opens the shared memory by its name, maps it for reading, and closes the handle, which the
// mapping does not need.
func (m *SharedMemory) Map() ([]byte, error) {
	defer m.Close()
	if m.Size <= 0 || m.Size > 1<<40 || int64(int(m.Size)) != m.Size {
		return nil, errors.New("remote: shared memory of an invalid size")
	}
	n, err := windows.UTF16PtrFromString(m.Name)
	if err != nil {
		return nil, err
	}
	r, _, e := procOpenFileMappingW.Call(windows.FILE_MAP_READ, 0, uintptr(unsafe.Pointer(n)))
	if r == 0 {
		return nil, fmt.Errorf("remote: OpenFileMapping %s: %w", m.Name, e)
	}
	m.handle = r
	return mapView(windows.Handle(r), windows.FILE_MAP_READ, int(m.Size))
}

// mapView maps size bytes of the mapping h, and checks that they are there.
func mapView(h windows.Handle, access uint32, size int) ([]byte, error) {
	addr, err := windows.MapViewOfFile(h, access, 0, 0, uintptr(size))
	if err != nil {
		return nil, fmt.Errorf("remote: MapViewOfFile: %w", err)
	}
	var mbi windows.MemoryBasicInformation
	if err := windows.VirtualQuery(addr, &mbi, unsafe.Sizeof(mbi)); err != nil || mbi.RegionSize < uintptr(size) {
		_ = windows.UnmapViewOfFile(addr)
		return nil, errors.New("remote: the shared memory is smaller than it says")
	}
	return unsafe.Slice((*byte)(unsafe.Pointer(addr)), size), nil
}

// Close closes the handle of the shared memory, which a mapping of it does not need.
func (m *SharedMemory) Close() error {
	if m.handle == 0 {
		return nil
	}
	err := windows.CloseHandle(windows.Handle(m.handle))
	m.handle = 0
	return err
}

// Unmap unmaps memory Map or NewSharedMemory mapped.
func Unmap(b []byte) error {
	if len(b) == 0 {
		return nil
	}
	return windows.UnmapViewOfFile(uintptr(unsafe.Pointer(&b[0])))
}
