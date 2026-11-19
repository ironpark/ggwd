// Copyright 2026 The ggwd Authors
// SPDX-License-Identifier: Apache-2.0

package glfw

import (
	"github.com/ebitengine/purego/objc"

	"github.com/ironpark/ggwd/internal/cocoa"
)

// A window whose title bar is hidden keeps its frame, its shadow and its traffic lights, but its
// content runs under where the title bar was: the bar is transparent and has no title. The app
// draws its own header there, places the traffic lights to sit in it, and moves the window by it
// with StartDrag.

const nsWindowStyleMaskFullSizeContentView = 1 << 15

var (
	sel_setTitlebarAppearsTransparent = objc.RegisterName("setTitlebarAppearsTransparent:")
	sel_setTitleVisibility            = objc.RegisterName("setTitleVisibility:")
	sel_standardWindowButton          = objc.RegisterName("standardWindowButton:")
	sel_superview                     = objc.RegisterName("superview")
	sel_pressedMouseButtons           = objc.RegisterName("pressedMouseButtons")
	sel_performWindowDragWithEvent    = objc.RegisterName("performWindowDragWithEvent:")
	sel_standardUserDefaults          = objc.RegisterName("standardUserDefaults")
	sel_stringForKey                  = objc.RegisterName("stringForKey:")
)

// hideTitlebar makes the title bar of a window made with the full-size content view
// transparent, with no title.
func (w *Window) hideTitlebar() {
	w.platform.titlebarHidden = true
	w.platform.object.Send(sel_setTitlebarAppearsTransparent, true)
	w.platform.object.Send(sel_setTitleVisibility, 1) // NSWindowTitleHidden
}

// SetTrafficLights puts the close button's top-left corner x and y points from the top-left
// corner of a window whose title bar is hidden, the other two buttons following it as AppKit
// spaces them. It must be called from the main thread.
func (w *Window) SetTrafficLights(x, y float64) {
	if !w.platform.titlebarHidden {
		return
	}
	w.platform.trafficLights = &[2]float64{x, y}
	w.placeTrafficLights()
}

// placeTrafficLights puts the traffic lights where SetTrafficLights asked, again: AppKit lays
// them out afresh as the window resizes, goes in and out of full screen, and becomes key. The
// title bar's container is made tall enough to hold them there, as they are drawn in it.
func (w *Window) placeTrafficLights() {
	at := w.platform.trafficLights
	if at == nil || w.platform.object == 0 {
		return
	}
	window := w.platform.object
	var buttons [3]objc.ID
	for i := range buttons {
		buttons[i] = window.Send(sel_standardWindowButton, i) // close, miniaturize, zoom
		if buttons[i] == 0 {
			return
		}
	}
	container := buttons[0].Send(sel_superview).Send(sel_superview)
	if container == 0 {
		return
	}
	closeFrame := cocoa.Send[cocoa.NSRect](buttons[0], sel_frame)
	miniFrame := cocoa.Send[cocoa.NSRect](buttons[1], sel_frame)
	windowFrame := cocoa.Send[cocoa.NSRect](window, sel_frame)
	bar := cocoa.Send[cocoa.NSRect](container, sel_frame)
	bar.Size.Height = closeFrame.Size.Height + at[1]
	bar.Origin.Y = windowFrame.Size.Height - bar.Size.Height
	container.Send(sel_setFrame, bar)
	space := miniFrame.Origin.X - closeFrame.Origin.X
	for i, b := range buttons {
		b.Send(sel_setFrameOrigin, cocoa.NSPoint{X: at[0] + float64(i)*space, Y: 0})
	}
}

// StartDrag moves the window with the pointer from the last press of its left button in the
// content, while the button is still down, as a press on the title bar would. It must be called
// from the main thread, and returns once the drag is over; the release that ends it, which the
// drag takes, is reported to the window as a release.
func (w *Window) StartDrag() {
	down := w.platform.lastMouseDown
	if down == 0 {
		return
	}
	if objc.ID(objc.GetClass("NSEvent")).Send(sel_pressedMouseButtons)&1 == 0 {
		return
	}
	w.platform.object.Send(sel_performWindowDragWithEvent, down)
	if objc.ID(objc.GetClass("NSEvent")).Send(sel_pressedMouseButtons)&1 == 0 {
		w.inputMouseClick(MouseButton1, Release, 0)
	}
}

// sel_performSelector_withObject_afterDelay runs a selector from the run loop.
var sel_performSelector_withObject_afterDelay = objc.RegisterName("performSelector:withObject:afterDelay:")

// TitlebarDoubleClick does what a double-click on the title bar does in the system's settings:
// zoom the window, minimize it, or nothing. It must be called from the main thread.
func (w *Window) TitlebarDoubleClick() {
	defaults := objc.ID(objc.GetClass("NSUserDefaults")).Send(sel_standardUserDefaults)
	key := cocoa.NSString_alloc().InitWithUTF8String("AppleActionOnDoubleClick")
	defer key.ID.Send(sel_release)
	action := ""
	if s := defaults.Send(sel_stringForKey, key.ID); s != 0 {
		action = cocoa.NSString{ID: s}.String()
	}
	switch action {
	case "None":
	case "Minimize":
		w.platform.object.Send(sel_miniaturize, 0)
	default:
		// zoom: animates the frame before it returns, a resize at each step. Called here, from a
		// call on the main thread, it would hold the thread with no frame drawn at any step, and
		// the last frame stretched to the window until the end. Run from the run loop instead,
		// in the next event polling, it resizes as a drag on the frame does, a frame at each step.
		cocoa.Send[objc.ID](w.platform.object, sel_performSelector_withObject_afterDelay, sel_zoom, objc.ID(0), float64(0))
	}
}

// noteMouseDown keeps the press of the left button in the content, for StartDrag.
func (w *Window) noteMouseDown(event objc.ID) {
	if w.platform.lastMouseDown != 0 {
		w.platform.lastMouseDown.Send(sel_release)
	}
	w.platform.lastMouseDown = event.Send(sel_retain)
}
