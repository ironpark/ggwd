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

//go:build ggwd_remote && ((unix && !ios && !android) || windows)

package viewer_test

import (
	"bytes"
	"fmt"
	"image/color"
	"io/fs"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	ggfxremote "github.com/ironpark/ggfx/remote"

	"github.com/ironpark/ggwd"
	"github.com/ironpark/ggwd/remote"
	"github.com/ironpark/ggwd/remote/viewer"
)

// These tests run the remote backend in a child process, the app, against a fake viewer in the
// test over a real socket: an app is one ggwd.Run in a process.

const appEnv = "GGWD_REMOTE_TEST_APP"

func init() {
	// ggwd.Run runs on the main thread.
	runtime.LockOSThread()
}

func TestMain(m *testing.M) {
	if os.Getenv(appEnv) != "" {
		if err := runApp(); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		os.Exit(0)
	}
	os.Exit(m.Run())
}

// runApp is the child's app: a window that counts the keys pressed in its title, fills its
// screen, and refuses to close the first time it is asked.
func runApp() error {
	var w *ggwd.Window
	keys, refused := 0, false
	return ggwd.Run(ggwd.HandlerFunc(func(ev ggwd.Event) error {
		switch ev := ev.(type) {
		case ggwd.StartEvent:
			var err error
			w, err = ggwd.NewWindow(&ggwd.WindowOptions{Title: "remote", Width: 120, Height: 80, Resizable: true})
			return err
		case ggwd.KeyEvent:
			switch {
			case !ev.Pressed:
			case ev.Key == ggwd.KeyT:
				w.SetTextInputEnabled(true)
				w.SetTextInputRect(10, 20.5, 1, 18)
				w.SetTextInputContext("ab", "cd")
			case ev.Key == ggwd.KeyV:
				// A paste of the viewer's clipboard, copied back with a mark.
				text, err := ggwd.ClipboardText()
				if err != nil {
					w.SetTitle("paste failed: " + err.Error())
					break
				}
				if err := ggwd.SetClipboardText(text + "!"); err != nil {
					w.SetTitle("copy failed: " + err.Error())
					break
				}
				w.SetTitle("pasted " + text)
			case ev.Key == ggwd.KeyL:
				w.SetSizeLimits(100, 50, 0, 0)
				w.SetResizable(false)
				w.StartDrag()
			default:
				keys++
				w.SetTitle(fmt.Sprintf("keys %d", keys))
				w.RequestFrame()
			}
		case ggwd.CompositionEvent:
			w.SetTitle(fmt.Sprintf("composition %s %d %d %t", ev.Text, ev.Start, ev.End, ev.Done))
		case ggwd.TextEvent:
			w.SetTitle(fmt.Sprintf("text %s %v %t", ev.Text, *ev.Replacement, ev.Passthrough))
		case ggwd.DropEvent:
			entries, err := fs.ReadDir(ev.Files, ".")
			if err != nil {
				return err
			}
			var names []string
			for _, e := range entries {
				names = append(names, e.Name())
			}
			w.SetTitle("drop " + strings.Join(names, ","))
		case ggwd.FrameEvent:
			ev.Screen.Fill(color.RGBA{uint8(keys * 40), 0x80, 0, 0xff})
		case ggwd.CloseEvent:
			if !refused {
				refused = true
				ev.KeepOpen()
			}
		}
		return nil
	}), nil)
}

// fakeViewer is the viewer's end: it reads the app's messages into a channel.
type fakeViewer struct {
	t    *testing.T
	conn *remote.Conn
	msgs chan remote.Message
	errs chan error
	shm  []byte
	app  *exec.Cmd
	out  *bytes.Buffer
}

func socketPath(t *testing.T) string {
	dir, err := os.MkdirTemp("", "ggwd")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.RemoveAll(dir) })
	return filepath.Join(dir, "s")
}

func startApp(t *testing.T, path string) (*exec.Cmd, *bytes.Buffer) {
	t.Helper()
	cmd := exec.Command(os.Args[0])
	cmd.Env = append(os.Environ(), appEnv+"=1", remote.EnvSocket+"="+path)
	out := &bytes.Buffer{}
	cmd.Stdout, cmd.Stderr = out, out
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = cmd.Process.Kill(); _, _ = cmd.Process.Wait() })
	return cmd, out
}

