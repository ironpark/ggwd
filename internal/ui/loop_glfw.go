// Copyright 2024 The Ebitengine Authors
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

//go:build !android && !ios && !js && !nintendosdk && !playstation5

package ui

import (
	stdcontext "context"
	"errors"
	"runtime"
	"time"

	"golang.org/x/sync/errgroup"

	"github.com/ironpark/ggfx/host"
	"github.com/ironpark/ggwd/internal/colormode"
	"github.com/ironpark/ggwd/internal/glfw"
	"github.com/ironpark/ggwd/internal/hook"
	"github.com/ironpark/ggwd/internal/thread"
)

// errWindowClosed is returned by updateWindow when the window asked to close.
var errWindowClosed = errors.New("ui: window closed")

// runLoop runs the loop. initMain runs on the main thread first; start, if any, runs on the loop's
// goroutine before the first iteration.
func (u *UserInterface) runLoop(options *RunOptions, initMain func() error, start func() error) error {
	if options.SingleThread || buildTagSingleThread || runtime.GOOS == "js" {
		return u.runSingleThread(options, initMain, start)
	}
	return u.runMultiThread(options, initMain, start)
}

func (u *UserInterface) runMultiThread(options *RunOptions, initMain func() error, start func() error) error {
	u.mainThread = thread.NewOSThread()

	ctx, cancel := stdcontext.WithCancel(stdcontext.Background())
	defer cancel()

	var wg errgroup.Group

	// Run the loop's goroutine. The renderer's own thread starts at the renderer's initialization.
	wg.Go(func() error {
		defer cancel()
		// The loop terminates the renderer as it ends; this stops it when the initialization fails.
		defer host.Terminate()

		var err error
		u.mainThread.Call(func() {
			err = initMain()
		})
		if err != nil {
			return err
		}

		return u.loopFrames(start)
	})

	// Run the main thread. The loop is the thread's whole life, so a call arriving after
	// it ends is a no-op rather than a block forever.
	_ = u.mainThread.LoopAndStop(ctx)
	return wg.Wait()
}

func (u *UserInterface) runSingleThread(options *RunOptions, initMain func() error, start func() error) error {
	// Initialize the main thread first so the thread is available at u.run (#809).
	u.mainThread = thread.NewNoopThread()
	u.renderOnMainThread = true

	if err := initMain(); err != nil {
		return err
	}

	if err := u.loopFrames(start); err != nil {
		return err
	}

	return nil
}

// initGraphicsOnMainThread initializes GLFW and creates the graphics driver. It must be called on
// the main thread, once.
func (u *UserInterface) initGraphicsOnMainThread(options *RunOptions) error {
	if err := u.ensureGLFWInit(); err != nil {
		return err
	}

	setApplePressAndHoldEnabled(options.ApplePressAndHoldEnabled)

	if err := host.Init(&host.Options{
		GraphicsLibrary: options.GraphicsLibrary,
		ColorSpace:      options.ColorSpace,
		SingleThread:    u.renderOnMainThread,
		MainThread:      u.mainThread.Call,
	}); err != nil {
		return err
	}

	if host.CurrentDriver().Library == host.GraphicsLibraryOpenGL {
		if err := u.createGLContextWindow(); err != nil {
			return err
		}
	}
	return nil
}

// createGLContextWindow creates the hidden window whose context the OpenGL driver draws in. Every
// window's context shares objects with it and only copies its screen from there, so no window's
// context has to outlive the others, and closing a window leaves the drawn objects alone.
//
// createGLContextWindow must be called from the main thread.
func (u *UserInterface) createGLContextWindow() error {
	if err := glfw.WindowHint(glfw.Visible, glfw.False); err != nil {
		return err
	}
	if err := u.setOpenGLWindowHints(); err != nil {
		return err
	}
	w, err := glfw.CreateWindow(1, 1, "", nil, nil)
	if err != nil {
		return err
	}
	// The windows set the hints they need; start them from the defaults.
	if err := glfw.DefaultWindowHints(); err != nil {
		return err
	}
	u.glContextWindow = w
	host.SetGLContext(w)
	return nil
}

