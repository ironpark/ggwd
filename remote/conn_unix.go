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
	"encoding/binary"
	"errors"
	"fmt"
	"net"
	"os"
	"sync"
	"syscall"
	"time"
)

// Conn is one end of the protocol over a unix socket. Send may be called from several goroutines
// at once; Receive from one.
type Conn struct {
	c *net.UnixConn

	sendM sync.Mutex

	buf []byte // received and not yet decoded
	fds []int  // files received and not yet given to a SharedMemory
}

// NewConn returns the protocol's end of c.
func NewConn(c *net.UnixConn) *Conn {
	return &Conn{c: c}
}

// Dial connects to the viewer at path, trying again for DialTimeout while nothing listens there.
func Dial(path string) (*Conn, error) {
	deadline := time.Now().Add(DialTimeout)
	for {
		c, err := net.DialUnix("unix", nil, &net.UnixAddr{Name: path, Net: "unix"})
		if err == nil {
			return NewConn(c), nil
		}
		if time.Now().After(deadline) {
			return nil, err
		}
		time.Sleep(20 * time.Millisecond)
	}
}

// Send sends m. A SharedMemory's file goes with it.
func (c *Conn) Send(m Message) error {
	b := Encode(m)
	c.sendM.Lock()
	defer c.sendM.Unlock()
	if sm, ok := m.(*SharedMemory); ok {
		if sm.File == nil {
			return errors.New("remote: a SharedMemory without its file")
		}
		n, oobn, err := c.c.WriteMsgUnix(b, syscall.UnixRights(int(sm.File.Fd())), nil)
		if err != nil {
			return err
		}
		if oobn == 0 {
			return errors.New("remote: the shared memory's file did not go")
		}
		b = b[n:]
	}
	for len(b) > 0 {
		n, err := c.c.Write(b)
		if err != nil {
			return err
		}
		b = b[n:]
	}
	return nil
}

// Receive returns the next message.
func (c *Conn) Receive() (Message, error) {
	for {
		if len(c.buf) >= 4 {
			n := binary.LittleEndian.Uint32(c.buf)
			if n > MaxFrameSize {
				return nil, fmt.Errorf("remote: a frame of %d bytes", n)
			}
			if len(c.buf) >= 4+int(n) {
				m, err := Decode(c.buf[4 : 4+n])
				c.buf = c.buf[4+n:]
				if err != nil {
					return nil, err
				}
				if sm, ok := m.(*SharedMemory); ok {
					if len(c.fds) == 0 {
						return nil, errors.New("remote: a SharedMemory came without its file")
					}
					sm.File = os.NewFile(uintptr(c.fds[0]), "ggwd-remote-shm")
					c.fds = c.fds[1:]
				}
				return m, nil
			}
		}
		if err := c.read(); err != nil {
			return nil, err
		}
	}
}

func (c *Conn) read() error {
	var b [64 << 10]byte
	oob := make([]byte, syscall.CmsgSpace(4*4))
	n, oobn, _, _, err := c.c.ReadMsgUnix(b[:], oob)
	if err != nil {
		return err
	}
	if oobn > 0 {
		msgs, perr := syscall.ParseSocketControlMessage(oob[:oobn])
		if perr != nil {
			return perr
		}
		for _, msg := range msgs {
			fds, perr := syscall.ParseUnixRights(&msg)
			if perr != nil {
				return perr
			}
			c.fds = append(c.fds, fds...)
		}
	}
	// Compact what is left of the buffer before it grows.
	if len(c.buf) == 0 {
		c.buf = c.buf[:0:0]
	}
	c.buf = append(c.buf, b[:n]...)
	if n == 0 {
		return errors.New("remote: the connection closed")
	}
	return nil
}

// Close closes the connection, and the files received that no message took.
func (c *Conn) Close() error {
	for _, fd := range c.fds {
		_ = syscall.Close(fd)
	}
	c.fds = nil
	return c.c.Close()
}
