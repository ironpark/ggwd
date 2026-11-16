// Copyright 2026 The ggwd Authors
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

//go:build ggwd_remote && ((unix && !android && !ios) || windows) && !nintendosdk && !playstation5

package ui

import (
	stdcontext "context"
	"errors"
	"fmt"
	"image"
	"io/fs"
	"os"
	"sync"
	"sync/atomic"
	"time"

	"golang.org/x/sync/errgroup"

	"github.com/ironpark/ggfx/host"
	ggfxremote "github.com/ironpark/ggfx/remote"
	"github.com/ironpark/ggwd/internal/colormode"
	"github.com/ironpark/ggwd/internal/file"
	"github.com/ironpark/ggwd/internal/hook"
	"github.com/ironpark/ggwd/internal/thread"
	"github.com/ironpark/ggwd/remote"
)

func init() {
	newRemoteBackend = newRemoteUI
}

// newRemoteUI returns the remote backend when the environment names a viewer's socket.
func newRemoteUI(u *UserInterface) remoteBackend {
	path := os.Getenv(remote.EnvSocket)
	if path == "" {
		return nil
	}
	r := &remoteUI{u: u, path: path, wake: make(chan struct{}, 1), pixels: make(chan *remote.Pixels, 1)}
	r.cond = sync.NewCond(&r.mu)
	r.monitor = &Monitor{virtual: remoteMonitor{r}}
	return r
}

// remoteUI runs the app with its one window shown by a viewer: see package ggwd/remote. The app
// draws through ggfx's remote driver, whose batches it puts in shared memory for the viewer, and
// takes its input, its window's size and scale and the system's color mode from the viewer.
type remoteUI struct {
	u    *UserInterface
	path string
	conn *remote.Conn
	wake chan struct{}

	// monitor is the viewer's window's monitor, whose values follow the Config.
	monitor *Monitor

	// mu guards what the connection's reader and the loop share.
	mu      sync.Mutex
	cond    *sync.Cond
	cfg     remote.Config
	haveCfg bool
	win     *remoteWindow
	gone    bool // the connection ended
	// connReady is set once the viewer welcomed the app, and conn may be used from any goroutine.
	connReady bool

	// busy is which of the shared memory's two slots hold a batch the viewer has not acked yet,
	// and busyInput the input the frame in each answers. See frameReady for when there may be
	// two.
	busy      [2]bool
	busyInput [2]uint64

	// pixels carries the viewer's answer to a read of pixels.
	pixels chan *remote.Pixels

	// clipboard holds the clipboard requests waiting for the viewer's answer, by Seq.
	clipboardM    sync.Mutex
	clipboardSeq  uint64
	clipboardWait map[uint64]chan *remote.ClipboardText

	// lastInput is the Seq of the last input queued for the app.
	lastInput atomic.Uint64

	// What the loop and the render thread share; a frame's flush is synchronous, so they take turns.
	surface   *host.Surface
	shm       *remote.SharedMemory
	shmBytes  []byte // two slots of len(shmBytes)/2 bytes
	seq       uint64
	input     uint64 // the input the frame being flushed answers
	presented bool   // whether that frame's flush presented
	reported  uint64 // the last input a Batch or an Idle answered
}