// createWindowOnMainThread sets the window hints from the window's own settings and the process's
// options, creates
// the GLFW window, its surface and its callbacks. It must be called on the main thread after
// initGraphicsOnMainThread.
func (u *glfwBackend) createWindowOnMainThread() error {
	run := u.ui.runOptions
	if err := glfw.WindowHint(glfw.AutoIconify, glfw.False); err != nil {
		return err
	}

	// Window is shown after the first buffer swap (#2725).
	if err := glfw.WindowHint(glfw.Visible, glfw.False); err != nil {
		return err
	}

	if err := glfw.WindowHintString(glfw.X11ClassName, run.X11ClassName); err != nil {
		return err
	}

	if err := glfw.WindowHintString(glfw.X11InstanceName, run.X11InstanceName); err != nil {
		return err
	}

	// On macOS, window decoration should be initialized once after buffers are swapped (#2600).
	if runtime.GOOS != "darwin" {
		decorated := glfw.False
		if u.settings.isInitWindowDecorated() {
			decorated = glfw.True
		}
		if err := glfw.WindowHint(glfw.Decorated, decorated); err != nil {
			return err
		}
	}

	glfwTransparent := glfw.False
	if u.settings.isInitWindowTransparent() {
		glfwTransparent = glfw.True
	}
	if err := glfw.WindowHint(glfw.TransparentFramebuffer, glfwTransparent); err != nil {
		return err
	}

	// The OpenGL driver needs a window with a GL context, unlike the other drivers.
	// Set the context-related hints before creating a window.
	lib := host.CurrentDriver().Library
	if lib == host.GraphicsLibraryOpenGL {
		if err := u.ui.setOpenGLWindowHints(); err != nil {
			return err
		}
	}

	// A window created without a redirection surface shows nothing unless its content is presented
	// through DirectComposition, and only the graphics driver can tell whether that works (#3489).
	noRedirectionBitmap := glfw.False
	if host.CurrentDriver().DirectComposition {
		noRedirectionBitmap = glfw.True
	}
	if err := glfw.WindowHint(glfw.Win32NoRedirectionBitmap, noRedirectionBitmap); err != nil {
		return err
	}

	// internal/glfw is customized and the default client API is NoAPI, not OpenGLAPI.
	// Then, glfw.WindowHint(glfw.ClientAPI, glfw.NoAPI) doesn't have to be called.

	// Before creating a window, set it unresizable no matter what u.isInitWindowResizable() is (#1987).
	// Making the window resizable here doesn't work correctly when switching to enable resizing.
	resizable := glfw.False
	if WindowResizingMode(u.settings.windowResizingMode.Load()) == WindowResizingModeEnabled {
		resizable = glfw.True
	}
	if err := glfw.WindowHint(glfw.Resizable, resizable); err != nil {
		return err
	}

	floating := glfw.False
	if u.settings.isInitWindowFloating() {
		floating = glfw.True
	}
	if err := glfw.WindowHint(glfw.Floating, floating); err != nil {
		return err
	}

	u.initUnfocused = u.settings.isInitWindowUnfocused()
	focused := glfw.True
	if u.initUnfocused {
		focused = glfw.False
	}
	if err := glfw.WindowHint(glfw.FocusOnShow, focused); err != nil {
		return err
	}

	mousePassthrough := glfw.False
	if u.settings.isInitWindowMousePassthrough() {
		mousePassthrough = glfw.True
	}
	if err := glfw.WindowHint(glfw.MousePassthrough, mousePassthrough); err != nil {
		return err
	}

	if err := u.createWindow(); err != nil {
		return err
	}

	// A SetPreferredColorMode running at the same time applies the color mode on the main thread
	// after this, so the stored value is the latest.
	if m := u.ui.PreferredColorMode(); m != colormode.Unknown {
		if err := u.setWindowColorModeImpl(m); err != nil {
			return err
		}
	}

	// Maximizing a window requires a proper size and position. Call Maximize here (#1117).
	if u.settings.isInitWindowMaximized() {
		if err := u.window.Maximize(); err != nil {
			return err
		}
	}

	if err := u.setWindowResizingModeForOS(WindowResizingMode(u.settings.windowResizingMode.Load())); err != nil {
		return err
	}

	if run.SkipTaskbar {
		// Ignore the error.
		_ = u.skipTaskbar()
	}

	// The OpenGL driver presents through the GLFW window; the others need the native view.
	var target host.SurfaceTarget = host.GLWindow{Context: u.window}
	if lib != host.GraphicsLibraryOpenGL {
		t, err := u.surfaceTarget()
		if err != nil {
			return err
		}
		target = t
	}
	surface, err := host.NewSurface(target, u.settings.isInitWindowTransparent())
	if err != nil {
		return err
	}
	u.surface = surface
	u.context.setSurface(surface)

	// Register callbacks after the window initialization done.
	// The callback might cause swapping frames, that assumes the window is already set (#2137).
	if err := u.registerWindowCloseCallback(); err != nil {
		return err
	}
	if err := u.registerWindowPosCallback(); err != nil {
		return err
	}
	if err := u.registerWindowFramebufferSizeCallback(); err != nil {
		return err
	}
	if err := u.registerWindowContentScaleCallback(); err != nil {
		return err
	}
	if err := u.registerInputCallbacks(); err != nil {
		return err
	}
	if err := u.registerDropCallback(); err != nil {
		return err
	}

	return nil
}

