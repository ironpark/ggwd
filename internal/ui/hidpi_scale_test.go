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
	"testing"
)

func TestSanitizeDevicePixelRatio(t *testing.T) {
	tests := []struct {
		in   float64
		want float64
	}{
		{1, 1},
		{1.25, 1.25},
		{1.5, 1.5},
		{2, 2},
		{0.75, 0.75},
		{0, 1},
		{-1, 1},
		{math.NaN(), 1},
		{math.Inf(1), 1},
		{math.Inf(-1), 1},
	}
	for _, tc := range tests {
		got := sanitizeDevicePixelRatio(tc.in)
		if got != tc.want {
			t.Errorf("sanitizeDevicePixelRatio(%v) = %v, want %v", tc.in, got, tc.want)
		}
	}
}

func TestScaledCanvasPixels(t *testing.T) {
	tests := []struct {
		logical float64
		scale   float64
		want    int
	}{
		// Exact integer scales
		{800, 1.0, 800},
		{800, 2.0, 1600},
		// Fractional DPRs where truncation caused 1px gaps/blur
		{801, 1.5, 1202},  // 1201.5 -> rounds to 1202 (truncation gave 1201)
		{300, 1.15, 345},  // 344.99999999999994 -> rounds to 345 (truncation gave 344)
		{801, 1.25, 1001}, // 1001.25 -> rounds to 1001
		{803, 1.25, 1004}, // 1003.75 -> rounds to 1004 (truncation gave 1003)
		{801, 1.75, 1402}, // 1401.75 -> rounds to 1402 (truncation gave 1401)
		{1000, 1.3333333, 1333},
		// Clamping & invalid inputs
		{1, 0.25, 1},
		{0, 2.0, 1},
		{-10, 2.0, 1},
		{math.NaN(), 2.0, 1},
		{800, 0, 800},
		{800, math.NaN(), 800},
	}
	for _, tc := range tests {
		got := scaledCanvasPixels(tc.logical, tc.scale)
		if got != tc.want {
			t.Errorf("scaledCanvasPixels(%v, %v) = %d, want %d", tc.logical, tc.scale, got, tc.want)
		}
	}
}

func TestDPRMediaQuery(t *testing.T) {
	tests := []struct {
		dpr  float64
		want string
	}{
		{1, "(resolution: 1dppx)"},
		{1.25, "(resolution: 1.25dppx)"},
		{1.5, "(resolution: 1.5dppx)"},
		{2, "(resolution: 2dppx)"},
		{0, "(resolution: 1dppx)"},
		{math.NaN(), "(resolution: 1dppx)"},
	}
	for _, tc := range tests {
		if got := dprMediaQuery(tc.dpr); got != tc.want {
			t.Errorf("dprMediaQuery(%v) = %q, want %q", tc.dpr, got, tc.want)
		}
	}
}

func TestMonitorIndexForWindowRect(t *testing.T) {
	left := image.Rect(0, 0, 1920, 1080)
	right := image.Rect(1920, 0, 3840, 1080)
	mons := []image.Rectangle{left, right}

	tests := []struct {
		name string
		win  image.Rectangle
		mons []image.Rectangle
		want int
	}{
		{
			name: "entirely on primary",
			win:  image.Rect(100, 100, 900, 700),
			mons: mons,
			want: 0,
		},
		{
			name: "entirely on secondary",
			win:  image.Rect(2000, 100, 2800, 700),
			mons: mons,
			want: 1,
		},
		{
			name: "straddling monitors majority on secondary despite top-left on primary",
			win:  image.Rect(1850, 100, 2650, 700),
			mons: mons,
			want: 1,
		},
		{
			name: "straddling monitors majority on primary",
			win:  image.Rect(1200, 100, 2000, 700),
			mons: mons,
			want: 0,
		},
		{
			name: "slightly off top-left edge of secondary",
			win:  image.Rect(1920, -20, 2720, 580),
			mons: mons,
			want: 1,
		},
		{
			name: "completely off-screen closest to secondary",
			win:  image.Rect(4000, 100, 4800, 700),
			mons: mons,
			want: 1,
		},
		{
			name: "completely above secondary edge-aligned with boundary",
			win:  image.Rect(1920, -400, 2720, -100),
			mons: mons,
			want: 1,
		},
		{
			name: "empty window rect",
			win:  image.Rect(100, 100, 100, 100),
			mons: mons,
			want: -1,
		},
		{
			name: "no monitors",
			win:  image.Rect(100, 100, 900, 700),
			mons: nil,
			want: -1,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := monitorIndexForWindowRect(tc.win, tc.mons); got != tc.want {
				t.Errorf("monitorIndexForWindowRect(%v) = %d, want %d", tc.win, got, tc.want)
			}
		})
	}
}