// runApp connects to the viewer, sets up the remote driver and runs the loop.
func (r *remoteUI) runApp(app App, options *RunOptions) error {
	u := r.u
	c, err := remote.Dial(r.path)
	if err != nil {
		return fmt.Errorf("ggwd: no remote viewer at %s (%s): %w", r.path, remote.EnvSocket, err)
	}
	r.conn = c
	defer c.Close()

	m, err := c.Receive()
	if err != nil {
		return fmt.Errorf("ggwd: the remote viewer at %s: %w", r.path, err)
	}
	welcome, ok := m.(*remote.Welcome)
	if !ok {
		return fmt.Errorf("ggwd: the remote viewer at %s began with a %T, not a Welcome", r.path, m)
	}
	if welcome.Version != remote.Version {
		return fmt.Errorf("ggwd: the remote viewer speaks protocol version %d; this app speaks %d", welcome.Version, remote.Version)
	}
	var info ggfxremote.Info
	if err := info.UnmarshalBinary(welcome.Driver); err != nil {
		return fmt.Errorf("ggwd: the remote viewer's driver: %w", err)
	}
	if err := c.Send(&remote.Hello{Version: remote.Version, PID: os.Getpid()}); err != nil {
		return err
	}
	r.mu.Lock()
	r.connReady = true
	r.mu.Unlock()
	if err := ggfxremote.Install(info, remoteTransport{r}); err != nil {
		return err
	}
	colormode.SetOverride(r.colorMode)
	defer r.releaseSharedMemory()

	u.startApp(app, options)
	// As on the desktop, the main thread serves calls that must run there, and the loop has a
	// goroutine of its own.
	mt := thread.NewOSThread()
	u.mainThread = mt
	ctx, cancel := stdcontext.WithCancel(stdcontext.Background())
	defer cancel()
	var wg errgroup.Group
	wg.Go(func() error {
		defer cancel()
		var err error
		mt.Call(func() {
			err = host.Init(&host.Options{MainThread: mt.Call})
		})
		if err != nil {
			return err
		}
		return r.loop(app)
	})
	_ = mt.LoopAndStop(ctx)
	return wg.Wait()
}

func (r *remoteUI) releaseSharedMemory() {
	if r.shmBytes != nil {
		_ = remote.Unmap(r.shmBytes)
		r.shmBytes = nil
	}
	if r.shm != nil {
		_ = r.shm.Close()
		r.shm = nil
	}
}

// loop runs the app's events and frames until its window closes or the viewer goes away.
func (r *remoteUI) loop(app App) error {
	u := r.u
	u.setRunning(true)
	defer func() {
		u.setRunning(false)
		host.Terminate()
		u.setTerminated()
	}()
	go r.read()

	if err := app.HandleEvent(StartEvent{}); err != nil {
		return err
	}
	if r.window() == nil {
		return RegularTermination
	}
	for {
		for !r.eventsPending() && !r.frameReady() && !r.ended() && !u.backgroundFrame.Load() {
			<-r.wake
		}
		if err := u.error(); err != nil {
			return err
		}
		advanceTick()
		if err := hook.RunBeforeUpdateHooks(); err != nil {
			return err
		}
		input := r.lastInput.Load()
		if err := u.dispatchEvents(); err != nil {
			return err
		}
		if r.ended() {
			// The ClosedEvent the window's end queued.
			if err := u.dispatchEvents(); err != nil {
				return err
			}
			r.mu.Lock()
			gone := r.gone
			r.mu.Unlock()
			if gone {
				return remote.ErrViewerGone
			}
			return RegularTermination
		}
		if u.backgroundFrame.Load() {
			if err := u.runBackgroundFrame(); err != nil {
				return err
			}
			// What it drew goes to the viewer as a batch of its own unless a frame follows.
			if !r.frameReady() {
				if _, err := host.Flush(false); err != nil {
					return err
				}
			}
		}
		if r.frameReady() {
			if err := r.frame(input); err != nil {
				return err
			}
		}
	}
}

func (r *remoteUI) window() *remoteWindow {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.win
}

func (r *remoteUI) eventsPending() bool {
	u := r.u
	u.eventsMu.Lock()
	defer u.eventsMu.Unlock()
	return len(u.events) > 0
}

// ended reports whether the window closed, by the app or by the viewer going away.
func (r *remoteUI) ended() bool {
	w := r.window()
	return w != nil && w.closed.Load()
}

