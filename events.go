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
	"io/fs"

	"github.com/ironpark/ggfx"
	"github.com/ironpark/ggwd/internal/ui"
)

// Event is what a Handler receives. Every event but StartEvent, WakeEvent, SystemColorModeEvent
// and MonitorEvent belongs to a Window.
//
// Positions and sizes are in device-independent pixels (DIP), and the positions in input events
// are relative to the window's client area.
type Event interface {
	isEvent()
}

// StartEvent is the first event. The handler creates its windows here.
type StartEvent struct{}

// WakeEvent is delivered once for every call to [Wake]. It is for no window, and arrives
// whether or not any window draws.
type WakeEvent struct{}

// SystemColorModeEvent reports that the system's color mode, which [SystemColorMode] returns,
// changed. It is for no window. The change is noticed within a second.
type SystemColorModeEvent struct {
	Mode ColorMode
}

// MonitorEvent reports that a monitor was connected or disconnected, or changed its resolution,
// scale or arrangement. It is for no window; [AppendMonitors] lists the monitors there are now. A
// window whose monitor changed also gets a ResizeEvent when its scale changed. The browser does
// not report it.
type MonitorEvent struct{}

// FrameEvent asks the handler to draw Screen. Screen is in physical pixels; Scale converts
// device-independent pixels to physical pixels. Screen is valid during the event only.
type FrameEvent struct {
	Window *Window
	Screen *ggfx.Image
	Scale  float64
}

// ResizeEvent reports that the client area size or the scale changed. A window gets one right
// after it is created.
type ResizeEvent struct {
	Window *Window
	Width  float64
	Height float64
	Scale  float64
}

// MoveEvent reports that the window moved, whoever moved it. X and Y are its position on its
// monitor, as [Window.Position] returns it. A window gets one right after it is created. The
// browser does not report it.
type MoveEvent struct {
	Window *Window
	X      float64
	Y      float64
}

// FocusEvent reports that the window gained or lost the keyboard focus.
type FocusEvent struct {
	Window  *Window
	Focused bool
}

// WindowState is whether a window is minimized, maximized or fullscreen. A window that is two of
// them, such as a minimized maximized window, is the first of fullscreen, minimized and
// maximized.
type WindowState int

const (
	WindowStateNormal     WindowState = WindowState(ui.WindowStateNormal)
	WindowStateMinimized  WindowState = WindowState(ui.WindowStateMinimized)
	WindowStateMaximized  WindowState = WindowState(ui.WindowStateMaximized)
	WindowStateFullscreen WindowState = WindowState(ui.WindowStateFullscreen)
)

// WindowStateEvent reports that the window was minimized, maximized, made fullscreen or restored,
// whether the app, the user or the platform did it. A window that starts in a state other than
// WindowStateNormal reports it after its first ResizeEvent. The browser reports fullscreen alone.
type WindowStateEvent struct {
	Window *Window
	State  WindowState
}

// CloseEvent reports that the user asked to close the window. The window closes after the event
// is handled unless KeepOpen is called.
type CloseEvent struct {
	Window   *Window
	keepOpen func()
}

// KeepOpen keeps the window open.
func (c CloseEvent) KeepOpen() {
	c.keepOpen()
}

// ClosedEvent reports that the window closed, after a CloseEvent that did not keep it open or a
// [Window.Close]. It is the window's last event: its methods do nothing from now on. [Run] ends
// when no window is open after it is handled.
type ClosedEvent struct {
	Window *Window
}

// KeyEvent reports a key press or release. Repeat is set for the presses the OS generates while
// the key is held.
type KeyEvent struct {
	Window    *Window
	Key       Key
	Pressed   bool
	Repeat    bool
	Modifiers Mods
}

// TextEvent carries the characters the user typed.
type TextEvent struct {
	Window *Window
	Text   string
	// Replacement is the text that Text replaces, when a native IME replaces surrounding text
	// (for example the macOS accent menu), and nil otherwise.
	Replacement *TextRange
	// Passthrough reports that the IME committed the text and also passed
	// the key that typed it on, as a KeyEvent, rather than consuming it:
	// the macOS Korean IME commits the syllable it holds as Enter is
	// pressed, and Enter is still for the app.
	Passthrough bool
}

// TextRange is a range of text around the caret, in UTF-8 byte offsets relative to the caret in
// the context given to [Window.SetTextInputContext]: Start is negative before the caret.
type TextRange struct {
	Start, End int
}

// CompositionEvent reports IME marked text on macOS, Windows and X11. Start and End
// are UTF-8 byte offsets within Text. Done ends the composition and Text is empty;
// committed characters arrive separately as TextEvent.
type CompositionEvent struct {
	Window     *Window
	Text       string
	Start, End int
	Done       bool
}

