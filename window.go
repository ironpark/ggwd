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
	"image"
	"math"

	"github.com/ironpark/ggwd/internal/ui"
	"github.com/ironpark/ggwd/internal/windowbridge"
)

// Point is a position in device-independent pixels.
type Point struct {
	X, Y float64
}

// WindowOptions describes a window to create with NewWindow. The zero value is a decorated,
// visible, fixed-size 640x480 window centered on the monitor.
//
// Sizes and positions are in device-independent pixels. The platform's window has whole ones,
// so they are rounded.
type WindowOptions struct {
	Title string

	// Width and Height are the client area size. 0 uses 640x480.
	Width  float64
	Height float64

	// Position is the position on the monitor. nil centers the window.
	Position *Point

	Resizable   bool
	Undecorated bool
	Floating    bool
	Hidden      bool
	Maximized   bool
	Unfocused   bool

	// Transparent makes the window's framebuffer composite with what is behind the window.
	// DirectX cannot present such a surface, so NewWindow fails with it on a Windows machine
	// that chose the DirectX driver.
	Transparent bool

	// The size limits of the client area. 0 means no limit.
	MinWidth  float64
	MinHeight float64
	MaxWidth  float64
	MaxHeight float64
}

// Window is a window created by NewWindow. Its methods can be called from any goroutine, and do
// nothing once the window is closed (see [ClosedEvent]).
//
// Sizes and positions are in device-independent pixels. The platform's window has whole ones,
// so the ones given are rounded.
type Window struct {
	ui ui.AppWindow
}

func init() {
	windowbridge.UI = func(w any) ui.AppWindow { return w.(*Window).ui }
}

// NewWindow creates a window. It can be called only while Run runs, from the handler.
func NewWindow(o *WindowOptions) (*Window, error) {
	if o == nil {
		o = &WindowOptions{}
	}
	uo := &ui.WindowOptions{
		Title:       o.Title,
		Width:       dip(o.Width),
		Height:      dip(o.Height),
		Decorated:   !o.Undecorated,
		Resizable:   o.Resizable,
		Floating:    o.Floating,
		Visible:     !o.Hidden,
		Maximized:   o.Maximized,
		Transparent: o.Transparent,
		Unfocused:   o.Unfocused,
		MinWidth:    dip(o.MinWidth),
		MinHeight:   dip(o.MinHeight),
		MaxWidth:    dip(o.MaxWidth),
		MaxHeight:   dip(o.MaxHeight),
	}
	if o.Position != nil {
		uo.X, uo.Y, uo.PositionSet = dip(o.Position.X), dip(o.Position.Y), true
	}
	w := &Window{}
	uw, err := ui.Get().NewWindow(uo, w)
	if err != nil {
		return nil, err
	}
	w.ui = uw
	return w, nil
}

// dip rounds a length in device-independent pixels to the whole one the platform's window has.
func dip(v float64) int {
	return int(math.Round(v))
}

// sizeLimit converts a size limit to the window system's, where a negative one is no limit.
func sizeLimit(v float64) int {
	if v <= 0 {
		return -1
	}
	return dip(v)
}

// Close closes the window after the current event is handled, without a CloseEvent. A
// ClosedEvent follows.
func (w *Window) Close() {
	w.ui.Close()
}

// RequestFrame asks for one FrameEvent. Frames are not delivered otherwise, except after a
// resize and when the window is first shown.
func (w *Window) RequestFrame() {
	w.ui.RequestFrame()
}

// SetTitle sets the title.
func (w *Window) SetTitle(title string) {
	w.ui.SetTitle(title)
}

// Size returns the client area size.
func (w *Window) Size() (width, height float64) {
	ww, wh := w.ui.Size()
	return float64(ww), float64(wh)
}

// SetSize sets the client area size.
func (w *Window) SetSize(width, height float64) {
	w.ui.SetSize(dip(width), dip(height))
}

// Position returns the position on the monitor.
func (w *Window) Position() (x, y float64) {
	px, py := w.ui.Position()
	return float64(px), float64(py)
}

// SetPosition sets the position on the monitor.
func (w *Window) SetPosition(x, y float64) {
	w.ui.SetPosition(dip(x), dip(y))
}

// SetSizeLimits sets the size limits of the client area. 0 means no limit, as in
// [WindowOptions].
func (w *Window) SetSizeLimits(minWidth, minHeight, maxWidth, maxHeight float64) {
	w.ui.SetSizeLimits(sizeLimit(minWidth), sizeLimit(minHeight), sizeLimit(maxWidth), sizeLimit(maxHeight))
}