// frameReady reports whether the window wants a frame that may run now. A frame runs when the
// viewer has acked every batch, so that an animation runs no faster than the viewer shows it;
// and, with one batch in flight, when there is input the batch does not answer: such a frame is
// drawn at once rather than a viewer's frame later, which would add that frame to the input's
// latency. There are never more than two batches in flight.
func (r *remoteUI) frameReady() bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.win == nil || r.win.closed.Load() || r.gone || !r.haveCfg || r.cfg.PixelWidth <= 0 || r.cfg.PixelHeight <= 0 ||
		!r.win.context.wantsFrame() {
		return false
	}
	switch {
	case !r.busy[0] && !r.busy[1]:
		return true
	case r.busy[0] && r.busy[1]:
		return false
	}
	inFlight := r.busyInput[0]
	if r.busy[1] {
		inFlight = r.busyInput[1]
	}
	return r.lastInput.Load() > inFlight
}

// release frees the slot of batch seq, which the viewer is done with.
func (r *remoteUI) release(seq uint64) {
	r.mu.Lock()
	r.busy[seq%2] = false
	r.cond.Broadcast()
	r.mu.Unlock()
}

// frame runs one frame on the remote surface. Its flush ships the frame's batch.
func (r *remoteUI) frame(input uint64) error {
	w := r.window()
	r.mu.Lock()
	cfg := r.cfg
	r.mu.Unlock()
	w.context.frameRequested.Store(false)

	f, err := host.BeginFrame()
	if err != nil {
		return err
	}
	screen := f.Screen(r.surface, cfg.PixelWidth, cfg.PixelHeight)
	r.input, r.presented = input, false
	herr := r.u.app.HandleEvent(FrameEvent{
		Window:    w,
		Screen:    screen,
		Scale:     cfg.Scale,
		Kept:      r.surface.Kept(),
		SetDamage: r.surface.SetDamage,
	})
	if err := f.End(); err != nil {
		return err
	}
	if herr != nil {
		return herr
	}
	if _, err := host.Flush(true); err != nil {
		return err
	}
	if !r.presented && input > r.reported {
		r.reported = input
		return r.send(&remote.Idle{InputSeq: input})
	}
	return nil
}

// send sends m, and treats a viewer gone as the error that ends the loop.
func (r *remoteUI) send(m remote.Message) error {
	if err := r.conn.Send(m); err != nil {
		r.lost()
		return remote.ErrViewerGone
	}
	return nil
}

// read receives the viewer's messages until the connection ends.
func (r *remoteUI) read() {
	u := r.u
	for {
		m, err := r.conn.Receive()
		if err != nil {
			r.lost()
			return
		}
		switch m := m.(type) {
		case *remote.Config:
			r.configure(m)
		case *remote.Input:
			w := r.window()
			if w == nil {
				continue
			}
			if ev := inputEvent(w, m); ev != nil {
				u.pushEvent(ev)
			}
			r.lastInput.Store(m.Seq)
		case *remote.Close:
			if w := r.window(); w != nil {
				u.pushEvent(CloseEvent{Window: w})
			}
		case *remote.Ack:
			r.release(m.Seq)
		case *remote.Pixels:
			r.release(m.Seq)
			r.pixels <- m
		case *remote.ClipboardText:
			r.clipboardM.Lock()
			if ch := r.clipboardWait[m.Seq]; ch != nil {
				ch <- m
				delete(r.clipboardWait, m.Seq)
			}
			r.clipboardM.Unlock()
		default:
			u.setError(fmt.Errorf("ggwd: the remote viewer sent a %T, which a viewer does not", m))
		}
		r.wakeLoop()
	}
}

// lost ends the window when the viewer goes away: the app gets its ClosedEvent, and the loop ends
// with ErrViewerGone.
func (r *remoteUI) lost() {
	r.mu.Lock()
	first := !r.gone
	r.gone = true
	r.cond.Broadcast()
	w := r.win
	r.mu.Unlock()
	if !first {
		return
	}
	select {
	case r.pixels <- &remote.Pixels{Err: remote.ErrViewerGone.Error()}:
	default:
	}
	r.clipboardM.Lock()
	for seq, ch := range r.clipboardWait {
		close(ch)
		delete(r.clipboardWait, seq)
	}
	r.clipboardM.Unlock()
	if w != nil && w.closed.CompareAndSwap(false, true) {
		r.u.pushEvent(ClosedEvent{Window: w})
	}
	r.wakeLoop()
}

