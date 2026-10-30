// Copyright 2014 Hajime Hoshi
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
	"os"
	"path/filepath"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/ironpark/ggfx/host"
	"github.com/ironpark/ggwd/internal/ui"
)

// Termination is a special error which indicates termination without error.
var Termination = ui.RegularTermination

// RunOptions are the options of [Run]. The zero value is the default of every option.
type RunOptions struct {
	// GraphicsLibrary is the graphics library ggfx draws with.
	//
	// The default (zero) value is host.GraphicsLibraryAuto, which lets ggfx choose the graphics library.
	GraphicsLibrary host.GraphicsLibrary

	// ColorSpace indicates the color space of the screen.
	//
	// ColorSpace is available only with some graphics libraries (macOS Metal and WebGL so far).
	// Otherwise, ColorSpace is ignored.
	//
	// The default (zero) value is host.ColorSpaceDefault, which means that color space depends on the environment.
	ColorSpace host.ColorSpace

	// SingleThread indicates whether the single thread mode is used explicitly or not.
	// The single thread mode disables ggwd's thread safety to unlock maximum performance.
	// If you use this you will have to manage threads yourself.
	// Window methods will no longer be concurrent-safe in the single thread mode. They must be
	// called from the main thread or the same goroutine as the event handler.
	//
	// SingleThread works only with desktops.
	//
	// If SingleThread is false, and if the build tag `ggwdsinglethread` is specified,
	// the single thread mode is used.
	//
	// The default (zero) value is false, which means that the single thread mode is disabled.
	SingleThread bool

	// MacOS are the options of macOS, ignored elsewhere.
	MacOS MacOSOptions

	// Windows are the options of Windows, ignored elsewhere.
	Windows WindowsOptions

	// X11 are the options of X11, ignored elsewhere.
	X11 X11Options

	// Browser are the options of the browser, which has one canvas rather than windows.
	Browser BrowserOptions
}

// MacOSOptions are the options of macOS.
type MacOSOptions struct {
	// PressAndHold enables the press-and-hold feature: pressing and holding a key might show a
	// menu to select a character glyph variant. This is useful for GUI applications, but it
	// changes what the keyboard reports: pressing and holding Q would not repeat 'q' as
	// [TextEvent].
	//
	// The default (zero) value is false: pressing and holding a key repeats it.
	PressAndHold bool
}

// WindowsOptions are the options of Windows.
type WindowsOptions struct {
	// SkipTaskbar keeps the windows' icons off the taskbar.
	//
	// The default (zero) value is false, which means that an icon is shown on a taskbar.
	SkipTaskbar bool
}

// X11Options are the options of X11.
type X11Options struct {
	// ClassName is a class name in the ICCCM WM_CLASS window property. The default is the
	// instance name with its first letter in upper case.
	ClassName string

	// InstanceName is an instance name in the ICCCM WM_CLASS window property. The default is
	// the executable's name.
	InstanceName string
}

// BrowserOptions are the options of the browser. They are read at initialization, before the
// canvas is bound, and are ignored elsewhere.
type BrowserOptions struct {
	// Unfocused starts the page without taking focus. Desktop windows use
	// [WindowOptions.Unfocused] instead.
	//
	// The default (zero) value is false, which means that the page takes focus.
	Unfocused bool

	// Transparent makes the canvas composite with the page behind it. Desktop windows use
	// [WindowOptions.Transparent] instead.
	//
	// The default (zero) value is false: the canvas is opaque.
	Transparent bool

	// DisableHiDPI renders at one pixel per CSS pixel: the device scale factor is always 1. This
	// is faster on HiDPI displays, at the expense of rendering quality.
	//
	// The default (zero) value is false, which means that HiDPI is enabled.
	DisableHiDPI bool
}

func toUIRunOptions(options *RunOptions) *ui.RunOptions {
	if options == nil {
		options = &RunOptions{}
	}
	o := ui.RunOptions{
		GraphicsLibrary:          options.GraphicsLibrary,
		InitUnfocused:            options.Browser.Unfocused,
		ScreenTransparent:        options.Browser.Transparent,
		SkipTaskbar:              options.Windows.SkipTaskbar,
		SingleThread:             options.SingleThread,
		DisableHiDPI:             options.Browser.DisableHiDPI,
		ColorSpace:               options.ColorSpace,
		ApplePressAndHoldEnabled: options.MacOS.PressAndHold,
		X11ClassName:             options.X11.ClassName,
		X11InstanceName:          options.X11.InstanceName,
	}
	if o.X11InstanceName == "" {
		o.X11InstanceName = defaultX11InstanceName()
	}
	if o.X11ClassName == "" {
		r, n := utf8.DecodeRuneInString(o.X11InstanceName)
		o.X11ClassName = string(unicode.ToUpper(r)) + o.X11InstanceName[n:]
	}
	return &o
}

// defaultX11InstanceName is the executable's name, as the ICCCM suggests, or "ggwd" when it has
// none.
func defaultX11InstanceName() string {
	if len(os.Args) == 0 || os.Args[0] == "" {
		return "ggwd"
	}
	base := filepath.Base(os.Args[0])
	return strings.TrimSuffix(base, filepath.Ext(base))
}

// AbsPather is a directory entry or a file that has a path in the real file system.
type AbsPather interface {
	// AbsPath returns the absolute path in the real file system.
	AbsPath() string
}

// RunOnMainThread runs f on the main thread and returns after f returns. It is how to reach a
// platform's API that must be called there. Called on the main thread, as in a native callback,
// it runs f at once.
//
// RunOnMainThread does nothing before [Run] starts or after it ends, and in the browser it runs f
// at once.
func RunOnMainThread(f func()) {
	ui.Get().RunOnMainThread(f)
}

// Wake makes the handler receive a [WakeEvent] soon, even while every window is hidden, occluded
// or minimized and draws no frames. It is how work handed over from another goroutine reaches
// the handler without waiting for a window to draw. Wake can be called from any goroutine.
func Wake() {
	ui.Get().Wake()
}