// DragPhase identifies a stage of a file drag before or after the drop.
type DragPhase int

const (
	DragEntered DragPhase = iota
	DragMoved
	DragExited
	DragEnded
)

// DragEvent reports a native file drag on macOS and Windows. Files is nil when the platform
// cannot provide the paths. DropEvent delivers the files when the user drops them.
type DragEvent struct {
	Window *Window
	Phase  DragPhase
	X, Y   float64
	Files  fs.FS
}

// MouseMoveEvent reports the cursor position. Modifiers are the modifier keys held as the
// cursor moved.
type MouseMoveEvent struct {
	Window    *Window
	X         float64
	Y         float64
	Modifiers Mods
}

// MouseEnterEvent reports that the cursor entered the client area, or left it when Entered is
// false. A MouseMoveEvent follows an entry.
type MouseEnterEvent struct {
	Window  *Window
	Entered bool
}

// MouseButtonEvent reports a button press or release at the cursor position. Modifiers are the
// modifier keys held at the press, including one pressed while another window had the focus.
type MouseButtonEvent struct {
	Window    *Window
	Button    MouseButton
	Pressed   bool
	X         float64
	Y         float64
	Modifiers Mods
}

// ScrollEvent reports a wheel or trackpad scroll, in lines: a mouse wheel's notch is one on the
// desktop. A browser's scroll in pixels counts 20 device-independent pixels a line, and one in
// pages counts the canvas's height. Modifiers are the modifier keys held as the scroll happened.
type ScrollEvent struct {
	Window    *Window
	X         float64
	Y         float64
	Modifiers Mods
	// Phase is where the scroll is in the trackpad gesture making it, on
	// a platform that says (macOS): ScrollPhaseEnded is the fingers
	// lifting, which moves nothing. A wheel's scrolls have ScrollPhaseNone.
	Phase ScrollPhase
}

// ScrollPhase is where a scroll is in the trackpad gesture that makes it.
type ScrollPhase int

const (
	ScrollPhaseNone          ScrollPhase = ScrollPhase(ui.ScrollPhaseNone)          // a wheel, or no gesture reported
	ScrollPhaseBegan         ScrollPhase = ScrollPhase(ui.ScrollPhaseBegan)         // the fingers start to move
	ScrollPhaseChanged       ScrollPhase = ScrollPhase(ui.ScrollPhaseChanged)       // the fingers move on
	ScrollPhaseEnded         ScrollPhase = ScrollPhase(ui.ScrollPhaseEnded)         // the fingers lift
	ScrollPhaseMomentum      ScrollPhase = ScrollPhase(ui.ScrollPhaseMomentum)      // momentum after they lift
	ScrollPhaseMomentumEnded ScrollPhase = ScrollPhase(ui.ScrollPhaseMomentumEnded) // the momentum stops
)

// TouchPhase is what happened to a touch.
type TouchPhase int

const (
	TouchPhaseBegan TouchPhase = TouchPhase(ui.TouchPhaseBegan)
	TouchPhaseMoved TouchPhase = TouchPhase(ui.TouchPhaseMoved)
	TouchPhaseEnded TouchPhase = TouchPhase(ui.TouchPhaseEnded)
)

// TouchEvent reports a touch.
type TouchEvent struct {
	Window *Window
	ID     TouchID
	Phase  TouchPhase
	X      float64
	Y      float64
}

// DropEvent reports files dropped on the window.
type DropEvent struct {
	Window *Window
	Files  fs.FS
}

func (StartEvent) isEvent()           {}
func (WakeEvent) isEvent()            {}
func (SystemColorModeEvent) isEvent() {}
func (MonitorEvent) isEvent()         {}
func (FrameEvent) isEvent()           {}
func (ResizeEvent) isEvent()          {}
func (MoveEvent) isEvent()            {}
func (FocusEvent) isEvent()           {}
func (WindowStateEvent) isEvent()     {}
func (CloseEvent) isEvent()           {}
func (ClosedEvent) isEvent()          {}
func (KeyEvent) isEvent()             {}
func (TextEvent) isEvent()            {}
func (CompositionEvent) isEvent()     {}
func (DragEvent) isEvent()            {}
func (MouseMoveEvent) isEvent()       {}
func (MouseEnterEvent) isEvent()      {}
func (MouseButtonEvent) isEvent()     {}
func (ScrollEvent) isEvent()          {}
func (TouchEvent) isEvent()           {}
func (DropEvent) isEvent()            {}
