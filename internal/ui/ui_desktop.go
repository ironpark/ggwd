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
	"sync"
	"sync/atomic"
	"time"

	"github.com/ironpark/ggwd/internal/glfw"
)

type userInterfaceImpl struct {
	// glContextWindow is the hidden window whose context the OpenGL driver draws in, and nil with
	// the other drivers. It lives until GLFW terminates.
	glContextWindow *glfw.Window

	// renderOnMainThread reports whether frames are rendered on the main thread, as in the single
	// thread mode.
	renderOnMainThread bool

	// windows are the open windows, in the order they were created. windows must be accessed
	// from the main thread.
	windows []*glfwBackend

	// pollingEvents reports whether the main thread is polling events for the loop.
	// pollingEvents must be accessed from the main thread.
	pollingEvents bool

	// forcingFrame reports whether a frame is being rendered from an event callback.
	// forcingFrame must be accessed from the main thread.
	forcingFrame bool

	// waitingEvents reports whether the main thread waits for OS events, and serves no call
	// until it has one. pendingCalls counts the calls on their way to the main thread, which
	// keep it from starting such a wait.
	waitingEvents atomic.Bool
	pendingCalls  atomic.Int32

	unfocusedNextWake time.Time

	// initMonitor is the monitor a window is created on unless it is given a position.
	initMonitor atomic.Pointer[Monitor]

	glfwInitOnce sync.Once
}

func (u *UserInterface) init() error {
	return nil
}

func (u *UserInterface) setInitMonitor(m *Monitor) {
	u.initMonitor.Store(m)
}

func (u *UserInterface) getInitMonitor() *Monitor {
	return u.initMonitor.Load()
}

// AppendMonitors appends the current monitors to the passed in mons slice and returns it.
func (u *UserInterface) AppendMonitors(monitors []*Monitor) []*Monitor {
	// Ensure GLFW is initialized so that the monitor list is available.
	if err := u.ensureGLFWInit(); err != nil {
		return monitors
	}
	return theMonitors.append(monitors)
}

// RunOnMainThread runs f on the main thread and returns after it ran. It runs f at once when it
// is called on the main thread, and does nothing unless the loop runs.
func (u *UserInterface) RunOnMainThread(f func()) {
	if !u.isRunning() || u.isTerminated() {
		return
	}
	u.runOnMainThread(func() { withAutoreleasePool(f) })
}

// runOnMainThread runs f on the main thread and returns after it ran, at once when it is called
// there.
func (u *UserInterface) runOnMainThread(f func()) {
	if t, ok := u.mainThread.(interface{ IsCurrent() bool }); ok && t.IsCurrent() {
		f()
		return
	}
	// The main thread can be blocked in WaitEvents, which serves no call until the OS has an
	// event. Wake it then, and keep it from starting a wait until f runs; the loop goroutine has
	// work to do after the wait returns, so the main thread picks up f before the loop asks it to
	// wait again. A call while the loop handles events does not wake it, which would only run
	// another iteration.
	u.pendingCalls.Add(1)
	defer u.pendingCalls.Add(-1)
	if u.waitingEvents.Load() {
		u.wakeLoop()
	}
	u.mainThread.Call(f)
}

func (u *UserInterface) KeyName(key Key) string {
	gk, ok := uiKeyToGLFWKey[key]
	if !ok {
		return ""
	}
	var name string
	u.RunOnMainThread(func() {
		n, err := glfw.GetKeyName(gk, 0)
		if err != nil {
			u.setError(err)
			return
		}
		name = n
	})
	return name
}

// wakeLoop makes the loop run an iteration. It does nothing before the loop runs, whose first
// iteration dispatches the queued events anyway.
func (u *UserInterface) wakeLoop() {
	// This check can slip past a termination running on the main thread, and then
	// PostEmptyEvent touches GLFW's state after glfw.Terminate. PostEmptyEvent must stay
	// harmless in that case.
	if !u.isRunning() || u.isTerminated() {
		return
	}
	// The main thread can be blocked in WaitEvents; wake it. PostEmptyEvent is concurrent safe.
	if err := glfw.PostEmptyEvent(); err != nil {
		u.setError(err)
	}
}

// applyColorMode gives every open window the preferred color mode.
func (u *UserInterface) applyColorMode() {
	u.RunOnMainThread(func() {
		m := u.PreferredColorMode()
		for _, w := range u.windows {
			if err := w.setWindowColorModeImpl(m); err != nil {
				u.setError(err)
				return
			}
		}
	})
}

// focusedWindow returns the window that has the keyboard focus, or nil when none has it.
//
// focusedWindow must be called from the main thread.
func (u *UserInterface) focusedWindow() *glfwBackend {
	for _, w := range u.windows {
		if a, err := w.window.GetAttrib(glfw.Focused); err == nil && a == glfw.True {
			return w
		}
	}
	return nil
}

// monitorDeviceScaleFactor returns the device scale factor of the focused window's monitor, or 1
// when no window has the focus.
func (u *UserInterface) monitorDeviceScaleFactor() float64 {
	s := 1.0
	u.RunOnMainThread(func() {
		w := u.focusedWindow()
		if w == nil {
			return
		}
		if m, err := w.currentMonitor(); err == nil && m != nil {
			s = m.DeviceScaleFactor()
		}
	})
	return s
}
