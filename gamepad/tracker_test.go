// Copyright 2026 The ggip Authors
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

package gamepad_test

import (
	"testing"

	"github.com/ironpark/ggwd/gamepad"
	"github.com/ironpark/ggwd/gamepaddb"
)

// poll drives the tracker with one virtual gamepad snapshot and returns the events it produced.
func poll(t *testing.T, tr *gamepad.Tracker, states []gamepad.VirtualGamepadState) []gamepad.Event {
	t.Helper()
	if err := gamepad.Update(0, states); err != nil {
		t.Fatal(err)
	}
	// The tracker rate-limits itself; a test polls whenever it says so.
	gamepad.ResetLastPoll(tr)
	return tr.Update()
}

func TestTrackerReportsConnectionAndDisconnection(t *testing.T) {
	tr := &gamepad.Tracker{}
	defer poll(t, tr, []gamepad.VirtualGamepadState{})

	evs := poll(t, tr, []gamepad.VirtualGamepadState{
		{ID: 0, SDLID: "id0", Name: "Pad 0", Buttons: []bool{false, false}, Axes: []float64{0, 0}},
	})
	if len(evs) != 1 {
		t.Fatalf("first poll produced %d events; want 1 connect: %v", len(evs), evs)
	}
	c, ok := evs[0].(gamepad.ConnectEvent)
	if !ok {
		t.Fatalf("first event is %T; want ConnectEvent", evs[0])
	}
	if c.ID != 0 || c.SDLID != "id0" {
		t.Errorf("connect event = %+v; want ID 0 and SDLID id0", c)
	}
	if !tr.Connected() {
		t.Error("connected() = false after a gamepad connected")
	}

	// The resting state of a fresh gamepad is not a change.
	if evs := poll(t, tr, []gamepad.VirtualGamepadState{
		{ID: 0, SDLID: "id0", Name: "Pad 0", Buttons: []bool{false, false}, Axes: []float64{0, 0}},
	}); len(evs) != 0 {
		t.Errorf("unchanged poll produced %v; want no events", evs)
	}

	evs = poll(t, tr, []gamepad.VirtualGamepadState{})
	if len(evs) != 1 {
		t.Fatalf("removal produced %d events; want 1 disconnect: %v", len(evs), evs)
	}
	if d, ok := evs[0].(gamepad.DisconnectEvent); !ok || d.ID != 0 {
		t.Errorf("event = %+v; want DisconnectEvent for ID 0", evs[0])
	}
	if tr.Connected() {
		t.Error("connected() = true after the only gamepad went away")
	}
}

func TestTrackerReportsButtonsAndAxes(t *testing.T) {
	tr := &gamepad.Tracker{}
	defer poll(t, tr, []gamepad.VirtualGamepadState{})

	poll(t, tr, []gamepad.VirtualGamepadState{
		{ID: 0, SDLID: "id0", Buttons: []bool{false, false}, Axes: []float64{0, 0}},
	})

	evs := poll(t, tr, []gamepad.VirtualGamepadState{
		{ID: 0, SDLID: "id0", Buttons: []bool{false, true}, Axes: []float64{0, 0.5}},
	})

	var button *gamepad.ButtonEvent
	var axis *gamepad.AxisEvent
	for _, ev := range evs {
		switch ev := ev.(type) {
		case gamepad.ButtonEvent:
			if button != nil {
				t.Fatalf("more than one button event: %v", evs)
			}
			button = &ev
		case gamepad.AxisEvent:
			if axis != nil {
				t.Fatalf("more than one axis event: %v", evs)
			}
			axis = &ev
		}
	}
	if button == nil {
		t.Fatalf("no button event in %v", evs)
	}
	if button.Button != 1 || !button.Pressed {
		t.Errorf("button event = %+v; want button 1 pressed", *button)
	}
	if axis == nil {
		t.Fatalf("no axis event in %v", evs)
	}
	if axis.Axis != 1 || axis.Value != 0.5 {
		t.Errorf("axis event = %+v; want axis 1 at 0.5", *axis)
	}
}

func TestTrackerIgnoresAxisNoise(t *testing.T) {
	tr := &gamepad.Tracker{}
	defer poll(t, tr, []gamepad.VirtualGamepadState{})

	poll(t, tr, []gamepad.VirtualGamepadState{
		{ID: 0, SDLID: "id0", Axes: []float64{0}},
	})

	// A resting stick jitters well below the threshold; reporting it would wake the loop forever.
	evs := poll(t, tr, []gamepad.VirtualGamepadState{
		{ID: 0, SDLID: "id0", Axes: []float64{gamepad.AxisThreshold / 2}},
	})
	for _, ev := range evs {
		if _, ok := ev.(gamepad.AxisEvent); ok {
			t.Errorf("a move below the threshold produced %+v", ev)
		}
	}
}

func TestTrackerReportsStandardLayout(t *testing.T) {
	tr := &gamepad.Tracker{}
	defer poll(t, tr, []gamepad.VirtualGamepadState{})

	state := func(v float64) []gamepad.VirtualGamepadState {
		return []gamepad.VirtualGamepadState{{
			ID:    0,
			SDLID: "id0",
			StandardButtons: map[gamepaddb.StandardButton]gamepad.VirtualStandardGamepadButton{
				gamepaddb.StandardButtonRightBottom: {Pressed: v > 0, Value: v},
			},
			StandardAxes: map[gamepaddb.StandardAxis]float64{
				gamepaddb.StandardAxisLeftStickHorizontal: v,
			},
		}}
	}

	evs := poll(t, tr, state(0))
	if len(evs) != 1 {
		t.Fatalf("first poll produced %d events; want 1 connect: %v", len(evs), evs)
	}
	if c := evs[0].(gamepad.ConnectEvent); !c.Standard {
		t.Error("connect event reports no standard layout for a gamepad that has one")
	}

	evs = poll(t, tr, state(1))
	var sawButton, sawAxis bool
	for _, ev := range evs {
		switch ev := ev.(type) {
		case gamepad.StandardButtonEvent:
			sawButton = true
			if ev.Button != gamepaddb.StandardButtonRightBottom || !ev.Pressed || ev.Value != 1 {
				t.Errorf("standard button event = %+v", ev)
			}
		case gamepad.StandardAxisEvent:
			sawAxis = true
			if ev.Axis != gamepaddb.StandardAxisLeftStickHorizontal || ev.Value != 1 {
				t.Errorf("standard axis event = %+v", ev)
			}
		}
	}
	if !sawButton {
		t.Errorf("no standard button event in %v", evs)
	}
	if !sawAxis {
		t.Errorf("no standard axis event in %v", evs)
	}
}
