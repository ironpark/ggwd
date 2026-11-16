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

//go:build ggwd_remote && unix && !android && !ios && !nintendosdk && !playstation5

package ui

import (
	"image"
	"sync/atomic"

	"github.com/ironpark/ggwd/remote"
)

// remoteWindow is the app's window, shown by the viewer.
type remoteWindow struct {
	r       *remoteUI
	context *eventContext
	handle  any
	title   string
	cursor  CursorShape
	mode    CursorMode
	closed  atomic.Bool
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
	w.context.requestFrame()
	w.r.wakeLoop()
}

func (w *remoteWindow) SetTitle(title string) {
	if w.title == title {
		return
	}
	w.title = title
	_ = w.r.send(&remote.SetTitle{Title: title})
}

// Size returns the viewer's window's size in device-independent pixels.
func (w *remoteWindow) Size() (int, int) {
	c := w.r.config()
	return int(c.Width), int(c.Height)
}

func (w *remoteWindow) IsFocused() bool { return w.r.config().Focused }

func (w *remoteWindow) DeviceScaleFactor() float64 { return remoteMonitor{w.r}.deviceScaleFactor() }

func (w *remoteWindow) Monitor() *Monitor { return w.r.monitor }

func (w *remoteWindow) CursorShape() CursorShape { return w.cursor }

func (w *remoteWindow) SetCursorShape(shape CursorShape) {
	if w.cursor == shape {
		return
	}
	w.cursor = shape
	_ = w.r.send(&remote.SetCursor{Shape: uint8(w.cursor), Mode: uint8(w.mode)})
}

func (w *remoteWindow) CursorMode() CursorMode { return w.mode }

func (w *remoteWindow) SetCursorMode(mode CursorMode) {
	if w.mode == mode {
		return
	}
	w.mode = mode
	_ = w.r.send(&remote.SetCursor{Shape: uint8(w.cursor), Mode: uint8(w.mode)})
}

func (w *remoteWindow) Handle() any          { return w.handle }
func (w *remoteWindow) SetHandle(handle any) { w.handle = handle }

// What the remote backend cannot do yet, all in one place. The viewer owns the window: its
// placement, size, state and decoration are the viewer's and its user's, and the protocol carries
// none of these requests yet. The getters report the window as the viewer shows it: at the
// monitor's origin, visible, neither maximized, minimized nor fullscreen. There is no native
// window, so no native handle, no accessibility tree, and no text input context: text comes as
// TextEvents, without composition.

func (w *remoteWindow) SetSize(width, height int)                    {}
func (w *remoteWindow) Position() (int, int)                         { return 0, 0 }
func (w *remoteWindow) SetPosition(x, y int)                         {}
func (w *remoteWindow) SetSizeLimits(minw, minh, maxw, maxh int)     {}
func (w *remoteWindow) SetResizingMode(mode WindowResizingMode)      {}
func (w *remoteWindow) IsVisible() bool                              { return true }
func (w *remoteWindow) SetVisible(visible bool)                      {}
func (w *remoteWindow) SetDecorated(decorated bool)                  {}
func (w *remoteWindow) SetFloating(floating bool)                    {}
func (w *remoteWindow) StartDrag()                                   {}
func (w *remoteWindow) TitlebarDoubleClick()                         {}
func (w *remoteWindow) IsMaximized() bool                            { return false }
func (w *remoteWindow) Maximize()                                    {}
func (w *remoteWindow) IsMinimized() bool                            { return false }
func (w *remoteWindow) Minimize()                                    {}
func (w *remoteWindow) Restore()                                     {}
func (w *remoteWindow) SetIcon(iconImages []image.Image)             {}
func (w *remoteWindow) SetMousePassthrough(enabled bool)             {}
func (w *remoteWindow) RequestAttention()                            {}
func (w *remoteWindow) Focus()                                       {}
func (w *remoteWindow) IsFullscreen() bool                           { return false }
func (w *remoteWindow) SetFullscreen(fullscreen bool)                {}
func (w *remoteWindow) NativeHandle() uintptr                        { return 0 }
func (w *remoteWindow) SetTextInputEnabled(bool)                     {}
func (w *remoteWindow) SetTextInputContext(before, after string)     {}
func (w *remoteWindow) SetTextInputRect(x, y, width, height float64) {}
func (w *remoteWindow) AccessibilityView() uintptr                   { return 0 }
func (w *remoteWindow) SetAccessibilityHandlers(children func() uintptr, hitTest func(x, y float64) uintptr) {
}
func (w *remoteWindow) SetGetObjectHandler(func(wparam, lparam uintptr) (uintptr, bool)) {}
