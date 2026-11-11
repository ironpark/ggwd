package ui

import "testing"

func TestBrowserScrollKeepsTheBrowsersDistanceAndTellsAWheelFromATrackpad(t *testing.T) {
	for _, tc := range []struct {
		name           string
		dx, dy         float64
		mode           int
		wx, wy, page   float64
		y, line, moved float64 // moved is how far down the view scrolls
		precise        bool
	}{
		{name: "a notch in Chrome on Windows", dy: 100, wy: -120, y: -1, line: 100, moved: 100},
		{name: "three notches at once", dy: 300, wy: -360, y: -3, line: 100, moved: 300},
		{name: "a notch with the system set to six lines", dy: 200, wy: -120, y: -1, line: 200, moved: 200},
		{name: "a notch with the system set to one line", dy: 100.0 / 3, wy: -120, y: -100.0 / 3 / 10, line: 10, moved: 100.0 / 3, precise: true},
		{name: "a trackpad", dy: 7.5, wy: -22.5, y: -0.75, line: 10, moved: 7.5, precise: true},
		{name: "a trackpad on macOS that counts as a whole notch", dy: 40, wy: -120, y: -4, line: 10, moved: 40, precise: true},
		{name: "a browser with no wheelDelta", dy: 100, y: -10, line: 10, moved: 100, precise: true},
		{name: "a notch in Firefox, in lines", dy: 3, mode: domDeltaLine, y: -1, line: 100, moved: 100},
		{name: "a page", dy: 1, mode: domDeltaPage, page: 800, y: -1, line: 700, moved: 700},
		{name: "a page of no height", dy: 1, mode: domDeltaPage, y: -1, line: 400, moved: 400},
	} {
		_, y, line, precise := browserScroll(tc.dx, tc.dy, tc.mode, tc.wx, tc.wy, tc.page)
		if !near(y, tc.y) || !near(line, tc.line) || precise != tc.precise || !near(-y*line, tc.moved) {
			t.Errorf("%s: %v lines of %v, precise %v, moving %v; want %v of %v, %v, moving %v",
				tc.name, y, line, precise, -y*line, tc.y, tc.line, tc.precise, tc.moved)
		}
	}
}

// The axis that moved most decides, and both are counted in its lines.
func TestBrowserScrollCountsBothAxesInTheLinesOfTheOneThatMovedMost(t *testing.T) {
	x, y, line, precise := browserScroll(200, 10, domDeltaPixel, -240, -12, 0)
	if precise || !near(line, 100) || !near(x, -2) || !near(y, -0.1) {
		t.Fatalf("%v, %v lines of %v, precise %v; want -2, -0.1 of 100, a wheel's", x, y, line, precise)
	}
}

func near(a, b float64) bool { return a-b < 1e-9 && b-a < 1e-9 }
