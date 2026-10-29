# ggwd

ggwd is the window system of [ggui](https://github.com/ironpark/ggui):
windows, an event loop that delivers input and frames as events, and the
platform's cursor, clipboard, monitors, IME, drag and drop, accessibility and
native hooks. It came out of [ggfx](https://github.com/ironpark/ggfx), the
renderer it draws with, and uses nothing of ggfx but its public renderer API.

The window model and the event contract are documented in
[docs/window.md](docs/window.md). See [FORK.md](FORK.md) for what changed
against Ebitengine, which the window code derives from.

## Platforms

ggwd requires Go 1.27 or later, and so macOS 13 Ventura or later.

- macOS: Cocoa, with Metal or OpenGL
- Windows: Win32, with DirectX 11/12 or OpenGL
- Linux, FreeBSD, NetBSD, OpenBSD: X11 through glfw (requires cgo), with OpenGL
- Browser: one canvas, with WebGL, via `GOOS=js GOARCH=wasm`

## License

Apache-2.0, the same as Ebitengine. See [LICENSE](LICENSE) and
[NOTICE.md](NOTICE.md).