// destroy closes the window and releases what it owns. It must be called on the main thread.
func (u *glfwBackend) destroy() error {
	if u.closed {
		return nil
	}
	u.closed = true
	if u.surface != nil {
		u.surface.Dispose()
		u.surface = nil
	}
	if err := u.window.Destroy(); err != nil {
		return err
	}
	for i, w := range u.ui.windows {
		if w == u {
			u.ui.windows = append(u.ui.windows[:i], u.ui.windows[i+1:]...)
			break
		}
	}
	u.snapshot.Store(nil)
	u.ui.pushEvent(ClosedEvent{Window: u.appWindow()})
	return nil
}

func (u *UserInterface) loopFrames(start func() error) (err error) {
	u.setRunning(true)
	go u.watchSystemColorMode()
	defer func() {
		u.setRunning(false)
		host.Terminate()
		u.mainThread.Call(func() {
			// Mark the termination before terminating GLFW so that a concurrent-safe API
			// like ScheduleFrame stops touching GLFW's state before it is destroyed.
			u.setTerminated()
			if glfwErr := glfw.Terminate(); glfwErr != nil {
				err = errors.Join(err, glfwErr)
			}
		})
	}()

	if start != nil {
		if err := start(); err != nil {
			return err
		}
	}

	for {
		if err := u.updateFrame(); err != nil {
			return err
		}
	}
}

// pumpEvents processes the OS event queue once, waiting for an event when no window wants a frame.
// pumpEvents must be called from the main thread.
func (u *UserInterface) pumpEvents() error {
	u.pollingEvents = true
	defer func() {
		u.pollingEvents = false
	}()
	// Frames are rendered on request: wait for an event unless one is pending. A window that has
	// not presented yet is still hidden and needs its first frame.
	for _, w := range u.windows {
		if w.context.wantsFrame() || !w.bufferOnceSwapped {
			return glfw.PollEvents()
		}
	}
	// A call on its way here sees waitingEvents and wakes the wait, or is seen here.
	u.waitingEvents.Store(true)
	defer u.waitingEvents.Store(false)
	if u.pendingCalls.Load() > 0 {
		return glfw.PollEvents()
	}
	return glfw.WaitEvents()
}