// welcome starts an app, accepts it and says welcome with version.
func welcome(t *testing.T, version uint16) *fakeViewer {
	t.Helper()
	path := socketPath(t)
	ln, err := net.ListenUnix("unix", &net.UnixAddr{Name: path, Net: "unix"})
	if err != nil {
		t.Fatal(err)
	}
	defer ln.Close()
	app, out := startApp(t, path)
	_ = ln.SetDeadline(time.Now().Add(10 * time.Second))
	c, err := ln.AcceptUnix()
	if err != nil {
		t.Fatalf("the app did not connect: %v\n%s", err, out)
	}
	v := &fakeViewer{t: t, conn: remote.NewConn(c), msgs: make(chan remote.Message, 64), errs: make(chan error, 1), app: app, out: out}
	t.Cleanup(func() { v.conn.Close() })
	info, err := ggfxremote.Info{A8: true, MaxImageSize: 4096}.MarshalBinary()
	if err != nil {
		t.Fatal(err)
	}
	v.send(&remote.Welcome{Version: version, Driver: info})
	go func() {
		for {
			m, err := v.conn.Receive()
			if err != nil {
				v.errs <- err
				return
			}
			v.msgs <- m
		}
	}()
	return v
}

func (v *fakeViewer) send(m remote.Message) {
	v.t.Helper()
	if err := v.conn.Send(m); err != nil {
		v.t.Fatal(err)
	}
}

// next returns the next message other than a SharedMemory, which it maps.
func (v *fakeViewer) next(timeout time.Duration) remote.Message {
	v.t.Helper()
	for {
		select {
		case m := <-v.msgs:
			if sm, ok := m.(*remote.SharedMemory); ok {
				b, err := sm.Map()
				if err != nil {
					v.t.Fatal(err)
				}
				v.shm = b
				continue
			}
			return m
		case err := <-v.errs:
			v.t.Fatalf("the connection ended: %v\n%s", err, v.out)
		case <-time.After(timeout):
			return nil
		}
	}
}

func expect[M remote.Message](v *fakeViewer) M {
	v.t.Helper()
	m := v.next(10 * time.Second)
	got, ok := m.(M)
	if !ok {
		var want M
		v.t.Fatalf("got %T, want %T\n%s", m, want, v.out)
	}
	return got
}

// batch expects a Batch, checks its header and returns it.
func (v *fakeViewer) batch() *remote.Batch {
	v.t.Helper()
	b := expect[*remote.Batch](v)
	w, h, err := ggfxremote.ScreenSize(v.shm[b.Offset : b.Offset+b.Length])
	if err != nil {
		v.t.Fatal(err)
	}
	if w != 240 || h != 160 {
		v.t.Errorf("the batch draws to a screen of %dx%d, want 240x160", w, h)
	}
	return b
}

func (v *fakeViewer) open() {
	v.t.Helper()
	if h := expect[*remote.Hello](v); h.Version != remote.Version || h.PID != v.app.Process.Pid {
		v.t.Errorf("Hello: %+v", h)
	}
	if o := expect[*remote.OpenWindow](v); o.Title != "remote" || o.Width != 120 || o.Height != 80 || !o.Resizable {
		v.t.Errorf("OpenWindow: %+v", o)
	}
	v.send(&remote.Config{Width: 120, Height: 80, PixelWidth: 240, PixelHeight: 160, Scale: 2, Focused: true, MonitorWidth: 1000, MonitorHeight: 800, RefreshRate: 60})
}

