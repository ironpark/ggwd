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

//go:build !android && !ios && !js && !nintendosdk && !playstation5

package ui

import (
	"errors"
	"image"
	"runtime"

	"github.com/ironpark/ggwd/internal/glfw"
	"github.com/ironpark/ggwd/internal/windowsystem"
)

// appWindow is an AppWindow on GLFW: what the app can do with one of its windows. Every method
// runs on the main thread, and does nothing once the window is closed.
type appWindow struct {
	backend *glfwBackend
	context *eventContext
	handle  any
}

var _ AppWindow = (*appWindow)(nil)

// RunApp runs the app until it returns RegularTermination, an error, or closes its last window.
func (u *UserInterface) RunApp(app App, options *RunOptions) error {
	if !windowsystem.Available() {
		return errors.New("ui: no window system is available")
	}
	u.startApp(app, options)
	return u.runLoop(options, func() error {
		return u.initGraphicsOnMainThread(options)
	}, func() error {
		if err := app.HandleEvent(StartEvent{}); err != nil {
			return err
		}
		var n int
		u.mainThread.Call(func() {
			n = len(u.windows)
		})
		if n == 0 {
			return RegularTermination
		}
		return nil
	})
}

// NewWindow creates a window for the app with the given handle attached. It can be called only
// while RunApp runs, from the goroutine handling the events.
func (u *UserInterface) NewWindow(o *WindowOptions, handle any) (AppWindow, error) {
	if u.app == nil || !u.isRunning() {
		return nil, errors.New("ui: NewWindow can be called only while RunApp runs")
	}
	if u.isTerminated() {
		return nil, errors.New("ui: NewWindow cannot be called after the app stopped")
	}

	settings := newWindowSettings(o)
	b := newGLFWBackend(u, settings)
	w := &appWindow{
		backend: b,
		handle:  handle,
	}
	w.context = newEventContext(u.app, w)
	b.context = w.context

	var err error
	u.runOnMainThread(func() {
		if u.isTerminated() {
			err = errors.New("ui: NewWindow cannot be called after the app stopped")
			return
		}
		if !o.PositionSet {
			if m := u.getInitMonitor(); m != nil {
				sw, sh := m.sizeInDIP()
				ww, wh := settings.getInitWindowSizeInDIP()
				x, y := InitialWindowPosition(int(sw), int(sh), ww, wh)
				settings.setInitWindowPositionInDIP(x, y)
			}
		}
		if err = b.createWindowOnMainThread(); err != nil {
			return
		}
		if err = b.takeSnapshot(); err != nil {
			return
		}
		u.windows = append(u.windows, b)

		// Tell the app the initial size, so that it has one path for sizes, unless creating the
		// window already did.
		s := 1.0
		if m, e := b.currentMonitor(); e == nil && m != nil {
			s = m.DeviceScaleFactor()
		}
		if b.reportedWidth == 0 && b.reportedHeight == 0 {
			b.reportedWidth, b.reportedHeight, b.reportedScale = b.windowWidthInDIP, b.windowHeightInDIP, s
			u.pushEvent(ResizeEvent{Window: w, Width: float64(b.windowWidthInDIP), Height: float64(b.windowHeightInDIP), Scale: s})
		}
		b.reportPosition()
	})
	if err != nil {
		return nil, err
	}
	return w, nil
}

func (w *appWindow) Close() {
	w.backend.call(func() error {
		return w.backend.window.SetShouldClose(true)
	})
	w.backend.ui.wakeLoop()
}

func (w *appWindow) RequestFrame() {
	w.context.requestFrame()
	w.backend.ui.wakeLoop()
}

func (w *appWindow) SetTitle(title string) {
	b := w.backend
	if b.settings.title.Swap(title) == title {
		return
	}
	b.call(func() error {
		return b.setWindowTitle(b.settings.title.Load().(string))
	})
}

func (w *appWindow) Size() (int, int) {
	s := w.backend.state()
	return s.width, s.height
}

func (w *appWindow) SetSize(width, height int) {
	b := w.backend
	b.call(func() error {
		m, err := b.isWindowMaximized()
		if err != nil {
			return err
		}
		if m && runtime.GOOS != "darwin" {
			return nil
		}
		return b.setWindowSizeInDIP(width, height, true)
	})
}

