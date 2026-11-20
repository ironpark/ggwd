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

//go:build (unix && !ios && !android) || windows

package remote

import (
	"bytes"
	"net"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

// pair returns the two ends of a real unix socket.
func pair(t *testing.T) (*Conn, *Conn) {
	t.Helper()
	dir, err := os.MkdirTemp("", "ggwd")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.RemoveAll(dir) })
	ln, err := net.ListenUnix("unix", &net.UnixAddr{Name: filepath.Join(dir, "s"), Net: "unix"})
	if err != nil {
		t.Fatal(err)
	}
	defer ln.Close()
	done := make(chan *net.UnixConn)
	go func() {
		c, _ := ln.AcceptUnix()
		done <- c
	}()
	a, err := Dial(filepath.Join(dir, "s"))
	if err != nil {
		t.Fatal(err)
	}
	b := NewConn(<-done)
	t.Cleanup(func() { a.Close(); b.Close() })
	return a, b
}

// TestConn sends every message and a shared memory file across a socket.
func TestConn(t *testing.T) {
	app, viewer := pair(t)

	shm, mem, err := NewSharedMemory(1 << 16)
	if err != nil {
		t.Fatal(err)
	}
	defer Unmap(mem)
	defer shm.Close()
	copy(mem, "a batch")

	sent := make(chan struct{})
	defer func() { <-sent }()
	go func() {
		defer close(sent)
		for _, m := range messages {
			if _, ok := m.(*SharedMemory); ok {
				m = shm
			}
			if err := app.Send(m); err != nil {
				t.Error(err)
				return
			}
		}
	}()
	for _, want := range messages {
		got, err := viewer.Receive()
		if err != nil {
			t.Fatal(err)
		}
		if sm, ok := got.(*SharedMemory); ok {
			b, err := sm.Map()
			if err != nil {
				t.Fatal(err)
			}
			if !bytes.HasPrefix(b, []byte("a batch")) {
				t.Errorf("the shared memory reads %q", b[:7])
			}
			Unmap(b)
			continue
		}
		if p, ok := want.(*Pixels); ok {
			w := *p
			w.Pixels = [][]byte{{1, 2}, nil, {3}}
			want = &w
		}
		if !reflect.DeepEqual(got, want) {
			t.Errorf("got %+v, want %+v", got, want)
		}
	}
}