// configure takes the viewer's window as it is now, and tells the app what changed.
func (r *remoteUI) configure(c *remote.Config) {
	u := r.u
	r.mu.Lock()
	old, had := r.cfg, r.haveCfg
	r.cfg, r.haveCfg = *c, true
	w := r.win
	r.mu.Unlock()
	if w == nil {
		return
	}
	if !had || old.Width != c.Width || old.Height != c.Height || old.Scale != c.Scale ||
		old.PixelWidth != c.PixelWidth || old.PixelHeight != c.PixelHeight {
		u.pushEvent(ResizeEvent{Window: w, Width: c.Width, Height: c.Height, Scale: c.Scale})
	}
	if old.Focused != c.Focused {
		u.pushEvent(FocusEvent{Window: w, Focused: c.Focused})
	}
	if had && old.ColorMode != c.ColorMode {
		u.pushEvent(SystemColorModeEvent{Mode: colormode.ColorMode(c.ColorMode)})
	}
	if had && (old.MonitorWidth != c.MonitorWidth || old.MonitorHeight != c.MonitorHeight || old.Scale != c.Scale) {
		u.pushEvent(MonitorEvent{})
	}
	w.context.requestFrame()
}

func (r *remoteUI) config() remote.Config {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.cfg
}

func (r *remoteUI) colorMode() colormode.ColorMode {
	return colormode.ColorMode(r.config().ColorMode)
}

// remoteTransport is ggfx's transport to the viewer: the batches go through the shared memory,
// one at a time.
type remoteTransport struct{ r *remoteUI }

func (t remoteTransport) Send(batch []byte, present bool) error {
	r := t.r
	if err := r.ship(batch, present, nil); err != nil {
		return err
	}
	if present {
		r.presented = true
		r.reported = r.input
	}
	return nil
}

func (t remoteTransport) ReadPixels(batch []byte, id uint32, regions []image.Rectangle, dsts [][]byte) error {
	r := t.r
	if err := r.ship(batch, false, &remote.ReadRequest{Image: id, Regions: regions}); err != nil {
		return err
	}
	p := <-r.pixels
	if p.Err != "" {
		return fmt.Errorf("ggwd: the remote viewer read no pixels: %s", p.Err)
	}
	if len(p.Pixels) != len(dsts) {
		return errors.New("ggwd: the remote viewer read pixels of other regions")
	}
	for k, d := range dsts {
		if len(p.Pixels[k]) != len(d) {
			return errors.New("ggwd: the remote viewer read pixels of other regions")
		}
		copy(d, p.Pixels[k])
	}
	return nil
}

// ship puts batch in its slot of the shared memory once the viewer is done with the batch there
// before, and tells the viewer. Batch n goes to slot n%2. It runs on the render thread.
func (r *remoteUI) ship(batch []byte, present bool, read *remote.ReadRequest) error {
	seq := r.seq + 1
	slot := seq % 2
	grow := len(batch) > len(r.shmBytes)/2

	r.mu.Lock()
	// A larger file replaces the one the viewer may still read: it waits for both slots.
	for !r.gone && (r.busy[slot] || grow && r.busy[1-slot]) {
		r.cond.Wait()
	}
	if r.gone {
		r.mu.Unlock()
		return remote.ErrViewerGone
	}
	r.busy[slot], r.busyInput[slot] = true, r.input
	r.mu.Unlock()

	if grow {
		r.releaseSharedMemory()
		shm, b, err := remote.NewSharedMemory(2 * max(2*len(batch), 4<<20))
		if err != nil {
			return err
		}
		r.shm, r.shmBytes = shm, b
		if err := r.send(shm); err != nil {
			return err
		}
	}
	size := len(r.shmBytes) / 2
	offset := int(slot) * size
	copy(r.shmBytes[offset:offset+size], batch)
	r.seq = seq
	return r.send(&remote.Batch{
		Seq:      seq,
		InputSeq: r.input,
		Present:  present,
		Offset:   int64(offset),
		Length:   int64(len(batch)),
		Read:     read,
	})
}

