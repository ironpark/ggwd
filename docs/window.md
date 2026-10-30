# Windows and the event loop

ggwd drives a GUI, not a game. A GUI has any number of windows, redraws only
when something changed, and reacts to input as events rather than by polling a
per-tick snapshot. This document is the contract for that model. The legacy `Game`/`RunGame` API is
gone, inside `internal/ui` as well as on the public surface.

## Layers

```
ggwd.Run / ggwd.Window / ggwd.Event              public API (root package)
internal/ui: UserInterface + appWindow/glfwBackend one process, N windows, one loop
ggfx/host: Init, NewSurface, BeginFrame, Flush     the renderer: one device, N surfaces
```

### Renderer: `ggfx/host`

ggwd renders with ggfx through `ggfx/host`, and nothing else of ggfx. A
`host.Surface` is one presentation target: a CAMetalLayer on macOS, a DXGI swap
chain on Windows, the default framebuffer of a GL context, or the canvas in a
browser. The renderer owns one device and any number of surfaces. A window
creates its surface from what the platform has (`host.NSView`, `host.HWND`,
`host.GLWindow`, `host.Canvas`), and a frame draws to `Frame.Screen(surface,
w, h)` between `host.BeginFrame` and `Frame.End`.

`host.Flush(present)` presents every surface whose screen image was drawn
to since the previous flush. A surface that was not drawn is left alone, so a
frame for one window does not disturb the others. When a flush presents several
surfaces, only the first waits for vsync; the others present without waiting,
so N windows do not wait N refreshes.

With OpenGL, GL contexts do not share vertex arrays or framebuffers, so the
renderer draws in one context only: a hidden window's, which `internal/ui`
creates and hands over with `host.SetGLContext`. Every window's context shares
objects with it and copies its screen from there when presenting, so a window
closes without taking the drawn objects with it.

### UI: process and windows

`internal/ui` keeps process state and per-window state apart, and nothing of the
process is addressed through a window: there is no primary window.

Process state (`UserInterface`): the main thread, the renderer's
initialization, the monitors, the open windows in creation order, the event
queue, the preferred color mode, the error, and the loop.

Per-window state (`glfwBackend`, or the canvas): the platform handle, its
`host.Surface`, the window's settings (`windowSettings`: what `NewWindow` was
given and what the app set since), the DIP size and position bookkeeping,
fullscreen restore data, the cursor shape, a snapshot of its state for the
getters, and an `eventContext` that runs its frames: it hands the screen image
to the app as a `FrameEvent` and does nothing else. `appWindow` is what the app
holds; its methods run on the main thread and do nothing once the window is
closed.

What the platform shares between windows is addressed where it belongs: the
preferred color mode is applied to every window, the loop is paced for the
fastest monitor it presented to, and `exp/textinput`'s desktop hooks and
`LogicalPositionToClientPositionInNativePixels` follow the focused window.

### Loop

One iteration of the loop:

1. On the main thread, pump the OS event queue: `PollEvents` if any window has
   a frame pending, `WaitEvents` otherwise. `RequestFrame`, `Wake` and a call
   that needs the main thread while it waits post an empty event so the wait
   ends.
2. Still there, for every window: close it if it asked to (queuing a
   `ClosedEvent`), report a size, position or state the callbacks did not
   (`ResizeEvent`, `MoveEvent`, `WindowStateEvent`), take its snapshot, and
   gather its pixel size and monitor.
3. On the loop's goroutine, dispatch the queued events. Run ends here when no
   window is open, so a handler can still open one on the last `ClosedEvent`.
4. Apply icons that could not be applied yet (a fullscreen window takes none).
5. For every window with a frame pending: run its frame.
6. `host.Flush(present)` once, which presents every drawn surface, and pace:
   the vsync-ignored detection and the unfocused sleep apply once per
   iteration, not per window.

Window creation, destruction and every GLFW call go through the main thread.
Frames are rendered on the renderer's thread in multithread mode; N windows are
drawn sequentially there.

### Threads

