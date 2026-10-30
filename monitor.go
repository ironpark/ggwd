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
