// SPDX-License-Identifier: Apache-2.0

package glfw

import (
	"math"
	"testing"
)

func TestDecodeDPIChangedWParam(t *testing.T) {
	tests := []struct {
		name       string
		xdpi, ydpi uint32
		wantX      float32
		wantY      float32
	}{
		{name: "100%", xdpi: 96, ydpi: 96, wantX: 1.0, wantY: 1.0},
		{name: "125%", xdpi: 120, ydpi: 120, wantX: 1.25, wantY: 1.25},
		{name: "150%", xdpi: 144, ydpi: 144, wantX: 1.5, wantY: 1.5},
		{name: "200%", xdpi: 192, ydpi: 192, wantX: 2.0, wantY: 2.0},
		{name: "asymmetric LOWORD=X HIWORD=Y", xdpi: 144, ydpi: 192, wantX: 1.5, wantY: 2.0},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			wParam := _WPARAM(tc.xdpi | (tc.ydpi << 16))
			gotXDPI, gotYDPI, gotX, gotY := decodeDPIChangedWParam(wParam)
			if gotXDPI != tc.xdpi || gotYDPI != tc.ydpi {
				t.Fatalf("dpi = (%d, %d); want (%d, %d)", gotXDPI, gotYDPI, tc.xdpi, tc.ydpi)
			}
			if gotX != tc.wantX || gotY != tc.wantY {
				t.Fatalf("scale = (%v, %v); want (%v, %v)", gotX, gotY, tc.wantX, tc.wantY)
			}
		})
	}
}

func TestDPIScaledWindowSize(t *testing.T) {
	tests := []struct {
		name                     string
		outerW, outerH           int32
		srcFrameW, srcFrameH     int32
		dstFrameW, dstFrameH     int32
		oldDPI, newDPI           uint32
		scaleToMonitor           bool
		wantOuterW, wantOuterH   int32
		wantClientW, wantClientH int32
	}{
		{
			name: "100% to 200% preserves logical DIP client size with non-linear frame",
			// 800x600 client + 16x39 frame at 96 DPI -> 1600x1200 client + 20x65 frame at 192 DPI
			outerW: 816, outerH: 639,
			srcFrameW: 16, srcFrameH: 39,
			dstFrameW: 20, dstFrameH: 65,
			oldDPI: 96, newDPI: 192,
			scaleToMonitor: true,
			wantOuterW:     1620, wantOuterH: 1265,
			wantClientW: 1600, wantClientH: 1200,
		},
		{
			name:   "200% to 100% round-trip preserves logical DIP client size",
			outerW: 1620, outerH: 1265,
			srcFrameW: 20, srcFrameH: 65,
			dstFrameW: 16, dstFrameH: 39,
			oldDPI: 192, newDPI: 96,
			scaleToMonitor: true,
			wantOuterW:     816, wantOuterH: 639,
			wantClientW: 800, wantClientH: 600,
		},
		{
			name:   "100% to 150% fractional scale rounds client size",
			outerW: 817, outerH: 640, // 801x601 client
			srcFrameW: 16, srcFrameH: 39,
			dstFrameW: 18, dstFrameH: 52,
			oldDPI: 96, newDPI: 144,
			scaleToMonitor: true,
			wantOuterW:     1220, wantOuterH: 954,
			wantClientW: 1202, wantClientH: 902,
		},
		{
			name:   "borderless window scales client size cleanly",
			outerW: 800, outerH: 600,
			srcFrameW: 0, srcFrameH: 0,
			dstFrameW: 0, dstFrameH: 0,
			oldDPI: 96, newDPI: 144,
			scaleToMonitor: true,
			wantOuterW:     1200, wantOuterH: 900,
			wantClientW: 1200, wantClientH: 900,
		},
		{
			name:   "scaleToMonitor false preserves physical client size",
			outerW: 816, outerH: 639,
			srcFrameW: 16, srcFrameH: 39,
			dstFrameW: 20, dstFrameH: 65,
			oldDPI: 96, newDPI: 192,
			scaleToMonitor: false,
			wantOuterW:     820, wantOuterH: 665,
			wantClientW: 800, wantClientH: 600,
		},
		{
			name:   "zero oldDPI falls back to frame delta only",
			outerW: 816, outerH: 639,
			srcFrameW: 16, srcFrameH: 39,
			dstFrameW: 20, dstFrameH: 65,
			oldDPI: 0, newDPI: 192,
			scaleToMonitor: true,
			wantOuterW:     820, wantOuterH: 665,
			wantClientW: 800, wantClientH: 600,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			gotW, gotH := dpiScaledWindowSize(
				tc.outerW, tc.outerH,
				tc.srcFrameW, tc.srcFrameH,
				tc.dstFrameW, tc.dstFrameH,
				tc.oldDPI, tc.newDPI,
				tc.scaleToMonitor,
			)
			if gotW != tc.wantOuterW || gotH != tc.wantOuterH {
				t.Fatalf("outer = (%d, %d); want (%d, %d)", gotW, gotH, tc.wantOuterW, tc.wantOuterH)
			}
			gotClientW, gotClientH := gotW-tc.dstFrameW, gotH-tc.dstFrameH
			if gotClientW != tc.wantClientW || gotClientH != tc.wantClientH {
				t.Fatalf("client = (%d, %d); want (%d, %d)", gotClientW, gotClientH, tc.wantClientW, tc.wantClientH)
			}
		})
	}
}

