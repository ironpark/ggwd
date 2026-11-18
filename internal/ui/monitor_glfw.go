// Copyright 2023 The Ebitengine Authors
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
	"image"
	"slices"
	"sync"
	"sync/atomic"

	"github.com/ironpark/ggwd/internal/glfw"
)

// Monitor is a wrapper around glfw.Monitor.
type Monitor struct {
	m         *glfw.Monitor
	videoMode *glfw.VidMode

	// id is the monitor's identity; see glfw.Monitor.GetID.
	id                 string
	name               string
	boundsInGLFWPixels image.Rectangle
	// workAreaInGLFWPixels is the part of the bounds that the taskbar, the dock and the menu
	// bar leave to windows.
	workAreaInGLFWPixels image.Rectangle
	contentScale         float64

	// virtual provides the monitor's values when the monitor has no backing glfw.Monitor (a
	// virtualization guest's monitor). It is nil for a real monitor.
	virtual virtualMonitorSource
}

// virtualMonitorSource provides the values of a monitor that has no backing glfw.Monitor.
type virtualMonitorSource interface {
	deviceScaleFactor() float64
	outsideSize() (width, height float64)
}

// Name returns the monitor's name.
func (m *Monitor) Name() string {
	if m.virtual != nil {
		return ""
	}
	return m.name
}

// ID returns a name that tells the monitor from every other connected, even one of the same
// model, and that it keeps across restarts where the platform allows; see glfw.Monitor.GetID. It
// is empty for a virtual monitor, which is the only one.
func (m *Monitor) ID() string {
	if m.virtual != nil {
		return ""
	}
	return m.id
}

// Position returns the monitor's top-left corner in the desktop, in device-independent pixels;
// see monitorOrigin.
func (m *Monitor) Position() (x, y float64) {
	if m.virtual != nil {
		return 0, 0
	}
	return monitorOrigin(m.boundsInGLFWPixels.Min, m.DeviceScaleFactor())
}

// monitorOrigin converts a monitor's top-left corner in GLFW pixels to device-independent pixels.
// GLFW places every monitor in one space: on macOS, points from the top-left corner of the main
// display, y down, which are device-independent already; on Windows, the virtual screen's
// physical pixels from the top-left corner of the primary monitor; on X11, the root window's
// pixels. Elsewhere than macOS the corner is divided by the monitor's own scale, as a position on
// it is (see windowPositionInGLFWPixels), so that the corner plus a window's position on the
// monitor is where the window is. Where monitors have scales of their own, as Windows allows, the
// corners so made order the monitors as the desktop does but do not keep the distances between
// them; on X11 every monitor has the one scale.
func monitorOrigin(p image.Point, scale float64) (x, y float64) {
	return dipFromGLFWPixel(float64(p.X), scale), dipFromGLFWPixel(float64(p.Y), scale)
}

// DeviceScaleFactor is concurrent-safe as contentScale is immutable.
func (m *Monitor) DeviceScaleFactor() float64 {
	if m.virtual != nil {
		return m.virtual.deviceScaleFactor()
	}
	return m.contentScale
}

// RefreshRate returns the monitor's refresh rate in Hz. It returns 0 when the rate is unknown.
func (m *Monitor) RefreshRate() int {
	if m.videoMode == nil {
		return 0
	}
	return m.videoMode.RefreshRate
}

// Size returns the size of the monitor in device-independent pixels.
func (m *Monitor) Size() (float64, float64) {
	if m.virtual != nil {
		return m.virtual.outsideSize()
	}
	return m.sizeInDIP()
}

// WorkArea returns the part of the monitor that the taskbar, the dock and the menu bar leave to
// windows, in device-independent pixels relative to the monitor's top-left corner.
func (m *Monitor) WorkArea() (x, y, width, height float64) {
	if m.virtual != nil {
		w, h := m.virtual.outsideSize()
		return 0, 0, w, h
	}
	s := m.DeviceScaleFactor()
	a := m.workAreaInGLFWPixels.Sub(m.boundsInGLFWPixels.Min)
	return dipFromGLFWPixel(float64(a.Min.X), s), dipFromGLFWPixel(float64(a.Min.Y), s),
		dipFromGLFWPixel(float64(a.Dx()), s), dipFromGLFWPixel(float64(a.Dy()), s)
}

