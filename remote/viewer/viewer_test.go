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

package viewer_test

import (
	"reflect"
	"testing"
	"testing/fstest"

	"github.com/ironpark/ggwd"
	"github.com/ironpark/ggwd/remote"
	"github.com/ironpark/ggwd/remote/viewer"
)

func TestInput(t *testing.T) {
	for _, c := range []struct {
		ev   ggwd.Event
		want remote.Input
	}{
		{ggwd.MouseButtonEvent{Button: ggwd.MouseButtonRight, Pressed: true, X: 1, Y: 2, Modifiers: ggwd.Mods{Shift: true, Meta: true}},
			remote.Input{Kind: remote.InputMouseButton, Button: int(ggwd.MouseButtonRight), Pressed: true, X: 1, Y: 2, Mods: remote.ModShift | remote.ModMeta}},
		{ggwd.CompositionEvent{Text: "한", Start: 0, End: 3, Done: true},
			remote.Input{Kind: remote.InputComposition, Text: "한", Start: 0, End: 3, Done: true}},
		{ggwd.TextEvent{Text: "가"}, remote.Input{Kind: remote.InputText, Text: "가"}},
		{ggwd.TextEvent{Text: "가", Replacement: &ggwd.TextRange{Start: 2, End: 5}, Passthrough: true},
			remote.Input{Kind: remote.InputText, Text: "가", HasReplacement: true, Start: 2, End: 5, Passthrough: true}},
		// A file system of no real files, as a test's, has no paths to forward.
		{ggwd.DragEvent{Phase: ggwd.DragMoved, X: 3, Y: 4, Files: fstest.MapFS{"a": {}}},
			remote.Input{Kind: remote.InputDrag, Phase: int(ggwd.DragMoved), X: 3, Y: 4}},
	} {
		got, ok := viewer.Input(c.ev)
		if !ok || !reflect.DeepEqual(got, c.want) {
			t.Errorf("Input(%#v) = %+v, %t; want %+v", c.ev, got, ok, c.want)
		}
	}
	for _, ev := range []ggwd.Event{ggwd.FrameEvent{}, ggwd.ResizeEvent{}, ggwd.FocusEvent{}, ggwd.CloseEvent{}} {
		if _, ok := viewer.Input(ev); ok {
			t.Errorf("%T is input", ev)
		}
	}
}

func TestWindowOptions(t *testing.T) {
	got := viewer.WindowOptions(&remote.OpenWindow{
		Title: "t", Width: 300, Height: 200,
		PositionSet: true, X: 10, Y: 20,
		Resizable: true, MinWidth: 100, MinHeight: -1, MaxWidth: -1, MaxHeight: 900,
		Decorated: false, Floating: true, Hidden: true, Maximized: true, Unfocused: true,
		TitlebarHidden: true, TrafficLightsSet: true, TrafficLightsX: 14, TrafficLightsY: 16,
		Transparent: true,
	})
	want := &ggwd.WindowOptions{
		Title: "t", Width: 300, Height: 200,
		Position:  &ggwd.Point{X: 10, Y: 20},
		Resizable: true, Undecorated: true, Floating: true, Hidden: true, Maximized: true, Unfocused: true,
		Transparent: true, TitlebarHidden: true, TrafficLights: &ggwd.Point{X: 14, Y: 16},
		MinWidth: 100, MaxHeight: 900,
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("WindowOptions: %+v, want %+v", got, want)
	}

	// Zero optional flags leave Position and TrafficLights nil.
	minimal := viewer.WindowOptions(&remote.OpenWindow{Title: "m", Width: 100, Height: 100, Decorated: true})
	if minimal.Position != nil || minimal.TrafficLights != nil {
		t.Errorf("minimal WindowOptions had non-nil Position/TrafficLights: %+v", minimal)
	}
}

func TestAppEvent(t *testing.T) {
	if got, ok := viewer.AppEvent(ggwd.QuitEvent{}); !ok || got.Kind != remote.AppEventQuit {
		t.Errorf("AppEvent(QuitEvent) = %+v, %t", got, ok)
	}
	if got, ok := viewer.AppEvent(ggwd.ReopenEvent{}); !ok || got.Kind != remote.AppEventReopen {
		t.Errorf("AppEvent(ReopenEvent) = %+v, %t", got, ok)
	}
	if _, ok := viewer.AppEvent(ggwd.FrameEvent{}); ok {
		t.Error("AppEvent(FrameEvent) should be false")
	}
}

func TestConfigNilWindow(t *testing.T) {
	cfg := viewer.Config(nil, 400, 300, 2, true)
	if cfg.Width != 400 || cfg.Height != 300 || cfg.PixelWidth != 800 || cfg.PixelHeight != 600 || !cfg.Focused || !cfg.Visible {
		t.Errorf("Config(nil): %+v", cfg)
	}
}

func TestResetAndColorMode(t *testing.T) {
	defer ggwd.SetPreferredColorMode(ggwd.ColorModeUnknown)
	viewer.SetColorMode(&remote.SetColorMode{Mode: uint8(ggwd.ColorModeDark)})
	if got := ggwd.PreferredColorMode(); got != ggwd.ColorModeDark {
		t.Fatalf("PreferredColorMode = %v, want Dark", got)
	}
	viewer.Reset(nil)
	if got := ggwd.PreferredColorMode(); got != ggwd.ColorModeUnknown {
		t.Fatalf("PreferredColorMode after Reset = %v, want Unknown", got)
	}
}
