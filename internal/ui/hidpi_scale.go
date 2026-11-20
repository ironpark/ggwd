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

package ui

import (
	"image"
	"math"
	"strconv"
)

// sanitizeDevicePixelRatio returns ratio when it is a positive finite number,
// and 1 otherwise.
func sanitizeDevicePixelRatio(ratio float64) float64 {
	if math.IsNaN(ratio) || math.IsInf(ratio, 0) || ratio <= 0 {
		return 1
	}
	return ratio
}

// scaledCanvasPixels converts a logical CSS dimension to a backing-store pixel
// count using nearest-integer rounding (avoiding 1px gaps/blur at fractional
// DPRs like 125%/150%) and clamping to at least 1 pixel.
func scaledCanvasPixels(logical, scale float64) int {
	if math.IsNaN(logical) || math.IsInf(logical, 0) || logical <= 0 {
		return 1
	}
	scale = sanitizeDevicePixelRatio(scale)
	return max(1, int(math.Round(logical*scale)))
}

// dprMediaQuery formats the CSS media query used to detect changes in
// window.devicePixelRatio via window.matchMedia.
func dprMediaQuery(dpr float64) string {
	return "(resolution: " + strconv.FormatFloat(sanitizeDevicePixelRatio(dpr), 'f', -1, 64) + "dppx)"
}

// monitorIndexForWindowRect returns the index of the monitor in monitors that
// best contains win: the monitor with the largest intersection area, or when
// win does not overlap any monitor, the monitor closest to win's center. It
// returns -1 when monitors is empty or win is empty.
func monitorIndexForWindowRect(win image.Rectangle, monitors []image.Rectangle) int {
	if len(monitors) == 0 || win.Empty() {
		return -1
	}

	bestIdx := -1
	var bestArea int64
	for i, m := range monitors {
		if m.Empty() {
			continue
		}
		inter := win.Intersect(m)
		if inter.Empty() {
			continue
		}
		area := int64(inter.Dx()) * int64(inter.Dy())
		if area > bestArea {
			bestArea = area
			bestIdx = i
		}
	}
	if bestIdx >= 0 {
		return bestIdx
	}

	// No overlap: choose the monitor nearest to the window's center.
	cx := int64(win.Min.X) + int64(win.Dx())/2
	cy := int64(win.Min.Y) + int64(win.Dy())/2
	var bestDist int64
	for i, m := range monitors {
		if m.Empty() {
			continue
		}
		var dx, dy int64
		if cx < int64(m.Min.X) {
			dx = int64(m.Min.X) - cx
		} else if cx > int64(m.Max.X) {
			dx = cx - int64(m.Max.X)
		}
		if cy < int64(m.Min.Y) {
			dy = int64(m.Min.Y) - cy
		} else if cy > int64(m.Max.Y) {
			dy = cy - int64(m.Max.Y)
		}
		dist := dx*dx + dy*dy
		if bestIdx < 0 || dist < bestDist {
			bestDist = dist
			bestIdx = i
		}
	}
	return bestIdx
}
