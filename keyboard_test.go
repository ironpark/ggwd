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

package ggwd_test

import (
	"testing"

	"github.com/ironpark/ggwd"
)

func TestKeyboardModsFromEitherSide(t *testing.T) {
	var kb ggwd.Keyboard
	kb.Set(ggwd.KeyShiftRight, true)
	kb.Set(ggwd.KeyControlLeft, true)
	kb.Set(ggwd.KeyMeta, true)
	if got, want := kb.Mods(), (ggwd.Mods{Shift: true, Ctrl: true, Meta: true}); got != want {
		t.Errorf("Mods() = %+v; want %+v", got, want)
	}
	kb.Set(ggwd.KeyShiftRight, false)
	if kb.Mods().Shift {
		t.Error("Shift is held after its only key was released")
	}
	if !kb.IsPressed(ggwd.KeyControlLeft) || kb.IsPressed(ggwd.KeyControlRight) {
		t.Error("IsPressed does not tell the two Control keys apart")
	}
	kb.Reset()
	if kb.Mods() != (ggwd.Mods{}) || kb.IsPressed(ggwd.KeyControlLeft) {
		t.Error("Reset left a key held")
	}
}

func TestKeyboardIgnoresUndefinedKeys(t *testing.T) {
	var kb ggwd.Keyboard
	kb.Set(ggwd.KeyMax+1, true)
	kb.Set(-1, true)
	if kb.IsPressed(ggwd.KeyMax+1) || kb.IsPressed(-1) {
		t.Error("an undefined key reads as held")
	}
}

func TestModsOr(t *testing.T) {
	got := ggwd.Mods{Shift: true}.Or(ggwd.Mods{Alt: true})
	if want := (ggwd.Mods{Shift: true, Alt: true}); got != want {
		t.Errorf("Or = %+v; want %+v", got, want)
	}
}

func TestKeyboardSyncMods(t *testing.T) {
	var kb ggwd.Keyboard
	kb.Set(ggwd.KeyShiftLeft, true)
	kb.Set(ggwd.KeyMetaRight, true)
	kb.SyncMods(ggwd.Mods{Meta: true})
	if got, want := kb.Mods(), (ggwd.Mods{Meta: true}); got != want {
		t.Errorf("Mods() after SyncMods = %+v; want %+v", got, want)
	}
}
