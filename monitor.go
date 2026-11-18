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

package ggwd

import (
	"github.com/ironpark/ggwd/internal/ui"
)

// Monitor is a monitor available to the system.
type Monitor ui.Monitor

// Name returns the monitor's name. On Linux, this reports the monitors in xrandr format.
// On Windows, this reports "Generic PnP Monitor" for all monitors.
func (m *Monitor) Name() string {
	return (*ui.Monitor)(m).Name()
}

// ID returns a name for the monitor that tells it from every other monitor connected, even one of
// the same model, unlike Name, and that it keeps across restarts where the system allows: what to
// save to put a window back on the same monitor later.
//
//   - macOS: the display's UUID (CGDisplayCreateUUIDFromDisplayID), which macOS keeps the display
//     arrangement under; the same across restarts.
//   - Windows: the monitor's device interface path, which names its model and the connection it
//     is on; the same across restarts while the monitor stays on that connection.
//   - X11: the RandR output, such as "DP-1"; the same across restarts while the monitor stays on
//     that connector.
//
// It is empty in a browser, and for the monitor of a remote viewer, each of which is the only
// monitor there is.
func (m *Monitor) ID() string {
	return (*ui.Monitor)(m).ID()
}

// Position returns the monitor's top-left corner in the desktop, the space every monitor is placed
// in, in device-independent pixels: added to a window's [Window.Position] on the monitor, it is
// where the window is in the desktop.
//
//   - macOS: points from the top-left corner of the main display, y down, as the menu bar's
//     display is arranged in the Displays settings. A display above or left of it has negative
//     coordinates.
//   - Windows: the virtual screen's pixels from the top-left corner of the primary monitor,
//     divided by this monitor's scale. Where monitors have different scales the corners order
//     the monitors as the desktop does, but the distances between them are not to scale.
//   - X11: the root window's pixels divided by the scale, which every monitor shares.
//
// It is 0, 0 in a browser and for the monitor of a remote viewer.
func (m *Monitor) Position() (x, y float64) {
	return (*ui.Monitor)(m).Position()
}

// DeviceScaleFactor returns the device scale factor of the monitor.
//
// DeviceScaleFactor returns a meaningful value on high-DPI display environment,
// otherwise DeviceScaleFactor returns 1.
func (m *Monitor) DeviceScaleFactor() float64 {
	return (*ui.Monitor)(m).DeviceScaleFactor()
}

// Size returns the size of the monitor in device-independent pixels.
// This is the same as the screen size in fullscreen mode.
//
// A window that should fill the monitor can use [Window.SetFullscreen] instead.
func (m *Monitor) Size() (width, height float64) {
	return (*ui.Monitor)(m).Size()
}

// WorkArea returns the part of the monitor that the taskbar, the dock and the menu bar leave to
// windows, in device-independent pixels relative to the monitor's top-left corner: the area a
// window placed with [Window.SetPosition] is fully visible in.
func (m *Monitor) WorkArea() (x, y, width, height float64) {
	return (*ui.Monitor)(m).WorkArea()
}

// RefreshRate returns the monitor's refresh rate in Hz, or 0 when it is unknown, as it is in a
// browser.
func (m *Monitor) RefreshRate() int {
	return (*ui.Monitor)(m).RefreshRate()
}

// AppendMonitors returns the monitors reported by the system.
// On desktop platforms, the first monitor in the slice will be the primary monitor.
// Nothing is appended when no monitor is available.
// Any monitors added or removed will show up with subsequent calls to this function.
//
// AppendMonitors must be called on the main thread before [Run], and is concurrent-safe once Run
// has started.
func AppendMonitors(monitors []*Monitor) []*Monitor {
	for _, m := range ui.Get().AppendMonitors(nil) {
		monitors = append(monitors, (*Monitor)(m))
	}
	return monitors
}
