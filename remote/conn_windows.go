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

import "errors"

// Windows passes no files over a unix socket: a SharedMemory names its file mapping instead.

func (c *Conn) sendSharedMemory(b []byte, sm *SharedMemory) ([]byte, error) {
	if sm.Name == "" {
		return nil, errors.New("remote: a SharedMemory without its name")
	}
	return b, nil
}

func (c *Conn) receivedSharedMemory(sm *SharedMemory) error {
	if sm.Name == "" {
		return errors.New("remote: a SharedMemory came without its name")
	}
	return nil
}

func (c *Conn) readMsg(b []byte) (int, error) {
	return c.c.Read(b)
}

func (c *Conn) closeFiles() {}
