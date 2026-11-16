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

//go:build !android

package remote

import (
	"os"

	"golang.org/x/sys/unix"
)

// anonymousFile returns a memfd: memory with a file descriptor and no name in any file system, so
// that a small or noexec /tmp does not matter.
func anonymousFile() (*os.File, error) {
	fd, err := unix.MemfdCreate("ggwd-remote", unix.MFD_CLOEXEC)
	if err != nil {
		return anonymousTempFile()
	}
	return os.NewFile(uintptr(fd), "ggwd-remote"), nil
}
