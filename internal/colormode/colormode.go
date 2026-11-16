// Copyright 2026 The Ebitengine Authors
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

package colormode

import "sync/atomic"

type ColorMode int

const (
	Unknown ColorMode = iota
	Light
	Dark
)

// override is the color mode SystemColorMode reports in place of the system's, when set.
var override atomic.Pointer[func() ColorMode]

// SetOverride makes SystemColorMode report f's color mode instead of asking the system. An app
// whose window another process shows reports that process's: asking the system itself would
// create the app object on macOS, which an app with no window of its own must not.
func SetOverride(f func() ColorMode) {
	override.Store(&f)
}

func SystemColorMode() ColorMode {
	if f := override.Load(); f != nil {
		return (*f)()
	}
	return systemColorMode()
}
