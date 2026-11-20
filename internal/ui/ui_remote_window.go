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

//go:build ggwd_remote && ((unix && !android && !ios) || windows) && !nintendosdk && !playstation5

package ui

import (
	"bytes"
	"image"
	"image/png"
	"sync"
	"sync/atomic"

	"github.com/ironpark/ggwd/remote"
)

// remoteWindow is the app's window, shown by the viewer. Its methods may be called from any
// goroutine and do nothing once closed.
type remoteWindow struct {
	r       *remoteUI
	context *eventContext
	handle  any
	closed  atomic.Bool

	// mu guards mutable window options and cursor/text-input state mirrored to the viewer.
	mu        sync.Mutex
	cursor    CursorShape
	mode      CursorMode
	options   remote.OpenWindow
	textInput remote.TextInput
}

var _ AppWindow = (*remoteWindow)(nil)

// Close closes the window: the app gets its ClosedEvent and the viewer a Closed, and the loop
// ends as on the desktop when the last window closes.
func (w *remoteWindow) Close() {
	if !w.closed.CompareAndSwap(false, true) {
		return
	}
	w.r.u.pushEvent(ClosedEvent{Window: w})
	_ = w.r.send(&remote.Closed{})
	w.r.wakeLoop()
}

func (w *remoteWindow) RequestFrame() {
	if w.closed.Load() {
		return
	}
	w.context.requestFrame()
	w.r.wakeLoop()
}

func (w *remoteWindow) SetTitle(title string) {
	if w.closed.Load() {
		return
	}
	w.mu.Lock()
	if w.options.Title == title {
		w.mu.Unlock()
		return
	}
	w.options.Title = title
	w.mu.Unlock()
	_ = w.r.send(&remote.SetTitle{Title: title})
}

// setOptions tells the viewer the window's options after f changed them, if it did.
func (w *remoteWindow) setOptions(f func(o *remote.OpenWindow)) {
	if w.closed.Load() {
		return
	}
	w.mu.Lock()
	o := w.options
	f(&o)
	if o == w.options {
		w.mu.Unlock()
		return
	}
	w.options = o
	w.mu.Unlock()
	_ = w.r.send(&o)
}

func (w *remoteWindow) SetSizeLimits(minw, minh, maxw, maxh int) {
	limit := func(v int) float64 {
		if v <= 0 {
			return -1
		}
		return float64(v)
	}
	w.setOptions(func(o *remote.OpenWindow) {
		o.MinWidth, o.MinHeight, o.MaxWidth, o.MaxHeight = limit(minw), limit(minh), limit(maxw), limit(maxh)
	})
}

func (w *remoteWindow) SetResizingMode(mode WindowResizingMode) {
	w.setOptions(func(o *remote.OpenWindow) { o.Resizable = mode == WindowResizingModeEnabled })
}

func (w *remoteWindow) SetDecorated(decorated bool) {
	w.setOptions(func(o *remote.OpenWindow) { o.Decorated = decorated })
}

func (w *remoteWindow) SetFloating(floating bool) {
	w.setOptions(func(o *remote.OpenWindow) { o.Floating = floating })
}

func (w *remoteWindow) SetTitlebarHidden(hidden bool) {
	w.setOptions(func(o *remote.OpenWindow) { o.TitlebarHidden = hidden })
}

func (w *remoteWindow) SetTrafficLights(x, y int, set bool) {
	w.setOptions(func(o *remote.OpenWindow) {
		o.TrafficLightsSet = set
		if set {
			o.TrafficLightsX, o.TrafficLightsY = float64(x), float64(y)
		} else {
			o.TrafficLightsX, o.TrafficLightsY = 0, 0
		}
	})
}

func (w *remoteWindow) SetMousePassthrough(enabled bool) {
	w.setOptions(func(o *remote.OpenWindow) { o.MousePassthrough = enabled })
}

func (w *remoteWindow) sendRequest(req uint8) {
	if w.closed.Load() {
		return
	}
	_ = w.r.send(&remote.WindowRequest{Request: req})
}

func (w *remoteWindow) StartDrag()           { w.sendRequest(remote.RequestStartDrag) }
func (w *remoteWindow) TitlebarDoubleClick() { w.sendRequest(remote.RequestTitlebarDoubleClick) }
func (w *remoteWindow) Maximize()            { w.sendRequest(remote.RequestMaximize) }
func (w *remoteWindow) Minimize()            { w.sendRequest(remote.RequestMinimize) }
func (w *remoteWindow) Restore()             { w.sendRequest(remote.RequestRestore) }
func (w *remoteWindow) Focus()               { w.sendRequest(remote.RequestFocus) }
func (w *remoteWindow) RequestAttention()    { w.sendRequest(remote.RequestAttention) }

func (w *remoteWindow) SetVisible(visible bool) {
	if visible {
		w.sendRequest(remote.RequestShow)
	} else {
		w.sendRequest(remote.RequestHide)
	}
}