// SetResizable sets whether the user can resize the window.
func (w *Window) SetResizable(resizable bool) {
	mode := ui.WindowResizingModeDisabled
	if resizable {
		mode = ui.WindowResizingModeEnabled
	}
	w.ui.SetResizingMode(mode)
}

// Show shows the window.
func (w *Window) Show() {
	w.ui.SetVisible(true)
}

// Hide hides the window.
func (w *Window) Hide() {
	w.ui.SetVisible(false)
}

// IsVisible reports whether the window is shown.
func (w *Window) IsVisible() bool {
	return w.ui.IsVisible()
}

// SetDecorated sets whether the window has a frame.
func (w *Window) SetDecorated(decorated bool) {
	w.ui.SetDecorated(decorated)
}

// SetFloating sets whether the window stays above the others.
func (w *Window) SetFloating(floating bool) {
	w.ui.SetFloating(floating)
}

// Maximize maximizes the window. A window that is not resizable, or that has a maximum size,
// cannot be maximized.
func (w *Window) Maximize() {
	w.ui.Maximize()
}

// IsMaximized reports whether the window is maximized.
func (w *Window) IsMaximized() bool {
	return w.ui.IsMaximized()
}

// Minimize minimizes the window.
func (w *Window) Minimize() {
	w.ui.Minimize()
}

// IsMinimized reports whether the window is minimized.
func (w *Window) IsMinimized() bool {
	return w.ui.IsMinimized()
}

// Restore restores a maximized or minimized window.
func (w *Window) Restore() {
	w.ui.Restore()
}

// SetFullscreen sets whether the window is fullscreen.
func (w *Window) SetFullscreen(fullscreen bool) {
	w.ui.SetFullscreen(fullscreen)
}

// IsFullscreen reports whether the window is fullscreen.
func (w *Window) IsFullscreen() bool {
	return w.ui.IsFullscreen()
}

// IsFocused reports whether the window has the keyboard focus.
func (w *Window) IsFocused() bool {
	return w.ui.IsFocused()
}

// Focus gives the window the keyboard focus.
func (w *Window) Focus() {
	w.ui.Focus()
}

// RequestAttention asks the user's attention, like bouncing the dock icon.
func (w *Window) RequestAttention() {
	w.ui.RequestAttention()
}

// SetIcon sets the window icon on the platforms that show one. A window in fullscreen gets it
// when it leaves fullscreen.
func (w *Window) SetIcon(iconImages []image.Image) {
	w.ui.SetIcon(iconImages)
}

// SetMousePassthrough sets whether mouse events go through the window.
func (w *Window) SetMousePassthrough(enabled bool) {
	w.ui.SetMousePassthrough(enabled)
}

// DeviceScaleFactor returns the scale from device-independent pixels to physical pixels on the
// window's monitor.
func (w *Window) DeviceScaleFactor() float64 {
	return w.ui.DeviceScaleFactor()
}

// Monitor returns the monitor the window is on, or nil once the window is closed.
func (w *Window) Monitor() *Monitor {
	return (*Monitor)(w.ui.Monitor())
}

// CursorShape returns the cursor shape over the window.
func (w *Window) CursorShape() CursorShape {
	return CursorShape(w.ui.CursorShape())
}

// SetCursorShape sets the cursor shape over the window.
func (w *Window) SetCursorShape(shape CursorShape) {
	w.ui.SetCursorShape(ui.CursorShape(shape))
}

// CursorMode returns the cursor mode.
func (w *Window) CursorMode() CursorMode {
	return CursorMode(w.ui.CursorMode())
}

// SetCursorMode sets the cursor mode.
func (w *Window) SetCursorMode(mode CursorMode) {
	w.ui.SetCursorMode(ui.CursorMode(mode))
}

// SetTextInputEnabled enables native IME input on macOS, Windows and X11. Disabling
// it cancels marked text without removing the window's keyboard focus.
// It has no effect in the browser. Call it when an editable field gains or
// loses focus, not on each composition commit.
func (w *Window) SetTextInputEnabled(enabled bool) { w.ui.SetTextInputEnabled(enabled) }

// SetTextInputRect positions the IME candidate panel at the caret rectangle in
// the client area. It has no effect in the browser.
func (w *Window) SetTextInputRect(x, y, width, height float64) {
	w.ui.SetTextInputRect(x, y, width, height)
}

// SetTextInputContext supplies the text before and after the editable selection
// for native IME replacement ranges ([TextEvent.Replacement]). Update it when the
// caret or text changes.
func (w *Window) SetTextInputContext(before, after string) { w.ui.SetTextInputContext(before, after) }