// TestApp runs an app through its life: the handshake, its frames, input that reaches it, the
// pacing of its frames by the viewer's acks, and a close it refuses once.
func TestApp(t *testing.T) {
	v := welcome(t, remote.Version)
	v.open()

	b := v.batch()
	if !b.Present {
		t.Error("the first frame's batch does not present")
	}

	// Input while a frame is in flight is drawn at once, in a second frame in flight.
	key := func(seq uint64) {
		in, ok := viewer.Input(ggwd.KeyEvent{Key: ggwd.KeyA, Pressed: true})
		if !ok {
			t.Fatal("a key event is not input")
		}
		in.Seq = seq
		v.send(&in)
	}
	key(1)
	if title := expect[*remote.SetTitle](v); title.Title != "keys 1" {
		t.Errorf("SetTitle: %q", title.Title)
	}
	b2 := v.batch()
	if b2.Seq != b.Seq+1 || b2.InputSeq != 1 || !b2.Present || b2.Offset == b.Offset {
		t.Errorf("the second batch, in the other slot: %+v after %+v", b2, b)
	}

	// With two in flight, the next input waits for an ack.
	key(2)
	if title := expect[*remote.SetTitle](v); title.Title != "keys 2" {
		t.Errorf("SetTitle: %q", title.Title)
	}
	if m := v.next(300 * time.Millisecond); m != nil {
		t.Fatalf("got a %T with two batches in flight", m)
	}
	v.send(&remote.Ack{Seq: b.Seq})
	b3 := v.batch()
	if b3.Seq != b2.Seq+1 || b3.InputSeq != 2 || b3.Offset != b.Offset {
		t.Errorf("the third batch, in the first slot: %+v", b3)
	}
	v.send(&remote.Ack{Seq: b2.Seq})
	v.send(&remote.Ack{Seq: b3.Seq})

	// A resize makes a frame at the new size.
	v.send(&remote.Config{Width: 130, Height: 90, PixelWidth: 260, PixelHeight: 180, Scale: 2, Focused: true, MonitorWidth: 1000, MonitorHeight: 800, RefreshRate: 60})
	b4 := expect[*remote.Batch](v)
	if w, h, _ := ggfxremote.ScreenSize(v.shm[b4.Offset : b4.Offset+b4.Length]); w != 260 || h != 180 {
		t.Errorf("after the resize, a screen of %dx%d", w, h)
	}
	v.send(&remote.Ack{Seq: b4.Seq})

	// The first close is refused; the second closes the window and ends the app.
	v.send(&remote.Close{})
	if m := v.next(300 * time.Millisecond); m != nil {
		if _, ok := m.(*remote.Closed); ok {
			t.Fatal("the app closed though it refused to")
		}
	}
	v.send(&remote.Close{})
	expect[*remote.Closed](v)
	if err := v.app.Wait(); err != nil {
		t.Errorf("the app ended with %v\n%s", err, v.out)
	}
}

// await returns the first message pred accepts, acking the batches before it.
func (v *fakeViewer) await(what string, pred func(remote.Message) bool) remote.Message {
	v.t.Helper()
	deadline := time.Now().Add(10 * time.Second)
	for time.Now().Before(deadline) {
		m := v.next(time.Until(deadline))
		if m == nil {
			break
		}
		if pred(m) {
			return m
		}
		if b, ok := m.(*remote.Batch); ok {
			v.send(&remote.Ack{Seq: b.Seq})
		}
	}
	v.t.Fatalf("no %s came\n%s", what, v.out)
	return nil
}

func (v *fakeViewer) awaitTitle(title string) {
	v.t.Helper()
	v.await(fmt.Sprintf("title %q", title), func(m remote.Message) bool {
		s, ok := m.(*remote.SetTitle)
		return ok && s.Title == title
	})
}