func (w *remoteWindow) SetFullscreen(fullscreen bool) {
	if fullscreen {
		w.sendRequest(remote.RequestFullscreenEnter)
	} else {
		w.sendRequest(remote.RequestFullscreenExit)
	}
}

func (w *remoteWindow) SetSize(width, height int) {
	if w.closed.Load() {
		return
	}
	_ = w.r.send(&remote.SetSize{Width: float64(width), Height: float64(height)})
}

func (w *remoteWindow) SetPosition(x, y int) {
	if w.closed.Load() {
		return
	}
	_ = w.r.send(&remote.SetPosition{X: float64(x), Y: float64(y)})
}

func (w *remoteWindow) SetPositionOn(_ *Monitor, x, y int) {
	w.SetPosition(x, y)
}

func (w *remoteWindow) SetIcon(iconImages []image.Image) {
	if w.closed.Load() {
		return
	}
	var encoded [][]byte
	for _, img := range iconImages {
		if img == nil {
			continue
		}
		var buf bytes.Buffer
		if err := png.Encode(&buf, img); err == nil {
			encoded = append(encoded, buf.Bytes())
		}
	}
	_ = w.r.send(&remote.SetIcon{Icons: encoded})
}

// setTextInput tells the viewer the window's text input state after f changed it, if it did.
func (w *remoteWindow) setTextInput(f func(t *remote.TextInput)) {
	if w.closed.Load() {
		return
	}
	w.mu.Lock()
	t := w.textInput
	f(&t)
	if t == w.textInput {
		w.mu.Unlock()
		return
	}
	w.textInput = t
	w.mu.Unlock()
	_ = w.r.send(&t)
}

func (w *remoteWindow) SetTextInputEnabled(enabled bool) {
	w.setTextInput(func(t *remote.TextInput) { t.Enabled = enabled })
}

func (w *remoteWindow) SetTextInputContext(before, after string) {
	w.setTextInput(func(t *remote.TextInput) { t.Before, t.After = before, after })
}

func (w *remoteWindow) SetTextInputRect(x, y, width, height float64) {
	w.setTextInput(func(t *remote.TextInput) { t.X, t.Y, t.Width, t.Height = x, y, width, height })
}

// Size returns the viewer's window's size in device-independent pixels.
func (w *remoteWindow) Size() (int, int) {
	c := w.r.config()
	return int(c.Width), int(c.Height)
}

func (w *remoteWindow) Position() (int, int) {
	c := w.r.config()
	return int(c.X), int(c.Y)
}

func (w *remoteWindow) IsVisible() bool {
	c, have := w.r.configWithState()
	if !have {
		w.mu.Lock()
		hidden := w.options.Hidden
		w.mu.Unlock()
		return !hidden
	}
	return c.Visible
}

func (w *remoteWindow) IsMaximized() bool {
	return WindowState(w.r.config().State) == WindowStateMaximized
}

func (w *remoteWindow) IsMinimized() bool {
	return WindowState(w.r.config().State) == WindowStateMinimized
}

func (w *remoteWindow) IsFullscreen() bool {
	return WindowState(w.r.config().State) == WindowStateFullscreen
}

func (w *remoteWindow) IsFocused() bool { return w.r.config().Focused }

func (w *remoteWindow) DeviceScaleFactor() float64 { return remoteMonitor{w.r}.deviceScaleFactor() }

func (w *remoteWindow) Monitor() *Monitor { return w.r.monitor }

func (w *remoteWindow) CursorShape() CursorShape {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.cursor
}

func (w *remoteWindow) SetCursorShape(shape CursorShape) {
	if w.closed.Load() {
		return
	}
	w.mu.Lock()
	if w.cursor == shape {
		w.mu.Unlock()
		return
	}
	w.cursor = shape
	msg := &remote.SetCursor{Shape: uint8(w.cursor), Mode: uint8(w.mode)}
	w.mu.Unlock()
	_ = w.r.send(msg)
}

func (w *remoteWindow) CursorMode() CursorMode {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.mode
}

func (w *remoteWindow) SetCursorMode(mode CursorMode) {
	if w.closed.Load() {
		return
	}
	w.mu.Lock()
	if w.mode == mode {
		w.mu.Unlock()
		return
	}
	w.mode = mode
	msg := &remote.SetCursor{Shape: uint8(w.cursor), Mode: uint8(w.mode)}
	w.mu.Unlock()
	_ = w.r.send(msg)
}

func (w *remoteWindow) Handle() any          { return w.handle }
func (w *remoteWindow) SetHandle(handle any) { w.handle = handle }

// There is no local native window in remote mode, so no native handle and no local accessibility tree.
func (w *remoteWindow) NativeHandle() uintptr      { return 0 }
func (w *remoteWindow) AccessibilityView() uintptr { return 0 }
func (w *remoteWindow) SetAccessibilityHandlers(children func() uintptr, hitTest func(x, y float64) uintptr) {
}
func (w *remoteWindow) SetGetObjectHandler(func(wparam, lparam uintptr) (uintptr, bool)) {}
