// Copyright 2026 The ggfx Authors
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

package copycheck_test

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// copied are the packages under internal that are ggfx's, file for file.
var copied = []string{"cocoa", "microsoftgdk", "thread", "windowsystem", "winver"}

// differ are the files changed on purpose: thread.go reports a misuse without ggfx's debug
// package.
var differ = map[string]bool{
	"thread/thread.go": true,
}

// The copies import each other at ggwd's paths.
var (
	ggfxInternal = []byte("github.com/ironpark/ggfx/internal/")
	ggwdInternal = []byte("github.com/ironpark/ggwd/internal/")
)

func TestCopiesMatchGGFX(t *testing.T) {
	out, err := exec.Command("go", "list", "-m", "-f", "{{.Dir}}", "github.com/ironpark/ggfx").Output()
	if err != nil {
		t.Skipf("ggfx's module is not found: %v", err)
	}
	ggfx := filepath.Join(strings.TrimSpace(string(out)), "internal")
	ggwd := ".."

	for _, pkg := range copied {
		names := map[string]bool{}
		for _, dir := range []string{ggfx, ggwd} {
			entries, err := os.ReadDir(filepath.Join(dir, pkg))
			if err != nil {
				t.Fatal(err)
			}
			for _, e := range entries {
				if !e.IsDir() {
					names[e.Name()] = true
				}
			}
		}
		for name := range names {
			path := pkg + "/" + name
			if differ[path] {
				continue
			}
			a, errA := os.ReadFile(filepath.Join(ggfx, pkg, name))
			b, errB := os.ReadFile(filepath.Join(ggwd, pkg, name))
			switch {
			case errA != nil:
				t.Errorf("%s is ggwd's alone", path)
			case errB != nil:
				t.Errorf("%s is missing; ggfx has it", path)
			case !bytes.Equal(bytes.ReplaceAll(a, ggfxInternal, ggwdInternal), b):
				t.Errorf("%s differs from ggfx's; copy it over, or list it in differ", path)
			}
		}
	}
}
