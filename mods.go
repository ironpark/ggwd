// Copyright 2026 The ggip Authors
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

package ggwd

import "runtime"

// Mods are the modifier keys held at an input event, on either side. An event carries them even
// if a modifier is released before the next frame or the platform sent no key event of its own
// for it.
type Mods struct {
	Shift, Ctrl, Alt, Meta bool
}

// Or is the union of two modifier sets.
func (m Mods) Or(o Mods) Mods {
	return Mods{Shift: m.Shift || o.Shift, Ctrl: m.Ctrl || o.Ctrl, Alt: m.Alt || o.Alt, Meta: m.Meta || o.Meta}
}

// Cmd reports the platform's command modifier: Meta (⌘) on macOS, Ctrl elsewhere. Shortcuts such
// as select-all and paste check it.
func (m Mods) Cmd() bool {
	if isDarwin {
		return m.Meta
	}
	return m.Ctrl
}

// isDarwin reports whether the command modifier is Meta. A browser on a Mac is not darwin, and
// gets Ctrl.
const isDarwin = runtime.GOOS == "darwin"
