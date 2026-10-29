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

package gamepad

import (
	"github.com/ironpark/ggwd/gamepaddb"
)

// Event is what [Tracker.Update] reports: one of ConnectEvent, DisconnectEvent, ButtonEvent,
// AxisEvent, StandardButtonEvent and StandardAxisEvent.
type Event interface {
	isEvent()
}

// ConnectEvent reports a gamepad that appeared. Standard reports whether the gamepad has a
// standard layout, and so whether the standard events are reported for it.
type ConnectEvent struct {
	ID       ID
	Name     string
	SDLID    string
	Standard bool
}

// DisconnectEvent reports a gamepad that went away.
type DisconnectEvent struct {
	ID ID
}

// ButtonEvent reports one raw button changing. Button indices at and above the real button count
// are the hats' four directions.
type ButtonEvent struct {
	ID      ID
	Button  int
	Pressed bool
}

// AxisEvent reports one raw axis moving, in -1..1.
type AxisEvent struct {
	ID    ID
	Axis  int
	Value float64
}

// StandardButtonEvent reports one standard-layout button changing. Value is 0..1 for an analog
// button, such as a trigger; Pressed applies the dead zone an analog button needs, so a resting
// trigger is not pressed even when its value is not quite zero.
type StandardButtonEvent struct {
	ID      ID
	Button  gamepaddb.StandardButton
	Pressed bool
	Value   float64
}

// StandardAxisEvent reports one standard-layout axis moving, in -1..1.
type StandardAxisEvent struct {
	ID    ID
	Axis  gamepaddb.StandardAxis
	Value float64
}

func (ConnectEvent) isEvent()        {}
func (DisconnectEvent) isEvent()     {}
func (ButtonEvent) isEvent()         {}
func (AxisEvent) isEvent()           {}
func (StandardButtonEvent) isEvent() {}
func (StandardAxisEvent) isEvent()   {}