func (r *remoteUI) newWindow(o *WindowOptions, handle any) (AppWindow, error) {
	u := r.u
	if u.app == nil || !u.isRunning() {
		return nil, errors.New("ui: NewWindow can be called only while RunApp runs")
	}
	r.mu.Lock()
	if r.win != nil {
		r.mu.Unlock()
		return nil, errors.New("ui: an app shown by a remote viewer has one window")
	}
	w := &remoteWindow{r: r, handle: handle}
	w.context = newEventContext(u.app, w)
	r.win = w
	cfg, have := r.cfg, r.haveCfg
	r.mu.Unlock()

	var err error
	u.mainThread.Call(func() {
		r.surface, err = host.NewSurface(host.RemoteTarget{}, false)
	})
	if err != nil {
		return nil, err
	}
	width, height := o.Width, o.Height
	if width <= 0 || height <= 0 {
		width, height = 640, 480
	}
	limit := func(v int) float64 {
		if v <= 0 {
			return -1
		}
		return float64(v)
	}
	w.options = remote.OpenWindow{
		Title: o.Title, Width: float64(width), Height: float64(height), Resizable: o.Resizable,
		MinWidth: limit(o.MinWidth), MinHeight: limit(o.MinHeight), MaxWidth: limit(o.MaxWidth), MaxHeight: limit(o.MaxHeight),
		Decorated: o.Decorated, Floating: o.Floating, TitlebarHidden: o.TitlebarHidden, Transparent: o.Transparent,
	}
	opts := w.options
	if err := r.send(&opts); err != nil {
		return nil, err
	}
	if have {
		u.pushEvent(ResizeEvent{Window: w, Width: cfg.Width, Height: cfg.Height, Scale: cfg.Scale})
		if cfg.Focused {
			u.pushEvent(FocusEvent{Window: w, Focused: true})
		}
	}
	return w, nil
}

func (r *remoteUI) wakeLoop() {
	select {
	case r.wake <- struct{}{}:
	default:
	}
}

func (r *remoteUI) runOnMainThread(f func()) {
	u := r.u
	if !u.isRunning() || u.isTerminated() {
		return
	}
	if t, ok := u.mainThread.(interface{ IsCurrent() bool }); ok && t.IsCurrent() {
		f()
		return
	}
	u.mainThread.Call(f)
}

func (r *remoteUI) appendMonitors(monitors []*Monitor) []*Monitor {
	return append(monitors, r.monitor)
}

func (r *remoteUI) monitorDeviceScaleFactor() float64 {
	return remoteMonitor{r}.deviceScaleFactor()
}

// clipboardTimeout is how long clipboardText waits for the viewer's answer.
const clipboardTimeout = 5 * time.Second

// clipboardText asks the viewer for the text on its system's clipboard, which is the app's.
func (r *remoteUI) clipboardText() (string, error) {
	if !r.connected() {
		return "", errors.ErrUnsupported
	}
	ch := make(chan *remote.ClipboardText, 1)
	r.clipboardM.Lock()
	r.clipboardSeq++
	seq := r.clipboardSeq
	if r.clipboardWait == nil {
		r.clipboardWait = map[uint64]chan *remote.ClipboardText{}
	}
	r.clipboardWait[seq] = ch
	r.clipboardM.Unlock()
	defer func() {
		r.clipboardM.Lock()
		delete(r.clipboardWait, seq)
		r.clipboardM.Unlock()
	}()
	if err := r.send(&remote.ClipboardRequest{Seq: seq}); err != nil {
		return "", err
	}
	t := time.NewTimer(clipboardTimeout)
	defer t.Stop()
	select {
	case m, ok := <-ch:
		if !ok {
			return "", remote.ErrViewerGone
		}
		if !m.OK {
			return "", errors.New("ggwd: the remote viewer could not read the clipboard")
		}
		return m.Text, nil
	case <-t.C:
		return "", errors.New("ggwd: the remote viewer did not answer for the clipboard")
	}
}

