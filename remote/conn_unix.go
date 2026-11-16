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
	"os"
	"syscall"
)

// sendSharedMemory sends the start of sm's frame b with sm's file as an SCM_RIGHTS control
// message, and returns the rest of b.
func (c *Conn) sendSharedMemory(b []byte, sm *SharedMemory) ([]byte, error) {
	if sm.File == nil {
		return nil, errors.New("remote: a SharedMemory without its file")
	}
	n, oobn, err := c.c.WriteMsgUnix(b, syscall.UnixRights(int(sm.File.Fd())), nil)
	if err != nil {
		return nil, err
	}
	if oobn == 0 {
		return nil, errors.New("remote: the shared memory's file did not go")
	}
	return b[n:], nil
}

// receivedSharedMemory gives sm the file that came with it.
func (c *Conn) receivedSharedMemory(sm *SharedMemory) error {
	if len(c.fds) == 0 {
		return errors.New("remote: a SharedMemory came without its file")
	}
	sm.File = os.NewFile(uintptr(c.fds[0]), "ggwd-remote-shm")
	c.fds = c.fds[1:]
	return nil
}

// readMsg reads into b, keeping the files that come with it.
func (c *Conn) readMsg(b []byte) (int, error) {
	oob := make([]byte, syscall.CmsgSpace(4*4))
	n, oobn, _, _, err := c.c.ReadMsgUnix(b, oob)
	if err != nil {
		return 0, err
	}
	if oobn > 0 {
		msgs, err := syscall.ParseSocketControlMessage(oob[:oobn])
		if err != nil {
			return 0, err
		}
		for _, msg := range msgs {
			fds, err := syscall.ParseUnixRights(&msg)
			if err != nil {
				return 0, err
			}
			c.fds = append(c.fds, fds...)
		}
	}
	return n, nil
}

func (c *Conn) closeFiles() {
	for _, fd := range c.fds {
		_ = syscall.Close(fd)
	}
	c.fds = nil
}
