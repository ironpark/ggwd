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

//go:build !js

// Package backgroundtest checks that background frames run while the app's only window is
// hidden or minimized and draws no frames of its own: a viewer of a remote app depends on it. It
// is a package of its own because it runs ggwd.Run, once a process.
package backgroundtest

import (
	"errors"
	"fmt"
	"image/color"
	"os"
	"runtime"
	"testing"
	"time"

	"github.com/ironpark/ggfx"
	"github.com/ironpark/ggwd"
)

func init() {
	// ggwd.Run runs on the main thread.
	runtime.LockOSThread()
}

var result error

func TestMain(m *testing.M) {
	result = run()
	os.Exit(m.Run())
}

// run opens a window, hides it, then minimizes it once shown again, and in each state asks for
// a window frame and for background frames: the window frame must wait, and the background frames
// must come, drawing to an image and reading it back.
func run() error {
	var w *ggwd.Window
	var img *ggfx.Image
	phase := 0 // 0 shown, 1 hidden, 2 shown again, 3 minimized
	backgrounds := 0
	framesWhileAway := map[int]int{} // by phase
	deadline := time.Now().Add(10 * time.Second)
	next := func() {
		backgrounds = 0
		switch phase {
		case 1:
			w.Hide()
		case 2:
			w.Show()
		case 3:
			w.Minimize()
		}
		w.RequestFrame()
		ggwd.RequestBackgroundFrame()
	}
	err := ggwd.Run(ggwd.HandlerFunc(func(ev ggwd.Event) error {
		if time.Now().After(deadline) {
			return fmt.Errorf("phase %d: %d background frames in 10s", phase, backgrounds)
		}
		switch ev := ev.(type) {
		case ggwd.StartEvent:
			var err error
			w, err = ggwd.NewWindow(&ggwd.WindowOptions{Title: "ggwd background frames", Width: 160, Height: 120})
			if err != nil {
				return err
			}
			img = ggfx.NewImage(4, 4)
		case ggwd.FrameEvent:
			ev.Screen.Fill(color.Black)
			if phase == 1 || phase == 3 {
				framesWhileAway[phase]++
			}
			if phase == 0 || phase == 2 {
				phase++
				next()
			}
		case ggwd.BackgroundFrameEvent:
			img.Fill(color.RGBA{byte(backgrounds), 0x80, 0, 0xff})
			if got := img.At(1, 1).(color.RGBA); got.R != byte(backgrounds) {
				return fmt.Errorf("a background frame read %v back", got)
			}
			backgrounds++
			if backgrounds < 10 {
				ggwd.RequestBackgroundFrame()
				return nil
			}
			if phase == 1 {
				phase = 2
				next()
				return nil
			}
			if phase == 3 {
				return ggwd.Termination
			}
		}
		return nil
	}), nil)
	if err != nil {
		return err
	}
	for phase, n := range framesWhileAway {
		// One frame may already be on its way when the window goes.
		if n > 1 {
			return fmt.Errorf("phase %d: %d window frames ran while the window was hidden or minimized", phase, n)
		}
	}
	return nil
}

func TestBackgroundFramesRunWhileTheWindowIsAway(t *testing.T) {
	if errors.Is(result, ggwd.Termination) {
		result = nil
	}
	if result != nil {
		t.Error(result)
	}
}