func (w *appWindow) Position() (int, int) {
	s := w.backend.state()
	return s.x, s.y
}

func (w *appWindow) SetPosition(x, y int) {
	b := w.backend
	b.call(func() error {
		m, err := b.currentMonitor()
		if err != nil {
			return err
		}
		return b.setWindowPositionInDIP(x, y, m, true)
	})
}

func (w *appWindow) SetSizeLimits(minw, minh, maxw, maxh int) {
	b := w.backend
	if !b.settings.setWindowSizeLimitsInDIP(minw, minh, maxw, maxh) {
		return
	}
	b.call(b.updateWindowSizeLimits)
}

func (w *appWindow) SetResizingMode(mode WindowResizingMode) {
	b := w.backend
	if WindowResizingMode(b.settings.windowResizingMode.Swap(int32(mode))) == mode {
		return
	}
	b.call(func() error {
		return b.setWindowResizingMode(b.settings.resizingMode())
	})
}

func (w *appWindow) IsVisible() bool {
	return w.backend.state().visible
}

func (w *appWindow) SetVisible(visible bool) {
	b := w.backend
	b.call(func() error {
		return b.setWindowVisible(visible)
	})
}

func (w *appWindow) SetDecorated(decorated bool) {
	b := w.backend
	// On macOS the decoration is applied once more after the first frame (#2600); let that apply
	// this one.
	b.settings.initWindowDecorated.Store(decorated)
	b.call(func() error {
		return b.setWindowDecorated(decorated)
	})
}

func (w *appWindow) SetFloating(floating bool) {
	b := w.backend
	b.call(func() error {
		return b.setWindowFloating(floating)
	})
}

func (w *appWindow) IsMaximized() bool {
	b := w.backend
	if b.settings.resizingMode() != WindowResizingModeEnabled {
		return false
	}
	return b.state().maximized
}

func (w *appWindow) Maximize() {
	b := w.backend
	// Do not allow maximizing the window when the window is not resizable.
	// On Windows, it is possible to restore the window from being maximized by mouse-dragging,
	// and this can be an unexpected behavior (#1990).
	if b.settings.resizingMode() != WindowResizingModeEnabled || !b.settings.isWindowMaximizable() {
		return
	}
	b.call(b.maximizeWindow)
}

func (w *appWindow) IsMinimized() bool {
	return w.backend.state().minimized
}

func (w *appWindow) Minimize() {
	w.backend.call(w.backend.iconifyWindow)
}

func (w *appWindow) Restore() {
	w.backend.call(w.backend.restoreWindow)
}

func (w *appWindow) IsFullscreen() bool {
	return w.backend.state().fullscreen
}

func (w *appWindow) SetFullscreen(fullscreen bool) {
	w.backend.call(func() error {
		return w.backend.setFullscreen(fullscreen)
	})
}

func (w *appWindow) IsFocused() bool {
	return w.backend.state().focused
}

func (w *appWindow) Focus() {
	w.backend.call(w.backend.window.Focus)
}

func (w *appWindow) RequestAttention() {
	w.backend.call(w.backend.window.RequestAttention)
}

func (w *appWindow) SetIcon(iconImages []image.Image) {
	w.backend.settings.setIconImages(iconImages)
	if err := w.backend.updateIconIfNeeded(); err != nil {
		w.backend.ui.setError(err)
	}
}

func (w *appWindow) SetMousePassthrough(enabled bool) {
	w.backend.call(func() error {
		return w.backend.setWindowMousePassthrough(enabled)
	})
}

func (w *appWindow) CursorShape() CursorShape {
	return CursorShape(w.backend.cursorShape.Load())
}

func (w *appWindow) SetCursorShape(shape CursorShape) {
	b := w.backend
	if CursorShape(b.cursorShape.Swap(int32(shape))) == shape {
		return
	}
	b.call(func() error {
		return b.window.SetCursor(glfwSystemCursors[CursorShape(b.cursorShape.Load())])
	})
}

func (w *appWindow) CursorMode() CursorMode {
	return w.backend.state().cursorMode
}

