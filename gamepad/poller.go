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
	"sync"
	"time"

	"github.com/ironpark/ggwd"
	"github.com/ironpark/ggwd/native"
)

// Poller polls the gamepads for a ggwd app. It reads them on the main thread, through
// ggwd.RunOnMainThread, as often as its tracker asks, and wakes the loop with ggwd.Wake when it
// found something, so the loop still sleeps while nothing happens. The handler takes the events
// on ggwd.WakeEvent.
type Poller struct {
	mu      sync.Mutex
	pending []Event
	err     error

	stop     chan struct{}
	done     chan struct{}
	stopOnce sync.Once
}

// Start starts polling the gamepads. Call it from the handler, once the app has a window:
// on Windows the gamepads are read through window, which should live as long as they are read.
// window can be nil elsewhere.
func Start(window *ggwd.Window) *Poller {
	var hwnd uintptr
	if window != nil {
		// native.Handle runs on the main thread, so it cannot be asked from within a poll.
		hwnd = native.Handle(window)
	}
	g := &Poller{stop: make(chan struct{}), done: make(chan struct{})}
	go g.poll(hwnd)
	return g
}

func (g *Poller) poll(hwnd uintptr) {
	defer close(g.done)
	var tracker Tracker
	// The poll and its results live across the iterations, so a poll allocates nothing.
	var err error
	var found bool
	update := func() {
		if err = Update(hwnd, nil); err != nil {
			return
		}
		if evs := tracker.Update(); len(evs) > 0 {
			g.mu.Lock()
			g.pending = append(g.pending, evs...)
			g.mu.Unlock()
			found = true
		}
	}
	timer := time.NewTimer(0)
	<-timer.C
	defer timer.Stop()
	for {
		found = false
		ggwd.RunOnMainThread(update)
		if err != nil {
			g.mu.Lock()
			g.err = err
			g.mu.Unlock()
			ggwd.Wake()
			return
		}
		if found {
			ggwd.Wake()
		}
		timer.Reset(tracker.WaitTimeout())
		select {
		case <-g.stop:
			return
		case <-timer.C:
		}
	}
}

// Take returns the events found since the last Take, in order. Call it on ggwd.WakeEvent.
func (g *Poller) Take() []Event {
	g.mu.Lock()
	defer g.mu.Unlock()
	evs := g.pending
	g.pending = nil
	return evs
}

// Err returns the error that stopped the polling, or nil while it runs.
func (g *Poller) Err() error {
	g.mu.Lock()
	defer g.mu.Unlock()
	return g.err
}

// Stop stops the polling and waits for it to end. Do not call it from within the main thread.
func (g *Poller) Stop() {
	g.stopOnce.Do(func() { close(g.stop) })
	<-g.done
}
