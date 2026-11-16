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
// to speak the protocol of package remote: the window options, the Config and the Input of its
// window, and the app's requests applied to it. It is apart from package remote, which the app's
// side imports too, because it needs package ggwd.
//
// A viewer replays an app's batches inside frames of its own (see ggfx's package remote) and
// acknowledges each, and the app draws no further than two batches ahead of the acks. A window
// that is hidden, minimized or occluded gets no FrameEvents, so a viewer that replayed only in
// those would stall its app; it replays in a ggwd.BackgroundFrameEvent, which
// ggwd.RequestBackgroundFrame asks for, whatever its window is doing, and shows the result in its
// next window frame.
package viewer

import (
	"io/fs"
	"math"

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
		in := remote.Input{Kind: remote.InputText, Text: ev.Text, Passthrough: ev.Passthrough}
		if r := ev.Replacement; r != nil {
			in.HasReplacement, in.Start, in.End = true, r.Start, r.End
		}
		return in, true
	case ggwd.CompositionEvent:
		return remote.Input{Kind: remote.InputComposition, Text: ev.Text, Start: ev.Start, End: ev.End, Done: ev.Done}, true
	case ggwd.DragEvent:
		return remote.Input{Kind: remote.InputDrag, Phase: int(ev.Phase), X: ev.X, Y: ev.Y, Paths: Paths(ev.Files)}, true
	case ggwd.DropEvent:
		return remote.Input{Kind: remote.InputDrop, Paths: Paths(ev.Files)}, true
	}
	return remote.Input{}, false
}

// Paths returns the real paths of the files at the root of a drag's or a drop's file system:
// what the app, on the same machine, opens them by.
func Paths(files fs.FS) []string {
	if files == nil {
		return nil
	}
	entries, err := fs.ReadDir(files, ".")
	if err != nil {
		return nil
	}
	var paths []string
	for _, e := range entries {
		if p, ok := e.(ggwd.AbsPather); ok {
			paths = append(paths, p.AbsPath())
		}
	}
	return paths
}

// WindowOptions returns the options of a window that shows the app's window o. A negative size
// limit in o is none, as 0 is in ggwd.
func WindowOptions(o *remote.OpenWindow) *ggwd.WindowOptions {
	limit := func(v float64) float64 { return max(v, 0) }
	return &ggwd.WindowOptions{
		Title:          o.Title,
		Width:          o.Width,
		Height:         o.Height,
		Resizable:      o.Resizable,
		Undecorated:    !o.Decorated,
		Floating:       o.Floating,
		Transparent:    o.Transparent,
		TitlebarHidden: o.TitlebarHidden,
		MinWidth:       limit(o.MinWidth),
		MinHeight:      limit(o.MinHeight),
		MaxWidth:       limit(o.MaxWidth),
		MaxHeight:      limit(o.MaxHeight),
	}
}

// Config returns the Config of w, whose client area is width by height device-independent pixels
// at scale, as a ResizeEvent reports it.
func Config(w *ggwd.Window, width, height, scale float64, focused bool) remote.Config {
	c := remote.Config{
		Width:       width,
		Height:      height,
		PixelWidth:  int(math.Round(width * scale)),
		PixelHeight: int(math.Round(height * scale)),
		Scale:       scale,
		Focused:     focused,
		ColorMode:   uint8(ggwd.SystemColorMode()),
	}
	if m := w.Monitor(); m != nil {
		c.MonitorWidth, c.MonitorHeight = m.Size()
		c.RefreshRate = m.RefreshRate()
	}
	return c
}

// SetCursor applies the app's cursor to w.
func SetCursor(w *ggwd.Window, c *remote.SetCursor) {
	w.SetCursorShape(ggwd.CursorShape(c.Shape))
	w.SetCursorMode(ggwd.CursorMode(c.Mode))
}

// SetTextInput applies the app's text input state to w: the platform's input methods then
// compose into w where the app's caret is, and their CompositionEvents and TextEvents go to the
// app as Input.
func SetTextInput(w *ggwd.Window, t *remote.TextInput) {
	if t.Enabled {
		w.SetTextInputRect(t.X, t.Y, t.Width, t.Height)
		w.SetTextInputContext(t.Before, t.After)
	}
	w.SetTextInputEnabled(t.Enabled)
}

// Do does the app's request to w.
func Do(w *ggwd.Window, r *remote.WindowRequest) {
	switch r.Request {
	case remote.RequestStartDrag:
		w.StartDrag()
	case remote.RequestTitlebarDoubleClick:
		w.TitlebarDoubleClick()
	}
}
