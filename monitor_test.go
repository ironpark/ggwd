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

//go:build !js

package ggwd_test

import (
	"testing"

	"github.com/ironpark/ggwd"
	etesting "github.com/ironpark/ggwd/internal/testing"
)

func TestMonitorIDsTellMonitorsApart(t *testing.T) {
	ms := ggwd.AppendMonitors(nil)
	if len(ms) == 0 {
		t.Fatal("no monitors")
	}
	seen := map[string]bool{}
	for _, m := range ms {
		x, y := m.Position()
		w, h := m.Size()
		t.Logf("monitor %q: ID %q, at (%g, %g), %gx%g", m.Name(), m.ID(), x, y, w, h)
		if m.ID() == "" {
			t.Errorf("monitor %q has no ID", m.Name())
		}
		if seen[m.ID()] {
			t.Errorf("two monitors have the ID %q", m.ID())
		}
		seen[m.ID()] = true
	}
}

func TestWindowOpensOnTheMonitorItIsGiven(t *testing.T) {
	for _, m := range ggwd.AppendMonitors(nil) {
		w, err := ggwd.NewWindow(&ggwd.WindowOptions{
			Title:    "ggwd monitor test",
			Width:    200,
			Height:   150,
			Hidden:   true,
			Position: &ggwd.Point{X: 40, Y: 60},
			Monitor:  m,
		})
		if err != nil {
			t.Fatal(err)
		}
		if got := w.Monitor(); got == nil || got.ID() != m.ID() {
			t.Errorf("a window opened on %q is on %v", m.ID(), got)
		}
		if x, y := w.Position(); x != 40 || y != 60 {
			t.Errorf("a window opened at (40, 60) on %q is at (%g, %g)", m.ID(), x, y)
		}
		w.Close()
	}
}

func TestSetPositionOnMovesTheWindowOntoTheMonitor(t *testing.T) {
	w := etesting.Window()
	x0, y0 := w.Position()
	defer w.SetPosition(x0, y0)

	// On its own monitor it is SetPosition, whatever the window manager makes of the position.
	w.SetPosition(70, 90)
	x1, y1 := w.Position()
	w.SetPosition(x0, y0)
	w.SetPositionOn(w.Monitor(), 70, 90)
	if x2, y2 := w.Position(); x2 != x1 || y2 != y1 {
		t.Errorf("SetPositionOn its own monitor moved the window to (%g, %g); SetPosition to (%g, %g)", x2, y2, x1, y1)
	}

	ms := ggwd.AppendMonitors(nil)
	if len(ms) < 2 {
		t.Logf("one monitor: moving onto another is not tested")
		return
	}
	home := w.Monitor()
	defer w.SetPositionOn(home, x0, y0)
	for _, m := range ms {
		w.SetPositionOn(m, 70, 90)
		if got := w.Monitor(); got == nil || got.ID() != m.ID() {
			t.Errorf("SetPositionOn(%q) left the window on %v", m.ID(), got)
		}
	}
}
