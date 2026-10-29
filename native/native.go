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

// Package native reaches the platform's side of a ggwd window: its native handle, and the hooks
// a screen reader goes through on macOS and Windows. What it returns and takes is the
// platform's, and differs from one to another; an app that only draws and handles events does
// not need it.
package native

import (
	"github.com/ironpark/ggwd"
	"github.com/ironpark/ggwd/internal/windowbridge"
)

// Handle returns the platform's window object: an NSWindow on macOS, an HWND on Windows, an X11
// window on Linux. It runs on the main thread.
func Handle(w *ggwd.Window) uintptr {
	return windowbridge.UI(w).NativeHandle()
}

// AccessibilityView returns a mouse-transparent NSView owned by w on macOS, or zero elsewhere.
// The view is valid until the window closes.
func AccessibilityView(w *ggwd.Window) uintptr {
	return windowbridge.UI(w).AccessibilityView()
}

// SetAccessibilityHandlers supplies the macOS accessibility container's children (an
// autoreleased NSArray) and hit test (an NSAccessibilityElement). Hit test coordinates are native
// screen points. Callbacks run on the main thread and must not call window methods or wait for
// the event handler. Nil detaches them. Other platforms ignore these callbacks.
func SetAccessibilityHandlers(w *ggwd.Window, children func() uintptr, hitTest func(x, y float64) uintptr) {
	windowbridge.UI(w).SetAccessibilityHandlers(children, hitTest)
}

// SetGetObjectHandler handles Windows WM_GETOBJECT. The callback runs on the main thread and must
// not call window methods or wait for the event handler. Return false to use the default window
// procedure; nil removes the handler. Other platforms ignore it.
func SetGetObjectHandler(w *ggwd.Window, f func(wparam, lparam uintptr) (uintptr, bool)) {
	windowbridge.UI(w).SetGetObjectHandler(f)
}
