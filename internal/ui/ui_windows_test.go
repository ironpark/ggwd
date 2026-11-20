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

package ui

import (
	"math"
	"testing"
)

func TestMonitorDPI(t *testing.T) {
	tests := []struct {
		name  string
		scale float64
		want  uint32
	}{
		{name: "100%", scale: 1.0, want: 96},
		{name: "125%", scale: 1.25, want: 120},
		{name: "150%", scale: 1.5, want: 144},
		{name: "175%", scale: 1.75, want: 168},
		{name: "200%", scale: 2.0, want: 192},
		{name: "225%", scale: 2.25, want: 216},
		{name: "zero", scale: 0, want: 0},
		{name: "negative", scale: -1.5, want: 0},
		{name: "NaN", scale: math.NaN(), want: 0},
		{name: "Inf", scale: math.Inf(1), want: 0},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := monitorDPI(tc.scale); got != tc.want {
				t.Fatalf("monitorDPI(%v) = %d; want %d", tc.scale, got, tc.want)
			}
		})
	}
}

func TestInitialWindowSizeForCreate(t *testing.T) {
	// With scaleToMonitor=true (Windows), CreateWindow receives DIPs directly so
	// GLFW's createNativeWindow does not double-scale.
	if w, h := initialWindowSizeForCreate(800, 600, 2.0, true); w != 800 || h != 600 {
		t.Fatalf("scaleToMonitor=true: got (%d, %d); want (800, 600)", w, h)
	}
	if w, h := initialWindowSizeForCreate(800, 600, 1.5, true); w != 800 || h != 600 {
		t.Fatalf("scaleToMonitor=true at 1.5x: got (%d, %d); want (800, 600)", w, h)
	}

	// With scaleToMonitor=false, CreateWindow receives rounded GLFW pixels.
	if w, h := initialWindowSizeForCreate(800, 600, 1.5, false); w != 1200 || h != 900 {
		t.Fatalf("scaleToMonitor=false at 1.5x: got (%d, %d); want (1200, 900)", w, h)
	}
}