The getters (`Size`, `Position`, `IsVisible`, `IsFocused`, `IsMinimized`,
`IsMaximized`, `IsFullscreen`, `Monitor`, `CursorMode`, `native.Handle`) read
the window's snapshot and never wait for the main thread. The snapshot is taken
every iteration after the OS events, and after every change the app makes, so a
getter returns what a setter set; a change the OS makes shows after the next
iteration, together with its event.

The setters run on the main thread and return after they ran. Called on the
main thread itself, as in a native callback, they run at once, as
`RunOnMainThread` does. Called while the main thread waits for OS events, they
wake it first; called while the loop handles events, they don't, which would
only run another iteration.

## Public API

```go
type Handler interface {
    HandleEvent(Event) error
}
type HandlerFunc func(Event) error

// Run starts the loop and blocks until it ends. The handler receives
// StartEvent first and creates its windows there. Returning Termination
// ends the loop; so does having no window open after the events are handled.
func Run(h Handler, options *RunOptions) error

type RunOptions struct {
    GraphicsLibrary host.GraphicsLibrary
    ColorSpace      host.ColorSpace
    SingleThread    bool
    MacOS   MacOSOptions   // PressAndHold
    Windows WindowsOptions // SkipTaskbar
    X11     X11Options     // ClassName, InstanceName
    Browser BrowserOptions // Unfocused, Transparent, DisableHiDPI
}

// The zero value is a decorated, visible, fixed-size 640x480 window
// centered on the monitor. Sizes and positions are DIP, rounded to whole ones.
type WindowOptions struct {
    Title              string
    Width, Height      float64
    Position           *Point // on the monitor; nil centers
    Resizable, Undecorated, Floating, Hidden, Maximized, Transparent, Unfocused bool
    MinWidth, MinHeight, MaxWidth, MaxHeight float64 // 0 is no limit
}

// A transparent window is a property of its surface; DirectX refuses one.
func NewWindow(o *WindowOptions) (*Window, error) // only while Run runs

type Window struct{ ... }
func (w *Window) Close()                 // closes after the current event; ClosedEvent follows
func (w *Window) RequestFrame()          // ask for one FrameEvent
func (w *Window) SetTitle(string)
func (w *Window) Size() (float64, float64)
func (w *Window) SetSize(float64, float64)
func (w *Window) Position() (float64, float64)
func (w *Window) SetPosition(float64, float64)
func (w *Window) SetSizeLimits(minw, minh, maxw, maxh float64) // 0 is no limit
func (w *Window) SetResizable(bool); SetDecorated(bool); SetFloating(bool)
func (w *Window) Show(); Hide(); IsVisible() bool
func (w *Window) Maximize(); Minimize(); Restore(); IsMaximized(); IsMinimized()
func (w *Window) SetFullscreen(bool); IsFullscreen() bool
func (w *Window) IsFocused() bool; Focus(); RequestAttention()
func (w *Window) SetIcon([]image.Image); SetMousePassthrough(bool)
func (w *Window) DeviceScaleFactor() float64
func (w *Window) Monitor() *Monitor
func (w *Window) CursorShape(); SetCursorShape(CursorShape)
func (w *Window) CursorMode(); SetCursorMode(CursorMode)
func (w *Window) SetTextInputEnabled(bool)
func (w *Window) SetTextInputRect(x, y, width, height float64) // caret in client DIP
func (w *Window) SetTextInputContext(before, after string)

func (m *Monitor) Name() string
func (m *Monitor) Size() (float64, float64)
func (m *Monitor) WorkArea() (x, y, width, height float64) // relative to the monitor
func (m *Monitor) DeviceScaleFactor() float64
func (m *Monitor) RefreshRate() int // 0 when unknown
func AppendMonitors([]*Monitor) []*Monitor

func RunOnMainThread(f func()) // runs f at once on the main thread
func Wake()                    // queue a WakeEvent from any goroutine
```

The platform's side of a window is in package `native`, apart from the cross-platform API:

```go
func native.Handle(w *Window) uintptr            // NSWindow* / HWND / X11 window
func native.AccessibilityView(w *Window) uintptr // macOS NSView owned by ggwd; 0 elsewhere
func native.SetAccessibilityHandlers(w *Window, children func() uintptr, hitTest func(x, y float64) uintptr)
func native.SetGetObjectHandler(w *Window, f func(wparam, lparam uintptr) (uintptr, bool))
```

