# Fork notice

ggwd is the window system of [ggui](https://github.com/ironpark/ggui), split out of
[ggfx](https://github.com/ironpark/ggfx), which is itself a fork of
[Ebitengine](https://github.com/hajimehoshi/ebiten) at tag `v2.10.2`. The window code here is
Ebitengine's `internal/ui`, `internal/glfw` and the root package's window API, changed as below,
and it renders with ggfx through ggfx/host, ggfx's renderer API. Its history before the split is in
ggfx's.

Upstream commits to `internal/ui` and `internal/glfw` are cherry-picked here with their paths
rewritten; they are not expected to merge cleanly.

## A hidden window renders

`shouldPresentFrame` returns false both for a window the OS hid, by occluding or
iconifying it, and for a window the app asked to keep hidden. Upstream skipped
the frame in both cases, which made `WindowOptions.Hidden` a window that never
draws. The two are now separate: a hidden-on-purpose window runs its frames and
skips the buffer swap alone, so it is a rendering target for a screenshot dumper
that should not show anything. The loop skips a frame that cannot render, and
`renderFrame` reports only whether the frame ran; the loop presents it when the
window can show it.
An occluded window still skips the frame, which is what keeps the loop off the
GPU when nothing can be seen.

## Every change of size is a ResizeEvent

Upstream reported a resize only from the default framebuffer-size callback, which is swapped
out while a resize the app asked for waits for the framebuffer, and which ignores a window in
fullscreen. Such a resize reached the app as no event, and its frame was drawn at the old size.
Each loop iteration now compares the window's size and scale with what it last reported
(`reportSize`), and so does the end of every wait for the framebuffer: a difference is a
`ResizeEvent` and a frame request. The framebuffer is compared too, since on macOS it follows the
window a little later. A size that changes while events are dispatched makes the size gathered
for that iteration's frame stale, so the frame waits for the next iteration instead of being drawn
at the old size and using up the request.

A `WindowStateEvent` reports a window minimized, maximized, made fullscreen or restored, whoever
did it. Each loop iteration compares the state with the one last reported, as it does the size,
and so do GLFW's iconify and maximize callbacks: a platform reports the end of minimizing there,
and nothing may wake the loop after it. The browser reports fullscreen alone.

`MouseButtonEvent` and `ScrollEvent` carry the modifier keys held at the event, which GLFW passes
to the button callback and which the scroll callback now gets too (from the `NSEvent`,
`GetKeyState` or the X button state). An app that tracked the modifiers from key events missed a
modifier pressed while another window had the focus. `ScrollEvent` is in lines on every platform:
the browser's deltas, which upstream passed through in whatever unit `deltaMode` named, count 20
pixels or the canvas's height a line.

## X11 composes on the event path

An X11 window reports its on-the-spot preedit as `CompositionEvent`, as macOS and Windows
report their marked text, and its committed text arrives as `TextEvent` through the character
callback as before. `SetTextInputEnabled` gives the input context the window's key presses or
takes them back (`XSetICFocus`/`XUnsetICFocus`, resetting the composition), and
`SetTextInputRect` sets its spot location under the caret. While text input is enabled, a key
press the input method filters waits for it to decline the key, which `exp/textinput` used to
switch on for its sessions alone. `exp/textinput`'s X11 backend still works, and its handlers
take precedence where they are set.

`Wake` queues a `WakeEvent`, which is for no window and is dispatched whether or not any window
draws, so that work handed over from another goroutine reaches the handler while every window is
hidden or covered.

## Windows and the event loop

`Run`, `NewWindow`, `Window` and the event types in `app.go` are the API a
GUI uses: any number of windows, frames on request, input as events. The
design and the per-driver status are in [docs/window.md](docs/window.md).
The test suite runs on it: `internal/testing.MainWithRunLoop` opens one
window and runs the tests in its first frame. The legacy `Game`/`RunGame` API
is gone from the public surface, along with the root-level functions that
addressed the game's primary window; `*Window` carries them per window.

`RunOnMainThread` wakes the loop before it queues the function. The loop sleeps in
`WaitEvents` on the main thread when no window wants a frame, and the main thread serves no
call while it waits, so a call from another goroutine used to wait for the next OS event.

On macOS, `RunOnMainThread` runs each function inside an autorelease pool
of its own. The main thread's loop has no pool around the functions it is
handed, so without one every object they autoreleased lived as long as the
process.

## Not done yet

- IME composition on the event path in the browser, which still uses
  `exp/textinput` for it. Once it has one, `exp/textinput` can go: the
  Composer and its sessions are UI-level and live in ggui.
- Runtime validation of Windows native hooks (OLE drag-over, IMM composition,
  WM_GETOBJECT); they are implemented and cross-built. macOS native hooks have
  a two-window smoke test in `examples/nativehooks`.
- X11's input method was only checked by injecting the preedit into the window:
  no ibus or fcitx ran against it.
- The `!android && !ios && !js && !nintendosdk && !playstation5` build
  constraint is still spelled out across `internal/ui`, where `!js` would do.
- The deprecated v2.1 key aliases (`KeyDown`, `Key0`, ...) that `genkeys.go`
  emits into `keys.go`.

