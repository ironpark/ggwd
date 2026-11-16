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

// Package viewer has what a viewer, a ggwd app that shows another app's window, needs of ggwd
// to speak the protocol of package remote: the translation of its window's events. It is apart
// from package remote, which the app's side imports too, because it needs package ggwd.
package viewer

import (
	"github.com/ironpark/ggwd"
	"github.com/ironpark/ggwd/remote"
)

// Mods returns the protocol's modifiers of m.
func Mods(m ggwd.Mods) remote.Mods {
	var r remote.Mods
	if m.Shift {
		r |= remote.ModShift
	}
	if m.Ctrl {
		r |= remote.ModCtrl
	}
	if m.Alt {
		r |= remote.ModAlt
	}
	if m.Meta {
		r |= remote.ModMeta
	}
	return r
}

// Input returns the Input that forwards ev, an event of the viewer's window, and true; or false
// for an event that is not input. The caller numbers it: Seq is left 0.
func Input(ev ggwd.Event) (remote.Input, bool) {
	switch ev := ev.(type) {
	case ggwd.MouseMoveEvent:
		return remote.Input{Kind: remote.InputMouseMove, X: ev.X, Y: ev.Y, Mods: Mods(ev.Modifiers)}, true
	case ggwd.MouseButtonEvent:
		return remote.Input{Kind: remote.InputMouseButton, X: ev.X, Y: ev.Y, Button: int(ev.Button), Pressed: ev.Pressed, Mods: Mods(ev.Modifiers)}, true
	case ggwd.MouseEnterEvent:
		return remote.Input{Kind: remote.InputMouseEnter, Entered: ev.Entered}, true
	case ggwd.ScrollEvent:
		return remote.Input{Kind: remote.InputScroll, X: ev.X, Y: ev.Y, Line: ev.Line, Mods: Mods(ev.Modifiers), Phase: int(ev.Phase), Precise: ev.Precise}, true
	case ggwd.KeyEvent:
		return remote.Input{Kind: remote.InputKey, Key: int(ev.Key), Pressed: ev.Pressed, Repeat: ev.Repeat, Mods: Mods(ev.Modifiers)}, true
	case ggwd.TextEvent:
		return remote.Input{Kind: remote.InputText, Text: ev.Text}, true
	}
	return remote.Input{}, false
}