func (w *appWindow) SetCursorMode(mode CursorMode) {
	b := w.backend
	b.call(func() error {
		if err := b.window.SetInputMode(glfw.CursorMode, driverCursorModeToGLFWCursorMode(mode)); err != nil {
			return err
		}
		if mode == CursorModeVisible {
			return b.window.SetCursor(glfwSystemCursors[CursorShape(b.cursorShape.Load())])
		}
		return nil
	})
}

func (w *appWindow) DeviceScaleFactor() float64 {
	m := w.Monitor()
	if m == nil {
		return 1
	}
	return m.DeviceScaleFactor()
}

func (w *appWindow) Monitor() *Monitor {
	return w.backend.state().monitor
}

func (w *appWindow) NativeHandle() uintptr {
	return w.backend.state().nativeHandle
}

func (w *appWindow) Handle() any {
	return w.handle
}

func (w *appWindow) SetHandle(handle any) {
	w.handle = handle
}

// call runs f on the main thread, unless the window is closed or the UI has stopped, and then
// takes a snapshot of what f changed. An error f returns ends the loop.
func (u *glfwBackend) call(f func() error) {
	if u.ui.isTerminated() {
		return
	}
	u.ui.runOnMainThread(func() {
		if u.ui.isTerminated() || u.closed {
			return
		}
		if err := f(); err != nil {
			u.ui.setError(err)
			return
		}
		if err := u.takeSnapshot(); err != nil {
			u.ui.setError(err)
		}
	})
}

// windowSnapshot is a window's state that the getters read. Its zero value is a closed window's.
type windowSnapshot struct {
	width, height int
	x, y          int
	visible       bool
	focused       bool
	minimized     bool
	maximized     bool
	fullscreen    bool
	cursorMode    CursorMode
	monitor       *Monitor
	nativeHandle  uintptr
}

// state returns the window's state as of the last snapshot.
func (u *glfwBackend) state() windowSnapshot {
	if s := u.snapshot.Load(); s != nil {
		return *s
	}
	return windowSnapshot{}
}

// takeSnapshot records the window's state for the getters. The loop takes one every iteration,
// after the OS events, and every change the app makes takes one, so that a getter returns what
// a setter set.
//
// takeSnapshot must be called from the main thread.
func (u *glfwBackend) takeSnapshot() error {
	if u.closed {
		return nil
	}
	s := windowSnapshot{
		width:  u.windowWidthInDIP,
		height: u.windowHeightInDIP,
		x:      u.windowXInDIP,
		y:      u.windowYInDIP,
	}
	attrib := func(a glfw.Hint) (bool, error) {
		v, err := u.window.GetAttrib(a)
		return v == glfw.True, err
	}
	var err error
	if s.visible, err = attrib(glfw.Visible); err != nil {
		return err
	}
	if s.focused, err = attrib(glfw.Focused); err != nil {
		return err
	}
	if s.minimized, err = attrib(glfw.Iconified); err != nil {
		return err
	}
	if s.maximized, err = u.isWindowMaximized(); err != nil {
		return err
	}
	if s.fullscreen, err = u.isFullscreen(); err != nil {
		return err
	}
	mode, err := u.window.GetInputMode(glfw.CursorMode)
	if err != nil {
		return err
	}
	s.cursorMode = glfwCursorModeToCursorMode(mode)
	if s.monitor, err = u.currentMonitor(); err != nil {
		return err
	}
	if s.nativeHandle, err = u.nativeWindow(); err != nil {
		return err
	}
	u.snapshot.Store(&s)
	return nil
}

// appWindow returns the AppWindow of this backend.
func (u *glfwBackend) appWindow() AppWindow {
	return u.context.window
}

// cursorPositionInDIP converts a cursor position reported by GLFW to device-independent pixels.
// It must be called on the main thread.
func (u *glfwBackend) cursorPositionInDIP(x, y float64) (float64, float64) {
	m, err := u.currentMonitor()
	if err != nil || m == nil {
		return x, y
	}
	s := m.DeviceScaleFactor()
	return dipFromGLFWPixel(x, s), dipFromGLFWPixel(y, s)
}
