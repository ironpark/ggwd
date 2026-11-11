package glfw

import "testing"

func TestWheelNotchScrollsWhatTheSystemIsSetTo(t *testing.T) {
	for _, tc := range []struct {
		name        string
		delta       int16
		setting     uint32
		page        float64
		lines, line float64
		precise     bool
	}{
		{name: "a notch as Windows ships", delta: 120, setting: 3, lines: 1, line: 100},
		{name: "two notches back", delta: -240, setting: 3, lines: -2, line: 100},
		{name: "a notch set to six lines", delta: 120, setting: 6, lines: 1, line: 200},
		{name: "a wheel turned off", delta: 120, setting: 0, lines: 1, line: 0},
		{name: "a notch set to a page", delta: 120, setting: wheelPageScroll, page: 800, lines: 1, line: 700},
		{name: "a precision touchpad", delta: 30, setting: 3, lines: 2.5, line: PreciseLine, precise: true},
		{name: "a free wheel, back", delta: -12, setting: 3, lines: -1, line: PreciseLine, precise: true},
	} {
		lines, line, precise := wheelNotch(tc.delta, tc.setting, tc.page)
		if lines != tc.lines || line != tc.line || precise != tc.precise {
			t.Errorf("%s: %v lines of %v, precise %v; want %v of %v, %v", tc.name, lines, line, precise, tc.lines, tc.line, tc.precise)
		}
	}
}