// updateWindow brings the window's state up to date for a frame and returns its sizes.
// It returns errWindowClosed when the window asked to close.
//
// updateWindow must be called from the main thread.
func (u *glfwBackend) updateWindow() (screenWidth, screenHeight int, err error) {
	if err := u.ui.error(); err != nil {
		return 0, 0, err
	}

	sc, err := u.window.ShouldClose()
	if err != nil {
		return 0, 0, err
	}
	if sc {
		return 0, 0, errWindowClosed
	}

	if runtime.GOOS == "darwin" && u.bufferOnceSwapped {
		var err error
		u.darwinInitOnce.Do(func() {
			// On macOS, window decoration should be initialized once after buffers are swapped (#2600).
			decorated := glfw.False
			if u.settings.isInitWindowDecorated() {
				decorated = glfw.True
			}
			if err = u.window.SetAttrib(glfw.Decorated, decorated); err != nil {
				return
			}
		})
		if err != nil {
			return 0, 0, err
		}
	}

	// Showing the window (and the focus and size adjustments that go with it) is skipped when the window
	// is initially invisible, so an application started with SetWindowVisible(false) never shows a window.
	// A later SetWindowVisible(true) shows it through the regular path.
	if u.bufferOnceSwapped && u.settings.isInitWindowVisible() {
		var err error
		u.showWindowOnce.Do(func() {
			// Show the window after first buffer swap to avoid flash of white especially on Windows.
			if err = u.window.Show(); err != nil {
				return
			}
			if !u.initUnfocused {
				if err = u.window.Focus(); err != nil {
					return
				}
			}

			if runtime.GOOS == "darwin" || runtime.GOOS == "windows" {
				return
			}

			// On Linux or UNIX, there is a problematic desktop environment like i3wm
			// where an invisible window size cannot be initialized correctly (#2951).
			// Call SetSize explicitly after the window becomes visible.

			fullscreen, e := u.isFullscreen()
			if e != nil {
				err = e
				return
			}
			if fullscreen {
				return
			}

			m, e := u.currentMonitor()
			if e != nil {
				err = e
				return
			}
			s := m.DeviceScaleFactor()
			newW, newH := windowSizeInGLFWPixels(u.windowWidthInDIP, u.windowHeightInDIP, s)

			// Even though a framebuffer callback is not called, waitForFramebufferSizeCallback returns by timeout,
			// so it is safe to use this.
			if err = u.waitForFramebufferSizeCallback(u.window, func() error {
				return u.window.SetSize(newW, newH)
			}); err != nil {
				return
			}
		})
		if err != nil {
			return 0, 0, err
		}
	}

	return u.screenSize()
}

// windowFrame is what one window needs for a frame, gathered on the main thread.
type windowFrame struct {
	window            *glfwBackend
	render            bool
	present           bool
	screenWidth       int
	screenHeight      int
	deviceScaleFactor float64
	// refreshRate is the refresh rate of the window's monitor in Hz, or 0 when it is unknown.
	refreshRate int
}