// TestTextInputAndDrops checks the app's text input state and window options reaching the
// viewer, and the viewer's compositions, text and drops reaching the app as its usual events.
func TestTextInputAndDrops(t *testing.T) {
	v := welcome(t, remote.Version)
	v.open()
	v.batch()
	var seq uint64
	send := func(in remote.Input) {
		seq++
		in.Seq = seq
		v.send(&in)
	}
	input := func(ev ggwd.Event) {
		in, ok := viewer.Input(ev)
		if !ok {
			t.Fatalf("%T is not input", ev)
		}
		send(in)
	}

	input(ggwd.KeyEvent{Key: ggwd.KeyT, Pressed: true})
	want := remote.TextInput{Enabled: true, X: 10, Y: 20.5, Width: 1, Height: 18, Before: "ab", After: "cd"}
	v.await("the text input state", func(m remote.Message) bool {
		ti, ok := m.(*remote.TextInput)
		return ok && *ti == want
	})

	input(ggwd.KeyEvent{Key: ggwd.KeyL, Pressed: true})
	o := v.await("the window's new options", func(m remote.Message) bool {
		o, ok := m.(*remote.OpenWindow)
		return ok && !o.Resizable
	}).(*remote.OpenWindow)
	if o.MinWidth != 100 || o.MinHeight != 50 || o.MaxWidth >= 0 || o.MaxHeight >= 0 || o.Title != "remote" || o.Width != 120 {
		t.Errorf("OpenWindow after the changes: %+v", o)
	}

	v.await("the drag's request", func(m remote.Message) bool {
		r, ok := m.(*remote.WindowRequest)
		return ok && r.Request == remote.RequestStartDrag
	})

	input(ggwd.CompositionEvent{Text: "한", Start: 0, End: 3})
	v.awaitTitle("composition 한 0 3 false")
	input(ggwd.TextEvent{Text: "한", Replacement: &ggwd.TextRange{Start: 1, End: 2}, Passthrough: true})
	v.awaitTitle("text 한 {1 2} true")

	dir := t.TempDir()
	var paths []string
	for _, name := range []string{"a.txt", "b c.txt"} {
		p := filepath.Join(dir, name)
		if err := os.WriteFile(p, []byte(name), 0o644); err != nil {
			t.Fatal(err)
		}
		paths = append(paths, p)
	}
	send(remote.Input{Kind: remote.InputDrop, X: 5, Y: 6, Paths: paths})
	v.awaitTitle("drop a.txt,b c.txt")
}

// TestClipboard checks that the app's clipboard is the viewer's: a paste asks the viewer for its
// text, and a copy puts text there.
func TestClipboard(t *testing.T) {
	v := welcome(t, remote.Version)
	v.open()
	v.batch()
	in, _ := viewer.Input(ggwd.KeyEvent{Key: ggwd.KeyV, Pressed: true})
	in.Seq = 1
	v.send(&in)
	r := v.await("a clipboard request", func(m remote.Message) bool {
		_, ok := m.(*remote.ClipboardRequest)
		return ok
	}).(*remote.ClipboardRequest)
	v.send(&remote.ClipboardText{Seq: r.Seq + 1, Text: "not the answer", OK: true})
	v.send(&remote.ClipboardText{Seq: r.Seq, Text: "복사", OK: true})
	v.await("the copy", func(m remote.Message) bool {
		s, ok := m.(*remote.SetClipboard)
		return ok && s.Text == "복사!"
	})
	v.awaitTitle("pasted 복사")

	// A viewer that cannot read its clipboard says so.
	in.Seq = 2
	v.send(&in)
	r = v.await("a clipboard request", func(m remote.Message) bool {
		_, ok := m.(*remote.ClipboardRequest)
		return ok
	}).(*remote.ClipboardRequest)
	v.send(&remote.ClipboardText{Seq: r.Seq})
	v.await("the failed paste", func(m remote.Message) bool {
		s, ok := m.(*remote.SetTitle)
		return ok && strings.HasPrefix(s.Title, "paste failed")
	})
}

// TestViewerGone ends the connection under a running app, which exits with an error.
func TestViewerGone(t *testing.T) {
	v := welcome(t, remote.Version)
	v.open()
	v.batch()
	v.conn.Close()
	err := v.app.Wait()
	if err == nil || !strings.Contains(v.out.String(), remote.ErrViewerGone.Error()) {
		t.Errorf("the app ended with %v:\n%s", err, v.out)
	}
}

// TestVersionMismatch welcomes an app with another protocol version, which it refuses.
func TestVersionMismatch(t *testing.T) {
	v := welcome(t, remote.Version+1)
	err := v.app.Wait()
	if err == nil || !strings.Contains(v.out.String(), "protocol version") {
		t.Errorf("the app ended with %v:\n%s", err, v.out)
	}
}

// TestNoViewer starts an app with no viewer at its socket, which fails to start.
func TestNoViewer(t *testing.T) {
	app, out := startApp(t, socketPath(t))
	start := time.Now()
	err := app.Wait()
	if err == nil || !strings.Contains(out.String(), "no remote viewer") {
		t.Errorf("the app ended with %v:\n%s", err, out)
	}
	if d := time.Since(start); d > remote.DialTimeout+5*time.Second {
		t.Errorf("the app took %v to give up", d)
	}
}