// setClipboardText asks the viewer to put text on its system's clipboard.
func (r *remoteUI) setClipboardText(text string) error {
	if !r.connected() {
		return errors.ErrUnsupported
	}
	return r.send(&remote.SetClipboard{Text: text})
}

// connected reports whether the app talks to its viewer: from its Welcome on, until it goes.
func (r *remoteUI) connected() bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.connReady && !r.gone
}

// remoteMonitor is the viewer's window's monitor, as the last Config described it. As a virtual
// monitor, it has no name and no known refresh rate.
type remoteMonitor struct{ r *remoteUI }

func (m remoteMonitor) deviceScaleFactor() float64 {
	if s := m.r.config().Scale; s > 0 {
		return s
	}
	return 1
}

func (m remoteMonitor) outsideSize() (float64, float64) {
	c := m.r.config()
	return c.MonitorWidth, c.MonitorHeight
}

func modsFromRemote(m remote.Mods) KeyModifiers {
	return KeyModifiers{
		Shift: m&remote.ModShift != 0,
		Ctrl:  m&remote.ModCtrl != 0,
		Alt:   m&remote.ModAlt != 0,
		Meta:  m&remote.ModMeta != 0,
	}
}

// inputEvent returns the event of the viewer's input for w, or nil for a kind it does not know
// and for a drag or a drop whose paths make no file system.
func inputEvent(w AppWindow, in *remote.Input) Event {
	switch in.Kind {
	case remote.InputMouseMove:
		return MouseMoveEvent{Window: w, X: in.X, Y: in.Y, Modifiers: modsFromRemote(in.Mods)}
	case remote.InputMouseButton:
		return MouseButtonEvent{Window: w, Button: MouseButton(in.Button), Pressed: in.Pressed, X: in.X, Y: in.Y, Modifiers: modsFromRemote(in.Mods)}
	case remote.InputMouseEnter:
		return MouseEnterEvent{Window: w, Entered: in.Entered}
	case remote.InputScroll:
		return ScrollEvent{Window: w, X: in.X, Y: in.Y, Line: in.Line, Modifiers: modsFromRemote(in.Mods), Phase: ScrollPhase(in.Phase), Precise: in.Precise}
	case remote.InputKey:
		return KeyEvent{Window: w, Key: Key(in.Key), Pressed: in.Pressed, Repeat: in.Repeat, Modifiers: modsFromRemote(in.Mods)}
	case remote.InputText:
		return TextEvent{Window: w, Text: in.Text, ReplacementStart: in.Start, ReplacementEnd: in.End,
			HasReplacement: in.HasReplacement, Passthrough: in.Passthrough}
	case remote.InputComposition:
		return CompositionEvent{Window: w, Text: in.Text, Start: in.Start, End: in.End, Done: in.Done}
	case remote.InputDrag:
		var files fs.FS
		if len(in.Paths) > 0 {
			v, err := file.NewVirtualFS(in.Paths)
			if err != nil {
				return nil
			}
			files = v
		}
		return DragEvent{Window: w, Phase: in.Phase, X: in.X, Y: in.Y, Files: files}
	case remote.InputDrop:
		v, err := file.NewVirtualFS(in.Paths)
		if err != nil {
			return nil
		}
		return DropEvent{Window: w, Files: v}
	}
	return nil
}