func TestScaleLimitForDPI(t *testing.T) {
	tests := []struct {
		name           string
		limit          int
		oldDPI, newDPI uint32
		want           int
	}{
		{name: "DontCare preserved", limit: DontCare, oldDPI: 96, newDPI: 192, want: DontCare},
		{name: "100% to 200%", limit: 400, oldDPI: 96, newDPI: 192, want: 800},
		{name: "200% to 100%", limit: 800, oldDPI: 192, newDPI: 96, want: 400},
		{name: "100% to 150%", limit: 301, oldDPI: 96, newDPI: 144, want: 452},
		{name: "same DPI", limit: 500, oldDPI: 144, newDPI: 144, want: 500},
		{name: "zero oldDPI", limit: 500, oldDPI: 0, newDPI: 144, want: 500},
		{name: "clamps to at least 1", limit: 1, oldDPI: 288, newDPI: 96, want: 1},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := scaleLimitForDPI(tc.limit, tc.oldDPI, tc.newDPI); got != tc.want {
				t.Fatalf("scaleLimitForDPI(%d, %d, %d) = %d; want %d", tc.limit, tc.oldDPI, tc.newDPI, got, tc.want)
			}
		})
	}
}

func TestLogicalPageHeight(t *testing.T) {
	tests := []struct {
		name        string
		pixelHeight int
		scale       float32
		want        float64
	}{
		{name: "100%", pixelHeight: 800, scale: 1.0, want: 800},
		{name: "125%", pixelHeight: 1000, scale: 1.25, want: 800},
		{name: "150%", pixelHeight: 1200, scale: 1.5, want: 800},
		{name: "200%", pixelHeight: 1600, scale: 2.0, want: 800},
		{name: "zero scale falls back to pixel height", pixelHeight: 800, scale: 0, want: 800},
		{name: "negative scale falls back to pixel height", pixelHeight: 800, scale: -1.5, want: 800},
		{name: "NaN scale falls back to pixel height", pixelHeight: 800, scale: float32(math.NaN()), want: 800},
		{name: "Inf scale falls back to pixel height", pixelHeight: 800, scale: float32(math.Inf(1)), want: 800},
		{name: "non-positive height returns 0", pixelHeight: 0, scale: 2.0, want: 0},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := logicalPageHeight(tc.pixelHeight, tc.scale)
			if got != tc.want {
				t.Fatalf("logicalPageHeight(%d, %v) = %v; want %v", tc.pixelHeight, tc.scale, got, tc.want)
			}
		})
	}

	// Verify WHEEL_PAGESCROLL produces the same logical scroll line at 200% DPI as at 100% DPI.
	lines1x, line1x, _ := wheelNotch(120, wheelPageScroll, logicalPageHeight(800, 1.0))
	lines2x, line2x, _ := wheelNotch(120, wheelPageScroll, logicalPageHeight(1600, 2.0))
	if lines1x != lines2x || line1x != line2x || line2x != 700 {
		t.Fatalf("WHEEL_PAGESCROLL mismatch: 1x=(%v, %v), 2x=(%v, %v); want line=700", lines1x, line1x, lines2x, line2x)
	}
}
