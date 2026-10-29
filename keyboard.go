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

// Keyboard tracks which keys are held, from the presses and releases a window reports. The zero
// value holds no key. A Keyboard is not concurrent-safe.
type Keyboard struct {
	held [KeyMax + 1]bool
}

// Set records a press or a release of key. A key outside the defined ones is ignored.
func (k *Keyboard) Set(key Key, pressed bool) {
	if key >= 0 && key <= KeyMax {
		k.held[key] = pressed
	}
}

// IsPressed reports whether key is held.
func (k *Keyboard) IsPressed(key Key) bool {
	return key >= 0 && key <= KeyMax && k.held[key]
}

// Mods reports the modifier keys held, on either side.
func (k *Keyboard) Mods() Mods {
	return Mods{
		Shift: k.held[KeyShift] || k.held[KeyShiftLeft] || k.held[KeyShiftRight],
		Ctrl:  k.held[KeyControl] || k.held[KeyControlLeft] || k.held[KeyControlRight],
		Alt:   k.held[KeyAlt] || k.held[KeyAltLeft] || k.held[KeyAltRight],
		Meta:  k.held[KeyMeta] || k.held[KeyMetaLeft] || k.held[KeyMetaRight],
	}
}

// Reset forgets every held key. Call it when the window loses the focus: a window system does not
// report the releases that happen while another window has it.
func (k *Keyboard) Reset() {
	k.held = [KeyMax + 1]bool{}
}
