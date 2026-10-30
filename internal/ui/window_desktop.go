// Copyright 2026 The Ebitengine Authors
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

//go:build !android && !ios && !js && !nintendosdk && !playstation5

package ui

import (
	"image"
	"sync/atomic"

	"github.com/ironpark/ggwd/internal/glfw"
	"github.com/ironpark/ggwd/internal/microsoftgdk"
)

type windowSizeRange struct {
	minWidthInDIP  int
	minHeightInDIP int
	maxWidthInDIP  int
	maxHeightInDIP int
}

// windowSettings are the settings of one window: what NewWindow was given, and what the app set
// since that the platform's window does not keep for it. The app's goroutine stores them and the
// main thread reads them, so they are atomic.
type windowSettings struct {
	title atomic.Value

	windowSizeLimit atomic.Pointer[windowSizeRange]

	iconImages         atomic.Pointer[[]image.Image]
	windowResizingMode atomic.Int32

	initWindowDecorated        atomic.Bool
	initWindowVisible          atomic.Bool
	initWindowPositionInDIP    atomic.Pointer[image.Point]
	initWindowSizeInDIP        atomic.Pointer[image.Point]
	initWindowFloating         atomic.Bool
	initWindowTransparent      atomic.Bool
	initWindowUnfocused        atomic.Bool
	initWindowMaximized        atomic.Bool
	initWindowMousePassthrough atomic.Bool
}

// newWindowSettings returns the settings of a window created with o.
func newWindowSettings(o *WindowOptions) *windowSettings {
	s := &windowSettings{}
	s.title.Store(o.Title)
	p := image.Pt(640, 480)
	if o.Width > 0 && o.Height > 0 {
		p = image.Pt(o.Width, o.Height)
	}
	s.initWindowSizeInDIP.Store(&p)
	if o.PositionSet {
		s.setInitWindowPositionInDIP(o.X, o.Y)
	}
	s.initWindowDecorated.Store(o.Decorated)
	s.initWindowFloating.Store(o.Floating && !microsoftgdk.IsXbox())
	s.initWindowVisible.Store(o.Visible)
	s.initWindowMaximized.Store(o.Maximized)
	s.initWindowTransparent.Store(o.Transparent)
	s.initWindowUnfocused.Store(o.Unfocused)
	if o.Resizable {
		s.windowResizingMode.Store(int32(WindowResizingModeEnabled))
	}
	limit := func(v int) int {
		if v > 0 {
			return v
		}
		return glfw.DontCare
	}
	s.setWindowSizeLimitsInDIP(limit(o.MinWidth), limit(o.MinHeight), limit(o.MaxWidth), limit(o.MaxHeight))
	return s
}

func (s *windowSettings) getWindowSizeLimitsInDIP() (minw, minh, maxw, maxh int) {
	if microsoftgdk.IsXbox() {
		return glfw.DontCare, glfw.DontCare, glfw.DontCare, glfw.DontCare
	}

	r := s.windowSizeLimit.Load()
	return r.minWidthInDIP, r.minHeightInDIP, r.maxWidthInDIP, r.maxHeightInDIP
}

// setWindowSizeLimitsInDIP stores the size limits and reports whether they changed.
func (s *windowSettings) setWindowSizeLimitsInDIP(minw, minh, maxw, maxh int) bool {
	if microsoftgdk.IsXbox() {
		// Do nothing. The size is always fixed.
		return false
	}

	newS := &windowSizeRange{
		minWidthInDIP:  minw,
		minHeightInDIP: minh,
		maxWidthInDIP:  maxw,
		maxHeightInDIP: maxh,
	}
	old := s.windowSizeLimit.Swap(newS)
	return old == nil || *old != *newS
}

func (s *windowSettings) isWindowMaximizable() bool {
	_, _, maxw, maxh := s.getWindowSizeLimitsInDIP()
	return maxw == glfw.DontCare && maxh == glfw.DontCare
}

// adjustWindowSizeBasedOnSizeLimitsInDIP adjust the size based on the window size limits.
// width and height are in device-independent pixels.
func (s *windowSettings) adjustWindowSizeBasedOnSizeLimitsInDIP(width, height int) (int, int) {
	minw, minh, maxw, maxh := s.getWindowSizeLimitsInDIP()
	if minw >= 0 && width < minw {
		width = minw
	}
	if minh >= 0 && height < minh {
		height = minh
	}
	if maxw >= 0 && width > maxw {
		width = maxw
	}
	if maxh >= 0 && height > maxh {
		height = maxh
	}
	return width, height
}

func (s *windowSettings) resizingMode() WindowResizingMode {
	return WindowResizingMode(s.windowResizingMode.Load())
}

func (s *windowSettings) isInitWindowDecorated() bool {
	return s.initWindowDecorated.Load()
}

func (s *windowSettings) isInitWindowVisible() bool {
	return s.initWindowVisible.Load()
}

func (s *windowSettings) getIconImages() *[]image.Image {
	return s.iconImages.Load()
}

// resetIconImages resets the pending icon images.
// If new icon images are set after imgs is obtained, nothing happens.
func (s *windowSettings) resetIconImages(imgs *[]image.Image) {
	s.iconImages.CompareAndSwap(imgs, nil)
}

func (s *windowSettings) setIconImages(iconImages []image.Image) {
	// Even if iconImages is nil, always create a slice.
	// A 0-size slice and nil are distinguished.
	// See the comment in updateIconIfNeeded.
	newImages := make([]image.Image, len(iconImages))
	copy(newImages, iconImages)
	s.iconImages.Store(&newImages)
}

func (s *windowSettings) getInitWindowPositionInDIP() (int, int) {
	if microsoftgdk.IsXbox() {
		return 0, 0
	}

	pt := s.initWindowPositionInDIP.Load()
	if pt == nil {
		return invalidPos, invalidPos
	}
	return pt.X, pt.Y
}

func (s *windowSettings) setInitWindowPositionInDIP(x, y int) {
	if microsoftgdk.IsXbox() {
		return
	}

	pt := image.Pt(x, y)
	s.initWindowPositionInDIP.Store(&pt)
}

func (s *windowSettings) getInitWindowSizeInDIP() (int, int) {
	if microsoftgdk.IsXbox() {
		return microsoftgdk.MonitorResolution()
	}

	pt := s.initWindowSizeInDIP.Load()
	return pt.X, pt.Y
}

func (s *windowSettings) isInitWindowTransparent() bool {
	return s.initWindowTransparent.Load()
}

func (s *windowSettings) isInitWindowUnfocused() bool {
	return s.initWindowUnfocused.Load()
}

func (s *windowSettings) isInitWindowFloating() bool {
	return s.initWindowFloating.Load()
}

func (s *windowSettings) isInitWindowMaximized() bool {
	return s.initWindowMaximized.Load()
}

func (s *windowSettings) isInitWindowMousePassthrough() bool {
	return s.initWindowMousePassthrough.Load()
}
