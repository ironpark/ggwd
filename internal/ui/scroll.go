package ui

import "math"

// preciseLine is the points of a precise scroll that make a line:
// ggwd.PreciseLine.
const preciseLine = 10

// What a browser's wheel event scrolls by where it does not say in pixels.
const (
	// browserNotch is the pixels a wheel's notch scrolls a page, in Chrome,
	// Edge and Safari on Windows and Linux as they ship.
	browserNotch = 100
	// browserNotchLines is the lines a notch is where a browser counts a
	// wheel in lines, as Firefox does and ships with.
	browserNotchLines = 3
	// browserPageShare is how much of a page a scroll counted in pages
	// moves, leaving a little of the last in view.
	browserPageShare = 0.875
	// leastNotch is the least a wheel's notch scrolls, in pixels, for a
	// scroll in pixels to be taken as a wheel's: a trackpad's scroll that
	// happens to count as whole notches moves less for each.
	leastNotch = 50
)

// The deltaMode of a browser's wheel event.
const (
	domDeltaPixel = 0
	domDeltaLine  = 1
	domDeltaPage  = 2
)

// browserScroll turns a browser's wheel event into a scroll: dx and dy are
// its deltaX and deltaY in the unit mode, its deltaMode, says; wx and wy its
// wheelDeltaX and wheelDeltaY, which count a notch of a wheel as 120 and are
// zero where the browser has none; page is the height of the page in
// pixels. It returns the lines scrolled, positive up and to the left as
// Wheel reports them, how many pixels a line moves, and whether the scroll
// is precise.
//
// A browser reports a wheel and a trackpad alike, in pixels, and has
// already made of either the distance its pages scroll: that distance is
// kept. What tells them apart is that a wheel turns by whole notches: a
// scroll whose wheelDelta is whole notches of at least leastNotch pixels
// each is a wheel's, counted in notches, and any other is precise, the
// trackpad's own movement and the momentum the system adds to it.
func browserScroll(dx, dy float64, mode int, wx, wy, page float64) (x, y, line float64, precise bool) {
	switch mode {
	case domDeltaLine:
		return -dx / browserNotchLines, -dy / browserNotchLines, browserNotch, false
	case domDeltaPage:
		line = page * browserPageShare
		if page <= 0 {
			line = 4 * browserNotch
		}
		return -dx, -dy, line, false
	}
	// The axis that moved most says what kind of device it is.
	d, w := dy, wy
	if math.Abs(dx) > math.Abs(dy) {
		d, w = dx, wx
	}
	if notches := math.Abs(w) / 120; w != 0 && notches == math.Trunc(notches) && math.Abs(d)/notches >= leastNotch {
		line = math.Abs(d) / notches
		return -dx / line, -dy / line, line, false
	}
	return -dx / preciseLine, -dy / preciseLine, preciseLine, true
}