func (m *Monitor) sizeInDIP() (float64, float64) {
	w, h := m.boundsInGLFWPixels.Dx(), m.boundsInGLFWPixels.Dy()
	s := m.DeviceScaleFactor()
	return dipFromGLFWPixel(float64(w), s), dipFromGLFWPixel(float64(h), s)
}

type monitors struct {
	// monitors is the monitor list cache for desktop glfw compile targets.
	// populated by 'updateMonitors' which is called on init and every
	// monitor config change event.
	monitors []*Monitor

	mu sync.Mutex

	updateCalled atomic.Bool
}

var theMonitors monitors

func (m *monitors) append(ms []*Monitor) []*Monitor {
	if !m.updateCalled.Load() {
		panic("ui: (*monitors).update must be called before (*monitors).append is called")
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	return append(ms, m.monitors...)
}

func (m *monitors) contains(monitor *Monitor) bool {
	if !m.updateCalled.Load() {
		return false
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	return slices.Contains(m.monitors, monitor)
}

func (m *monitors) primaryMonitor() *Monitor {
	if !m.updateCalled.Load() {
		panic("ui: (*monitors).update must be called before (*monitors).primaryMonitor is called")
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	// GetMonitors might return nil in theory (#1878, #1887, #3241).
	// primaryMonitor can be called at the initialization, so monitors can be nil.
	if len(m.monitors) == 0 {
		return nil
	}
	return m.monitors[0]
}

// current returns the monitor of the list that is monitor, or that has its ID, or nil when its
// display is gone: the list is made anew whenever the monitors change, and a *Monitor from before
// is not in it.
func (m *monitors) current(monitor *Monitor) *Monitor {
	if monitor == nil || !m.updateCalled.Load() {
		return nil
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	if slices.Contains(m.monitors, monitor) {
		return monitor
	}
	if monitor.id == "" {
		return nil
	}
	for _, c := range m.monitors {
		if c.id == monitor.id {
			return c
		}
	}
	return nil
}

// monitorFromPosition returns a monitor for the given position (x, y),
// or returns nil if monitor is not found.
// The position is in GLFW pixels.
func (m *monitors) monitorFromPosition(x, y int) *Monitor {
	m.mu.Lock()
	defer m.mu.Unlock()

	for _, m := range m.monitors {
		// Use an inclusive range. On macOS, the cursor position can take this range (#2794).
		b := m.boundsInGLFWPixels
		if b.Min.X <= x && x <= b.Max.X && b.Min.Y <= y && y <= b.Max.Y {
			return m
		}
	}
	return nil
}

// update must be called from the main thread.
func (m *monitors) update() error {
	glfwMonitors, err := glfw.GetMonitors()
	if err != nil {
		return err
	}
	newMonitors := make([]*Monitor, 0, len(glfwMonitors))
	for _, m := range glfwMonitors {
		x, y, err := m.GetPos()
		if err != nil {
			return err
		}

		contentScale := 1.0

		// Keep calling GetContentScale until the returned scale is not 0 (#2051).
		// Retry this at most 5 times to avoid an infinite loop.
		for range 5 {
			// An error can happen e.g. when entering a screensaver on Windows (#2488).
			sx, _, err := m.GetContentScale()
			if err != nil {
				continue
			}
			if sx == 0 {
				continue
			}
			contentScale = float64(sx)
			break
		}

		videoMode, err := m.GetVideoMode()
		if err != nil {
			return err
		}
		name, err := m.GetName()
		if err != nil {
			return err
		}
		id, err := m.GetID()
		if err != nil {
			return err
		}
		w, h, err := glfwMonitorSizeInGLFWPixels(m)
		if err != nil {
			return err
		}
		b := image.Rect(x, y, x+w, y+h)
		wa := b
		if wx, wy, ww, wh, err := m.GetWorkarea(); err == nil && ww > 0 && wh > 0 {
			wa = image.Rect(wx, wy, wx+ww, wy+wh).Intersect(b)
		}
		newMonitors = append(newMonitors, &Monitor{
			m:                    m,
			videoMode:            videoMode,
			id:                   id,
			name:                 name,
			boundsInGLFWPixels:   b,
			workAreaInGLFWPixels: wa,
			contentScale:         contentScale,
		})
	}

	m.mu.Lock()
	m.monitors = newMonitors
	m.mu.Unlock()

	m.updateCalled.Store(true)
	return nil
}
