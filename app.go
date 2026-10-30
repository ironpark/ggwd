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

package ggwd

import (
	"errors"

	"github.com/ironpark/ggwd/internal/ui"
)

// Handler receives the events of an app run by Run.
type Handler interface {
	// HandleEvent handles one event. Returning Termination ends Run without an error.
	HandleEvent(ev Event) error
}

// HandlerFunc is a Handler made of one function.
type HandlerFunc func(ev Event) error

// HandleEvent implements Handler.
func (f HandlerFunc) HandleEvent(ev Event) error {
	return f(ev)
}

// Run starts the event loop and blocks until it ends. The handler receives StartEvent first and
// creates its windows there with NewWindow. Run ends when the handler returns Termination or an
// error, or when no window is open after the events are handled, like the ClosedEvent of the
// last window.
//
// Run must be called on the main thread, and only once in a process.
func Run(h Handler, options *RunOptions) error {
	op := toUIRunOptions(options)

	if err := ui.Get().RunApp(&appForUI{handler: h}, op); err != nil {
		if errors.Is(err, Termination) {
			return nil
		}
		return err
	}
	return nil
}

// appForUI adapts a Handler to ui.App.
type appForUI struct {
	handler Handler
}

func (a *appForUI) HandleEvent(ev ui.Event) error {
	return a.handler.HandleEvent(eventFromUI(ev))
}

func windowFromUI(w ui.AppWindow) *Window {
	if w == nil {
		return nil
	}
	return w.Handle().(*Window)
}

func eventFromUI(ev ui.Event) Event {
	switch ev := ev.(type) {
	case ui.StartEvent:
		return StartEvent{}
	case ui.WakeEvent:
		return WakeEvent{}
	case ui.SystemColorModeEvent:
		return SystemColorModeEvent{Mode: ColorMode(ev.Mode)}
	case ui.MonitorEvent:
		return MonitorEvent{}
	case ui.FrameEvent:
		return FrameEvent{Window: windowFromUI(ev.Window), Screen: ev.Screen, Scale: ev.Scale}
	case ui.ResizeEvent:
		return ResizeEvent{Window: windowFromUI(ev.Window), Width: ev.Width, Height: ev.Height, Scale: ev.Scale}
	case ui.MoveEvent:
		return MoveEvent{Window: windowFromUI(ev.Window), X: ev.X, Y: ev.Y}
	case ui.FocusEvent:
		return FocusEvent{Window: windowFromUI(ev.Window), Focused: ev.Focused}
	case ui.WindowStateEvent:
		return WindowStateEvent{Window: windowFromUI(ev.Window), State: WindowState(ev.State)}
	case ui.CloseEvent:
		return CloseEvent{Window: windowFromUI(ev.Window), keepOpen: ev.KeepOpen}
	case ui.ClosedEvent:
		return ClosedEvent{Window: windowFromUI(ev.Window)}
	case ui.KeyEvent:
		return KeyEvent{Window: windowFromUI(ev.Window), Key: Key(ev.Key), Pressed: ev.Pressed, Repeat: ev.Repeat, Modifiers: Mods(ev.Modifiers)}
	case ui.TextEvent:
		var r *TextRange
		if ev.HasReplacement {
			r = &TextRange{Start: ev.ReplacementStart, End: ev.ReplacementEnd}
		}
		return TextEvent{Window: windowFromUI(ev.Window), Text: ev.Text, Replacement: r, Passthrough: ev.Passthrough}
	case ui.CompositionEvent:
		return CompositionEvent{Window: windowFromUI(ev.Window), Text: ev.Text, Start: ev.Start, End: ev.End, Done: ev.Done}
	case ui.DragEvent:
		return DragEvent{Window: windowFromUI(ev.Window), Phase: DragPhase(ev.Phase), X: ev.X, Y: ev.Y, Files: ev.Files}
	case ui.MouseMoveEvent:
		return MouseMoveEvent{Window: windowFromUI(ev.Window), X: ev.X, Y: ev.Y, Modifiers: Mods(ev.Modifiers)}
	case ui.MouseEnterEvent:
		return MouseEnterEvent{Window: windowFromUI(ev.Window), Entered: ev.Entered}
	case ui.MouseButtonEvent:
		return MouseButtonEvent{Window: windowFromUI(ev.Window), Button: MouseButton(ev.Button), Pressed: ev.Pressed, X: ev.X, Y: ev.Y, Modifiers: Mods(ev.Modifiers)}
	case ui.ScrollEvent:
		return ScrollEvent{Window: windowFromUI(ev.Window), X: ev.X, Y: ev.Y, Modifiers: Mods(ev.Modifiers), Phase: ScrollPhase(ev.Phase)}
	case ui.TouchEvent:
		return TouchEvent{Window: windowFromUI(ev.Window), ID: TouchID(ev.ID), Phase: TouchPhase(ev.Phase), X: ev.X, Y: ev.Y}
	case ui.DropEvent:
		return DropEvent{Window: windowFromUI(ev.Window), Files: ev.Files}
	default:
		panic("ggwd: unknown ui event")
	}
}
