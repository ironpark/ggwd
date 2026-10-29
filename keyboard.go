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
	var held [4]bool
	for i, keys := range modKeys {
		for _, key := range keys {
			held[i] = held[i] || k.held[key]
		}
	}
	return Mods{Shift: held[0], Ctrl: held[1], Alt: held[2], Meta: held[3]}
}

// SyncMods forgets the modifier keys m says are not held. A modifier's release can go unreported
// with the focus kept, taken by a system shortcut such as ⌘⇧4 or an input source switch, and the
// key would read as held until pressed again; the modifiers an event carries are the window
// system's own word on what is held. Call it with an event's modifiers before recording its key.
func (k *Keyboard) SyncMods(m Mods) {
	held := [4]bool{m.Shift, m.Ctrl, m.Alt, m.Meta}
	for i, keys := range modKeys {
		if !held[i] {
			for _, key := range keys {
				k.held[key] = false
			}
		}
	}
}

// modKeys are the keys of each modifier: Shift, Ctrl, Alt and Meta, as either side's and each side's.
var modKeys = [4][3]Key{
	{KeyShift, KeyShiftLeft, KeyShiftRight},
	{KeyControl, KeyControlLeft, KeyControlRight},
	{KeyAlt, KeyAltLeft, KeyAltRight},
	{KeyMeta, KeyMetaLeft, KeyMetaRight},
}

// Reset forgets every held key. Call it when the window loses the focus: a window system does not
// report the releases that happen while another window has it.
func (k *Keyboard) Reset() {
	k.held = [KeyMax + 1]bool{}
}
