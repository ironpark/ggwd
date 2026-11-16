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

import "testing"

// TestSharedMemoryLifetime maps the app's file mapping by its name, as a viewer does, and checks
// that the mapping is gone once both sides let go of it.
func TestSharedMemoryLifetime(t *testing.T) {
	shm, mem, err := NewSharedMemory(1 << 16)
	if err != nil {
		t.Fatal(err)
	}
	received := &SharedMemory{Size: shm.Size, Name: shm.Name}
	b, err := received.Map()
	if err != nil {
		t.Fatal(err)
	}
	copy(mem, "a batch")
	if string(b[:7]) != "a batch" {
		t.Errorf("the viewer's mapping reads %q", b[:7])
	}
	if received.handle != 0 {
		t.Error("Map kept its handle")
	}
	if _, err := (&SharedMemory{Size: shm.Size * 2, Name: shm.Name}).Map(); err == nil {
		t.Error("a mapping larger than the shared memory mapped")
	}
	if err := Unmap(mem); err != nil {
		t.Fatal(err)
	}
	if err := shm.Close(); err != nil {
		t.Fatal(err)
	}
	// The viewer's view alone keeps it.
	if string(b[:7]) != "a batch" {
		t.Errorf("the viewer's mapping reads %q once the app let go", b[:7])
	}
	if err := Unmap(b); err != nil {
		t.Fatal(err)
	}
	if _, err := (&SharedMemory{Size: shm.Size, Name: shm.Name}).Map(); err == nil {
		t.Error("the shared memory outlived both sides")
	}
}
