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

//go:build !android && !ios && !js && !nintendosdk && !playstation5

package ui_test

import (
	"image"
	"math"
	"testing"

	"github.com/ironpark/ggwd/internal/ui"
)

// TestMonitorPositionAndWindowPositionMeet tests that a monitor's corner plus a position on it is
// where that position is in the desktop: the corner is converted as a position on the monitor is,
// so the two add up to the pixel a window put there lands on, divided by the scale.
func TestMonitorPositionAndWindowPositionMeet(t *testing.T) {
	for _, s := range deviceScaleFactorsForTest {
		for _, corner := range []image.Point{{0, 0}, {2560, 0}, {-1920, -300}, {1440, 900}} {
			b := image.Rectangle{Min: corner, Max: corner.Add(image.Pt(1920, 1080))}
			m := ui.NewMonitorWithIDForTest("m", b, s)
			mx, my := m.Position()
			px, py := ui.WindowPositionInGLFWPixelsForTest(100, 50, m)
			wantX := ui.DIPFromGLFWPixelForTest(float64(px), s)
			wantY := ui.DIPFromGLFWPixelForTest(float64(py), s)
			if math.Abs(mx+100-wantX) > 1 || math.Abs(my+50-wantY) > 1 {
				t.Errorf("scale %v, corner %v: the corner (%v, %v) plus (100, 50) is not where the window is, (%v, %v)", s, corner, mx, my, wantX, wantY)
			}
		}
	}
}

// TestCurrentMonitorFollowsTheIDAcrossAnUpdate tests that a monitor from before the monitors were
// listed again is found by its ID in the new list, wherever it was moved to, and that one whose
// display is gone is not taken for another, even one at the same place.
func TestCurrentMonitorFollowsTheIDAcrossAnUpdate(t *testing.T) {
	left := ui.NewMonitorWithIDForTest("left", image.Rect(0, 0, 1920, 1080), 1)
	right := ui.NewMonitorWithIDForTest("right", image.Rect(1920, 0, 3840, 1080), 1)
	if got := ui.CurrentMonitorForTest([]*ui.Monitor{left, right}, right); got != right {
		t.Errorf("a listed monitor: got %p, want %p", got, right)
	}

	// Listed again, the monitors are new values; the right one was moved left of the other.
	left2 := ui.NewMonitorWithIDForTest("left", image.Rect(0, 0, 1920, 1080), 1)
	right2 := ui.NewMonitorWithIDForTest("right", image.Rect(-1920, 0, 0, 1080), 1)
	if got := ui.CurrentMonitorForTest([]*ui.Monitor{left2, right2}, right); got != right2 {
		t.Errorf("a monitor listed again: got %p, want the one with its ID, %p", got, right2)
	}

	// Unplugged, and another monitor took its place.
	other := ui.NewMonitorWithIDForTest("other", image.Rect(1920, 0, 3840, 1080), 1)
	if got := ui.CurrentMonitorForTest([]*ui.Monitor{left2, other}, right); got != nil {
		t.Errorf("a monitor no longer connected: got %p, want nil", got)
	}
	if got := ui.CurrentMonitorForTest([]*ui.Monitor{left2, other}, nil); got != nil {
		t.Errorf("no monitor: got %p, want nil", got)
	}
}