Events. Every event but `StartEvent`, `WakeEvent`, `SystemColorModeEvent` and
`MonitorEvent` has a `Window *Window` field.

| event | fields | when |
|---|---|---|
| `StartEvent` | | once, before anything else |
| `WakeEvent` | | once for every `Wake` |
| `SystemColorModeEvent` | `Mode ColorMode` | the system's color mode changed; noticed within a second |
| `MonitorEvent` | | a monitor was connected or disconnected; not in the browser |
| `ResizeEvent` | `Width, Height float64` (DIP), `Scale` | right after creation, and whenever the size or scale changes |
| `MoveEvent` | `X, Y float64` (DIP on the monitor) | right after creation, and whenever the window moves; not in the browser |
| `FrameEvent` | `Screen *ggfx.Image` (pixels), `Scale float64` | after `RequestFrame`, after a resize, when the window is first shown |
| `FocusEvent` | `Focused bool` | |
| `WindowStateEvent` | `State WindowState` | minimized, maximized, fullscreen or restored, by the app, the user or the platform; the browser reports fullscreen alone |
| `CloseEvent` | | the user asked to close; the window closes after the event unless `KeepOpen()` was called |
| `ClosedEvent` | | the window closed; its last event, after which its methods do nothing |
| `KeyEvent` | `Key`, `Pressed`, `Repeat bool`, `Modifiers Mods` | OS key repeat; modifiers captured at the event, before later releases |
| `TextEvent` | `Text string`, `Replacement *TextRange`, `Passthrough bool` | committed text; `Replacement` is the UTF-8 byte range around the supplied context's caret that the text replaces, or nil |
| `CompositionEvent` | `Text string`, `Start, End int`, `Done bool` | macOS, Windows and X11 marked text; selection is UTF-8 bytes within Text; Done ends the composition with empty Text |
| `MouseMoveEvent` | `X, Y float64`, `Modifiers Mods` | modifiers held as the cursor moved |
| `MouseEnterEvent` | `Entered bool` | the cursor entered or left the client area |
| `MouseButtonEvent` | `Button`, `Pressed`, `X, Y`, `Modifiers Mods` | modifiers held at the press, even one pressed while another window had the focus |
| `ScrollEvent` | `X, Y float64`, `Modifiers Mods`, `Phase ScrollPhase` | in lines: a desktop wheel notch is one; the browser counts 20 DIP or the canvas's height a line |
| `TouchEvent` | `ID TouchID`, `Phase`, `X, Y` | browser only for now |
| `DropEvent` | `Files fs.FS` | |
| `DragEvent` | `Phase DragPhase`, `X, Y float64`, `Files fs.FS` | macOS/Windows file drag entered, moved, exited or ended; Files can be nil |

Coordinates in input events are DIP relative to the window's client area;
`FrameEvent.Screen` is in physical pixels and `Scale` converts between them.
`Screen` is valid during the event only.

`Window.Close()` closes without a `CloseEvent`; that event is for the user's
close button. Either way a `ClosedEvent` follows, and `Run` returns when no
window is open after the events are handled.

The public events and `internal/ui`'s are kept the same by a test
(`event_parity_test.go`): an event or a field added to one side and not the
other, or lost in the conversion, fails it.

### Native hooks

Native input and accessibility hooks are per window and main-thread marshalled.
Text input is initially enabled. SetTextInputEnabled(false) cancels composition
while leaving plain committed characters and keyboard focus available. Enable it
when an editor gains focus, update the caret rectangle after layout or scrolling,
and supply the surrounding text with SetTextInputContext. Do not disable it
between commits: a Korean IME can commit a syllable and mark the next one in the
same OS event. CompositionEvent and TextEvent retain their native order.

macOS uses the existing content view as its NSTextInputClient. Its native
selection and replacement ranges are converted from UTF-16 to UTF-8. Windows
uses IMM composition/result messages and positions the native candidate panel.
Committed result messages are handled once, without a duplicate WM_CHAR. X11
gives the input context the window's key presses while text input is enabled
and puts its spot location under the caret.