func (u *UserInterface) updateFrame() error {
	var unfocused = true
	var monitorChanged bool
	var frames []windowFrame

	var err error
	if u.mainThread.Call(func() {
		if err = u.pumpEvents(); err != nil {
			return
		}

		// A window can be destroyed in the loop; iterate over a copy.
		windows := append([]*glfwBackend(nil), u.windows...)
		for _, w := range windows {
			// On Windows, the focusing state might be always false (#987).
			// On Windows, even if a window is in another workspace, vsync seems to work.
			// Then let's assume the window is always 'focused' as a workaround.
			if runtime.GOOS == "windows" {
				unfocused = false
			} else {
				a, e := w.window.GetAttrib(glfw.Focused)
				if e != nil {
					err = e
					return
				}
				if a != glfw.False {
					unfocused = false
				}
			}

			visible, e := w.window.GetAttrib(glfw.Visible)
			if e != nil {
				err = e
				return
			}
			occluded, e := w.isWindowOccluded()
			if e != nil && !errors.Is(e, errors.ErrUnsupported) {
				err = e
				return
			}
			present := shouldPresentFrame(visible == glfw.True && !occluded, w.bufferOnceSwapped, w.settings.isInitWindowVisible())
			// A window the app asked to keep hidden is a rendering target, not a window
			// waiting to be shown: run its frames and skip the buffer swap alone. A window
			// the OS hid, by occluding or iconifying it, skips the frame altogether.
			render := present || !w.settings.isInitWindowVisible()

			sw, sh, e := w.updateWindow()
			if e == nil {
				// Catches every change of size and position the callbacks did not report.
				w.reportPosition()
				e = w.reportSize()
				// The size is gathered now; only a change after this makes it stale.
				w.sizeChanged.Store(false)
			}
			if e == nil {
				e = w.reportState()
			}
			if e == nil {
				e = w.takeSnapshot()
			}
			if errors.Is(e, errWindowClosed) {
				if e := w.destroy(); e != nil {
					err = e
					return
				}
				continue
			}
			if e != nil {
				err = e
				return
			}

			if e := w.updateSurfaceWindowState(); e != nil {
				err = e
				return
			}

			m, e := w.currentMonitor()
			if e != nil {
				err = e
				return
			}
			if m != w.lastFrameMonitor {
				monitorChanged = true
			}
			w.lastFrameMonitor = m

			frames = append(frames, windowFrame{
				window:            w,
				render:            render,
				present:           present,
				screenWidth:       sw,
				screenHeight:      sh,
				deviceScaleFactor: m.DeviceScaleFactor(),
				refreshRate:       m.RefreshRate(),
			})
		}
	}); err != nil {
		return err
	}

	// Whether swapping buffers waits for the display can differ per monitor, e.g. when the monitors
	// are driven by different GPUs. Measure it again on the new monitor.
	if monitorChanged {
		u.pacer.resetVsyncDetection()
	}

	// The tick advances once per iteration, so that the time-based caches, like the current
	// monitor, keep expiring. The before-update hooks, like the text input's, run for the same
	// reason.
	advanceTick()
	if err := hook.RunBeforeUpdateHooks(); err != nil {
		return err
	}
	if err := u.dispatchEvents(); err != nil {
		return err
	}

	// Run ends when no window is open once the events are handled: the handler can still open
	// one, like on the CloseEvent of the last window.
	var open int
	u.mainThread.Call(func() {
		open = len(u.windows)
	})
	if open == 0 {
		return RegularTermination
	}

	// Apply the icons that could not be applied yet, like in fullscreen.
	for _, f := range frames {
		if err := f.window.updateIconIfNeeded(); err != nil {
			return err
		}
	}

	var needsSwapBuffers bool
	// The loop is paced for the fastest of the monitors it presents to, so that no window is
	// held back by a slower monitor.
	var refreshRate int
	for _, f := range frames {
		// A frame that cannot be rendered keeps its request for a later iteration. So does one
		// whose window the app resized while its events were dispatched: the size gathered for it
		// is stale, and the frame would be drawn at the old size and use up the request.
		if !f.render || !f.window.context.wantsFrame() || f.window.sizeChanged.Load() {
			continue
		}
		ran, err := f.window.context.renderFrame(f.screenWidth, f.screenHeight, f.deviceScaleFactor, false)
		if err != nil {
			return err
		}
		if ran && f.present {
			needsSwapBuffers = true
			refreshRate = max(refreshRate, f.refreshRate)
		}
	}
	if err := u.pacer.flushCommandsAndWait(needsSwapBuffers, refreshRate); err != nil {
		return err
	}

	for _, f := range frames {
		w := f.window
		w.bufferOnceSwappedOnce.Do(func() {
			u.mainThread.Call(func() {
				w.bufferOnceSwapped = true
			})
		})
	}

	// When a window is not focused or in another space, SwapBuffers might return immediately and CPU might be busy.
	// Mitigate this by sleeping (#982, #2521).
	if unfocused {
		const wait = time.Second / 60
		now := time.Now()
		if next := u.unfocusedNextWake.Add(wait); next.After(now) {
			u.unfocusedNextWake = next
			time.Sleep(time.Until(next))
		} else {
			u.unfocusedNextWake = now
		}
	}

	return nil
}
