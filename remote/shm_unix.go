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

//go:build unix && !ios && !android

package remote

import (
	"errors"
	"syscall"
)

// NewSharedMemory makes a file of size bytes for the app's batches, mapped for writing. The file
// has no name by the time it returns, so nothing of it outlives the processes that have it open,
// however they end; the viewer gets it as a SharedMemory message. On Linux it is a memfd, which
// needs no file system; elsewhere a temporary file, unlinked at once.
func NewSharedMemory(size int) (*SharedMemory, []byte, error) {
	if size <= 0 {
		return nil, nil, errors.New("remote: shared memory of no size")
	}
	f, err := anonymousFile()
	if err != nil {
		return nil, nil, err
	}
	if err := f.Truncate(int64(size)); err != nil {
		_ = f.Close()
		return nil, nil, err
	}
	b, err := syscall.Mmap(int(f.Fd()), 0, size, syscall.PROT_READ|syscall.PROT_WRITE, syscall.MAP_SHARED)
	if err != nil {
		_ = f.Close()
		return nil, nil, err
	}
	return &SharedMemory{Size: int64(size), File: f}, b, nil
}

// Map maps the shared memory for reading and closes its file, which the mapping does not need.
func (m *SharedMemory) Map() ([]byte, error) {
	defer m.Close()
	if m.File == nil {
		return nil, errors.New("remote: shared memory without its file")
	}
	if m.Size <= 0 || m.Size > 1<<40 {
		return nil, errors.New("remote: shared memory of an invalid size")
	}
	fi, err := m.File.Stat()
	if err != nil {
		return nil, err
	}
	if fi.Size() < m.Size {
		return nil, errors.New("remote: the shared memory's file is smaller than it says")
	}
	return syscall.Mmap(int(m.File.Fd()), 0, int(m.Size), syscall.PROT_READ, syscall.MAP_SHARED)
}

// Close lets go of the shared memory's file, which a mapping of it does not need.
func (m *SharedMemory) Close() error {
	if m.File == nil {
		return nil
	}
	err := m.File.Close()
	m.File = nil
	return err
}

// Unmap unmaps memory Map or NewSharedMemory mapped.
func Unmap(b []byte) error {
	return syscall.Munmap(b)
}