macOS accessibility gets a lazily created, mouse-transparent NSView owned and
destroyed by ggwd. SetAccessibilityHandlers supplies that view's children as an
autoreleased NSArray and its hit test as an NSAccessibilityElement; hit-test
coordinates are native screen points. The bridge owns its element objects, not
the view. Windows forwards WM_GETOBJECT through SetGetObjectHandler; false
falls through to the default procedure. These synchronous native callbacks
must answer from a snapshot: do not wait for the event handler from them.
Detach callbacks and retire bridge objects before closing their window.
Unsupported platforms return zero for the view and ignore the
accessibility/IME setters.

File drags use Cocoa dragging callbacks and Windows IDropTarget, with one OLE
registration per window, revoked during teardown. X11 and the browser retain
DropEvent but do not emit pre-drop DragEvent. Native drag and composition
callbacks request a frame when state changes; they do not add a polling loop.
Dialogs use the initiating window's native handle, obtained before entering
RunOnMainThread.

Run `go run ./examples/nativehooks` to exercise text and file drags in two Metal
windows. `-smoke` injects Cocoa callbacks and checks window isolation, UTF-16
selection conversion, Korean commit/next-mark ordering, replacement ranges,
caret rectangle size, drag phases/coordinates and accessibility attachment.
The smoke test passes on OpenGL too (`GGFX_GRAPHICS_LIBRARY=opengl`). The smoke
injection is macOS-only.

### Gamepads

The loop does not read gamepads. They belong to the process rather than to a window and can only
be polled, so package `gamepad` reads them, and an app that wants them polls it. The loop gives it
what it needs: `RunOnMainThread` for the backends that must be read on the main thread, a window's
native handle for DirectInput, and `Wake` to hand the events it found to the loop, which sleeps
until the window system or a `Wake` has something to report. `gamepad.Start` does this; the
handler takes the events with `Poller.Take` on `WakeEvent`.

## What the event path does not do

- No ticks, no `TPS`, and no polling input at all: `inpututil`,
  `internal/inputstate` and `input.go`'s state functions are gone.
  `KeyEvent.Repeat` replaces `KeyPressDuration`. An application that needs key
  state keeps it from the events.
  `Tick()` still advances once per loop iteration, and the before-update
  hooks run then, so `exp/textinput` keeps working when polled every frame.
- No composition events in the browser yet. Its `exp/textinput` backend
  remains available; macOS, Windows and X11 have per-window CompositionEvent.
- One canvas on WebGL. DirectX runs several windows, but that was
  cross-compiled only.
- The clipboard is not supported on Windows yet (`ErrUnsupported`).

## Migration

`Game`, `RunGame` and `RunGameWithOptions` are gone. So are the root-level
functions that addressed the game's primary window: `ScreenSize`,
`ScreenSizeInFullscreen`, `Monitor`, `SetMonitor`, `SetWindowTitle`,
`SetWindowSize` and the rest of `window.go`, the cursor, fullscreen, focus and
vsync functions, and the TPS and FPS-mode API. Use `Window.Size()`,
`Window.Monitor()`, `Window.SetTitle()` and the other `*Window` methods, and
`RunOptions` instead of `RunGameOptions`.

From the API before the window split:

- Sizes and positions are `float64` DIP: `WindowOptions.Width/Height/Min*/Max*`,
  `Window.Size/SetSize/Position/SetPosition/SetSizeLimits` and `Monitor.Size`.
  `WindowOptions.Position` is a `*Point`. A size limit of 0 is no limit
  everywhere; `SetSizeLimits` took -1 for that.
- `RunOptions.ApplePressAndHoldEnabled` is `MacOS.PressAndHold`,
  `SkipTaskbar` is `Windows.SkipTaskbar`, and `X11ClassName`/`X11InstanceName`
  are `X11.ClassName`/`X11.InstanceName`.
- `TextEvent.HasReplacement`, `ReplacementStart` and `ReplacementEnd` are
  `TextEvent.Replacement`, a `*TextRange` that is nil without a replacement.
- Closing the first window no longer ends `Run` while another is open.
