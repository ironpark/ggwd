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

package remote

import (
	"encoding/binary"
	"errors"
	"fmt"
	"image"
	"math"
	"os"
	"time"
)

// Version is the version of the protocol the package doc describes.
const Version = 4

// EnvSocket is the environment variable that names the viewer's socket. An app built with the
// ggwd_remote tag runs remotely when it is set.
const EnvSocket = "GGWD_REMOTE_SOCKET"

// DialTimeout is how long an app tries to connect to a viewer that is not listening yet.
const DialTimeout = 3 * time.Second

// MaxFrameSize bounds a frame, so that a corrupt length cannot make the reader allocate much.
const MaxFrameSize = 256 << 20

// ErrViewerGone is what Run returns when the viewer goes away while the app runs.
var ErrViewerGone = errors.New("ggwd: the remote viewer went away")

// Message is one message of the protocol.
type Message interface {
	typ() uint8
	encode(w *writer)
	decode(r *reader)
}

// The message types.
const (
	typeWelcome = iota + 1
	typeHello
	typeOpenWindow
	typeConfig
	typeInput
	typeClose
	typeAck
	typePixels
	typeSetTitle
	typeSetCursor
	typeSharedMemory
	typeBatch
	typeIdle
	typeClosed
	typeTextInput
	typeWindowRequest
	typeClipboardRequest
	typeClipboardText
	typeSetClipboard
	typeSetSize
	typeSetPosition
	typeSetIcon
	typeSetColorMode
	typeSetDockIcon
	typeSetMenu
	typeUpdateMenuItem
	typeMenuSelected
	typeAppEvent
)

func newMessage(t uint8) Message {
	switch t {
	case typeWelcome:
		return &Welcome{}
	case typeHello:
		return &Hello{}
	case typeOpenWindow:
		return &OpenWindow{}
	case typeConfig:
		return &Config{}
	case typeInput:
		return &Input{}
	case typeClose:
		return &Close{}
	case typeAck:
		return &Ack{}
	case typePixels:
		return &Pixels{}
	case typeSetTitle:
		return &SetTitle{}
	case typeSetCursor:
		return &SetCursor{}
	case typeSharedMemory:
		return &SharedMemory{}
	case typeBatch:
		return &Batch{}
	case typeIdle:
		return &Idle{}
	case typeClosed:
		return &Closed{}
	case typeTextInput:
		return &TextInput{}
	case typeWindowRequest:
		return &WindowRequest{}
	case typeClipboardRequest:
		return &ClipboardRequest{}
	case typeClipboardText:
		return &ClipboardText{}
	case typeSetClipboard:
		return &SetClipboard{}
	case typeSetSize:
		return &SetSize{}
	case typeSetPosition:
		return &SetPosition{}
	case typeSetIcon:
		return &SetIcon{}
	case typeSetColorMode:
		return &SetColorMode{}
	case typeSetDockIcon:
		return &SetDockIcon{}
	case typeSetMenu:
		return &SetMenu{}
	case typeUpdateMenuItem:
		return &UpdateMenuItem{}
	case typeMenuSelected:
		return &MenuSelected{}
	case typeAppEvent:
		return &AppEvent{}
	}
	return nil
}

// Welcome is the viewer's first message.
type Welcome struct {
	Version uint16
	// Driver is the viewer's graphics driver info, as ggfx/remote.Info.MarshalBinary makes it.
	Driver []byte
}

// Hello is the app's first message.
type Hello struct {
	Version uint16
	PID     int
}

// OpenWindow describes the app's window, as ggwd.WindowOptions does. Sizes and positions are in
// device-independent pixels; a negative limit is none. The app sends it again whenever one of its
// options changes; a viewer applies what it can of it and keeps the window it has.
type OpenWindow struct {
	Title                                    string
	Width, Height                            float64
	PositionSet                              bool
	X, Y                                     float64
	Resizable                                bool
	MinWidth, MinHeight, MaxWidth, MaxHeight float64
	Decorated                                bool
	Floating                                 bool
	Hidden                                   bool
	Maximized                                bool
	Unfocused                                bool
	MousePassthrough                         bool
	TitlebarHidden                           bool
	TrafficLightsSet                         bool
	TrafficLightsX, TrafficLightsY           float64
	Transparent                              bool
}

// TextInput is the app's text input state, sent whenever it changes: whether the window takes
// text through the platform's input methods, the caret's rectangle in device-independent pixels,
// where a candidate window goes, and the text around the caret, which an input method may read.
type TextInput struct {
	Enabled             bool
	X, Y, Width, Height float64
	Before, After       string
}

// WindowRequest asks the viewer to perform an action on its window. Request is one of the Request
// constants.
type WindowRequest struct {
	Request uint8
}

// The requests of a WindowRequest.
const (
	// RequestStartDrag moves the window with the pointer from the press of the left button that
	// is still held, as the window's title bar does.
	RequestStartDrag uint8 = iota + 1
	// RequestTitlebarDoubleClick does what a double click on the title bar does.
	RequestTitlebarDoubleClick
	// RequestShow shows a hidden window.
	RequestShow
	// RequestHide hides the window.
	RequestHide
	// RequestMaximize maximizes the window.
	RequestMaximize
	// RequestMinimize minimizes the window.
	RequestMinimize
	// RequestRestore restores a maximized or minimized window.
	RequestRestore
	// RequestFullscreenEnter puts the window into fullscreen.
	RequestFullscreenEnter
	// RequestFullscreenExit takes the window out of fullscreen.
	RequestFullscreenExit
	// RequestFocus brings the window forward and gives it keyboard focus.
	RequestFocus
	// RequestAttention asks the user's attention.
	RequestAttention
)

// Config is the viewer's window as it is now: its client area in device-independent pixels and
// in pixels, which the app's screen is drawn at, its scale, its position on its monitor, whether
// it has the focus and is visible, its window state (ggwd.WindowState), the system's color mode
// (ggwd.ColorMode), and its monitor.
type Config struct {
	Width, Height               float64
	PixelWidth, PixelHeight     int
	Scale                       float64
	X, Y                        float64
	Focused                     bool
	Visible                     bool
	State                       uint8
	ColorMode                   uint8
	MonitorWidth, MonitorHeight float64
	RefreshRate                 int
}

// InputKind is the kind of an Input.
type InputKind uint8

// The kinds of input. Each uses the fields of Input the ggwd event of its name has.
const (
	InputMouseMove InputKind = iota + 1
	InputMouseButton
	InputMouseEnter
	InputScroll
	InputKey
	InputText
	InputComposition
	InputDrag
	InputDrop
)

// Mods are the modifier keys held, a bit each.
type Mods uint8

const (
	ModShift Mods = 1 << iota
	ModCtrl
	ModAlt
	ModMeta
)

// Input is an input event of the viewer's window, with ggwd's values: Key is a ggwd.Key, Button a
// ggwd.MouseButton, Phase a ggwd.ScrollPhase or, for a drag, a ggwd.DragPhase. Seq grows by one
// with each input; the app's batches say up to which one their frame answers.
type Input struct {
	Seq     uint64
	Kind    InputKind
	X, Y    float64
	Line    float64
	Button  int
	Key     int
	Pressed bool
	Repeat  bool
	// Entered is MouseEnter's; Precise is Scroll's.
	Entered bool
	Precise bool
	Mods    Mods
	Phase   int
	Text    string
	// Start and End are a composition's selection, in bytes of Text, and Done ends it. For a Text,
	// with HasReplacement, they are the bytes of the text before it replaces, and Passthrough
	// reports that an input method passed the key on.
	Start, End     int
	Done           bool
	HasReplacement bool
	Passthrough    bool
	// Paths are the files of a drag or a drop, absolute paths on the machine both sides run on.
	Paths []string
}

// Close asks the app to close its window, as the window's close button does.
type Close struct{}

// Ack says the viewer is done with batch Seq: replayed, and its shared memory free.
type Ack struct {
	Seq uint64
}

// Pixels answers a batch's ReadRequest, in place of an Ack: the regions' pixels, or the error the
// read ran into.
type Pixels struct {
	Seq    uint64
	Err    string
	Pixels [][]byte
}

// SetTitle sets the window's title.
type SetTitle struct {
	Title string
}

// SetCursor sets the cursor over the window: Shape is a ggwd.CursorShape, Mode a ggwd.CursorMode.
type SetCursor struct {
	Shape, Mode uint8
}

// SharedMemory gives the viewer the shared memory the app writes its batches to, Size bytes
// long. It replaces the one before, which the viewer unmaps.
type SharedMemory struct {
	Size int64
	// Name is the name of the shared memory where its file cannot go with the message: on
	// Windows, a file mapping's. It is empty on unix systems.
	Name string
	// File is the shared memory file on unix systems: on the app's side, the one to send; on the
	// viewer's, the one received, whose owner it then is.
	File *os.File

	handle uintptr // the file mapping on Windows
}

// Batch is a batch of the app's graphics driver calls at Offset in the shared memory, Length
// bytes long. Present reports that it ends with a present of the window. InputSeq is the last
// Input the frame answers. Read, when set, asks for pixels back.
type Batch struct {
	Seq      uint64
	InputSeq uint64
	Present  bool
	Offset   int64
	Length   int64
	Read     *ReadRequest
}

// ReadRequest asks the viewer to read regions of the app's image Image back once the batch is
// replayed.
type ReadRequest struct {
	Image   uint32
	Regions []image.Rectangle
}

// Idle says the app ran a frame that changed nothing it shows, answering the input up to
// InputSeq.
type Idle struct {
	InputSeq uint64
}

// Closed says the app closed its window.
type Closed struct{}

// ClipboardRequest asks the viewer for the text on the system's clipboard, which it answers with
// a ClipboardText of the same Seq.
type ClipboardRequest struct {
	Seq uint64
}

// ClipboardText answers the ClipboardRequest of Seq with the clipboard's text. OK is false when
// the viewer could not read it.
type ClipboardText struct {
	Seq  uint64
	Text string
	OK   bool
}

// SetClipboard asks the viewer to put Text on the system's clipboard.
type SetClipboard struct {
	Text string
}

// SetSize asks the viewer to resize its window's client area to Width by Height
// device-independent pixels.
type SetSize struct {
	Width, Height float64
}

// SetPosition asks the viewer to move its window to X, Y on its monitor in
// device-independent pixels.
type SetPosition struct {
	X, Y float64
}

// SetIcon sets the viewer's window icon from PNG-encoded images. An empty Icons slice clears the
// window icon.
type SetIcon struct {
	Icons [][]byte
}

// SetColorMode sets the viewer's preferred color mode (ggwd.ColorMode); 0 (ColorModeUnknown)
// clears the preference.
type SetColorMode struct {
	Mode uint8
}

// SetDockIcon shows or hides the viewer's Dock icon on macOS.
type SetDockIcon struct {
	Visible bool
}

// MenuItem is one entry in an app's native menu bar: an action, a separator, or a submenu when
// Items is non-empty.
type MenuItem struct {
	Title                 string
	Separator             bool
	Items                 []MenuItem
	ID                    int
	Key                   string
	Cmd, Shift, Alt, Ctrl bool
	Enabled               bool
	Checked               bool
}

// SetMenu replaces the viewer's native menu bar with Items. An empty Items slice clears it.
type SetMenu struct {
	Items []MenuItem
}

// UpdateMenuItem updates the enabled and/or checked state of an existing menu action by ID.
type UpdateMenuItem struct {
	ID         int
	SetEnabled bool
	Enabled    bool
	SetChecked bool
	Checked    bool
}

// MenuSelected tells the app that the user activated menu action ID in the viewer's native menu.
type MenuSelected struct {
	ID int
}

// AppEvent forwards a process-level desktop event from the viewer to the app.
type AppEvent struct {
	Kind uint8
}

// The kinds of an AppEvent.
const (
	AppEventQuit uint8 = iota + 1
	AppEventReopen
)

func (*Welcome) typ() uint8          { return typeWelcome }
func (*Hello) typ() uint8            { return typeHello }
func (*OpenWindow) typ() uint8       { return typeOpenWindow }
func (*Config) typ() uint8           { return typeConfig }
func (*Input) typ() uint8            { return typeInput }
func (*Close) typ() uint8            { return typeClose }
func (*Ack) typ() uint8              { return typeAck }
func (*Pixels) typ() uint8           { return typePixels }
func (*SetTitle) typ() uint8         { return typeSetTitle }
func (*SetCursor) typ() uint8        { return typeSetCursor }
func (*SharedMemory) typ() uint8     { return typeSharedMemory }
func (*Batch) typ() uint8            { return typeBatch }
func (*Idle) typ() uint8             { return typeIdle }
func (*Closed) typ() uint8           { return typeClosed }
func (*TextInput) typ() uint8        { return typeTextInput }
func (*WindowRequest) typ() uint8    { return typeWindowRequest }
func (*ClipboardRequest) typ() uint8 { return typeClipboardRequest }
func (*ClipboardText) typ() uint8    { return typeClipboardText }
func (*SetClipboard) typ() uint8     { return typeSetClipboard }
func (*SetSize) typ() uint8          { return typeSetSize }
func (*SetPosition) typ() uint8      { return typeSetPosition }
func (*SetIcon) typ() uint8          { return typeSetIcon }
func (*SetColorMode) typ() uint8     { return typeSetColorMode }
func (*SetDockIcon) typ() uint8      { return typeSetDockIcon }
func (*SetMenu) typ() uint8          { return typeSetMenu }
func (*UpdateMenuItem) typ() uint8   { return typeUpdateMenuItem }
func (*MenuSelected) typ() uint8     { return typeMenuSelected }
func (*AppEvent) typ() uint8         { return typeAppEvent }

// maxMenuDepth bounds nested submenu recursion when decoding SetMenu.
const maxMenuDepth = 32

func (m *SetSize) encode(w *writer) { w.f64(m.Width); w.f64(m.Height) }
func (m *SetSize) decode(r *reader) { m.Width, m.Height = r.f64(), r.f64() }

func (m *SetPosition) encode(w *writer) { w.f64(m.X); w.f64(m.Y) }
func (m *SetPosition) decode(r *reader) { m.X, m.Y = r.f64(), r.f64() }

func (m *SetIcon) encode(w *writer) {
	w.u32(uint32(len(m.Icons)))
	for _, b := range m.Icons {
		w.bytes(b)
	}
}

func (m *SetIcon) decode(r *reader) {
	if n := r.count(4); n > 0 {
		m.Icons = make([][]byte, n)
		for i := range m.Icons {
			m.Icons[i] = r.bytes()
		}
	}
}

func (m *SetColorMode) encode(w *writer) { w.u8(m.Mode) }
func (m *SetColorMode) decode(r *reader) { m.Mode = r.u8() }

func (m *SetDockIcon) encode(w *writer) { w.bool(m.Visible) }
func (m *SetDockIcon) decode(r *reader) { m.Visible = r.bool() }

func encodeMenuItems(w *writer, items []MenuItem) {
	w.u32(uint32(len(items)))
	for i := range items {
		it := &items[i]
		w.str(it.Title)
		w.bool(it.Separator)
		w.i32(it.ID)
		w.str(it.Key)
		w.bool(it.Cmd)
		w.bool(it.Shift)
		w.bool(it.Alt)
		w.bool(it.Ctrl)
		w.bool(it.Enabled)
		w.bool(it.Checked)
		encodeMenuItems(w, it.Items)
	}
}

// minMenuItemBytes is the encoded minimum size of one MenuItem:
// Title(4) + Separator(1) + ID(4) + Key(4) + 6 bools(6) + Items count(4) = 23 bytes.
const minMenuItemBytes = 23

func decodeMenuItems(r *reader, depth int) []MenuItem {
	if depth > maxMenuDepth {
		if r.err == nil {
			r.err = errors.New("menu nesting too deep")
		}
		return nil
	}
	n := r.count(minMenuItemBytes)
	if n == 0 {
		return nil
	}
	items := make([]MenuItem, n)
	for i := range items {
		it := &items[i]
		it.Title = r.str()
		it.Separator = r.bool()
		it.ID = r.i32()
		it.Key = r.str()
		it.Cmd = r.bool()
		it.Shift = r.bool()
		it.Alt = r.bool()
		it.Ctrl = r.bool()
		it.Enabled = r.bool()
		it.Checked = r.bool()
		it.Items = decodeMenuItems(r, depth+1)
		if r.err != nil {
			return nil
		}
	}
	return items
}

func (m *SetMenu) encode(w *writer) { encodeMenuItems(w, m.Items) }
func (m *SetMenu) decode(r *reader) { m.Items = decodeMenuItems(r, 0) }

func (m *UpdateMenuItem) encode(w *writer) {
	w.i32(m.ID)
	w.bool(m.SetEnabled)
	w.bool(m.Enabled)
	w.bool(m.SetChecked)
	w.bool(m.Checked)
}

func (m *UpdateMenuItem) decode(r *reader) {
	m.ID = r.i32()
	m.SetEnabled, m.Enabled = r.bool(), r.bool()
	m.SetChecked, m.Checked = r.bool(), r.bool()
}

func (m *MenuSelected) encode(w *writer) { w.i32(m.ID) }
func (m *MenuSelected) decode(r *reader) { m.ID = r.i32() }

func (m *AppEvent) encode(w *writer) { w.u8(m.Kind) }
func (m *AppEvent) decode(r *reader) { m.Kind = r.u8() }

func (m *ClipboardRequest) encode(w *writer) { w.u64(m.Seq) }
func (m *ClipboardRequest) decode(r *reader) { m.Seq = r.u64() }

func (m *ClipboardText) encode(w *writer) { w.u64(m.Seq); w.str(m.Text); w.bool(m.OK) }
func (m *ClipboardText) decode(r *reader) { m.Seq, m.Text, m.OK = r.u64(), r.str(), r.bool() }

func (m *SetClipboard) encode(w *writer) { w.str(m.Text) }
func (m *SetClipboard) decode(r *reader) { m.Text = r.str() }

func (m *WindowRequest) encode(w *writer) { w.u8(m.Request) }
func (m *WindowRequest) decode(r *reader) { m.Request = r.u8() }

func (m *TextInput) encode(w *writer) {
	w.bool(m.Enabled)
	w.f64(m.X)
	w.f64(m.Y)
	w.f64(m.Width)
	w.f64(m.Height)
	w.str(m.Before)
	w.str(m.After)
}

func (m *TextInput) decode(r *reader) {
	m.Enabled = r.bool()
	m.X, m.Y, m.Width, m.Height = r.f64(), r.f64(), r.f64(), r.f64()
	m.Before, m.After = r.str(), r.str()
}

func (m *Welcome) encode(w *writer) { w.u16(m.Version); w.bytes(m.Driver) }
func (m *Welcome) decode(r *reader) { m.Version = r.u16(); m.Driver = r.bytes() }

func (m *Hello) encode(w *writer) { w.u16(m.Version); w.u32(uint32(m.PID)) }
func (m *Hello) decode(r *reader) { m.Version = r.u16(); m.PID = int(r.u32()) }

func (m *OpenWindow) encode(w *writer) {
	w.str(m.Title)
	w.f64(m.Width)
	w.f64(m.Height)
	w.bool(m.PositionSet)
	w.f64(m.X)
	w.f64(m.Y)
	w.bool(m.Resizable)
	w.f64(m.MinWidth)
	w.f64(m.MinHeight)
	w.f64(m.MaxWidth)
	w.f64(m.MaxHeight)
	w.bool(m.Decorated)
	w.bool(m.Floating)
	w.bool(m.Hidden)
	w.bool(m.Maximized)
	w.bool(m.Unfocused)
	w.bool(m.MousePassthrough)
	w.bool(m.TitlebarHidden)
	w.bool(m.TrafficLightsSet)
	w.f64(m.TrafficLightsX)
	w.f64(m.TrafficLightsY)
	w.bool(m.Transparent)
}

func (m *OpenWindow) decode(r *reader) {
	m.Title = r.str()
	m.Width, m.Height = r.f64(), r.f64()
	m.PositionSet = r.bool()
	m.X, m.Y = r.f64(), r.f64()
	m.Resizable = r.bool()
	m.MinWidth, m.MinHeight, m.MaxWidth, m.MaxHeight = r.f64(), r.f64(), r.f64(), r.f64()
	m.Decorated, m.Floating = r.bool(), r.bool()
	m.Hidden, m.Maximized, m.Unfocused, m.MousePassthrough = r.bool(), r.bool(), r.bool(), r.bool()
	m.TitlebarHidden = r.bool()
	m.TrafficLightsSet = r.bool()
	m.TrafficLightsX, m.TrafficLightsY = r.f64(), r.f64()
	m.Transparent = r.bool()
}

func (m *Config) encode(w *writer) {
	w.f64(m.Width)
	w.f64(m.Height)
	w.u32(uint32(m.PixelWidth))
	w.u32(uint32(m.PixelHeight))
	w.f64(m.Scale)
	w.f64(m.X)
	w.f64(m.Y)
	w.bool(m.Focused)
	w.bool(m.Visible)
	w.u8(m.State)
	w.u8(m.ColorMode)
	w.f64(m.MonitorWidth)
	w.f64(m.MonitorHeight)
	w.u32(uint32(m.RefreshRate))
}

func (m *Config) decode(r *reader) {
	m.Width, m.Height = r.f64(), r.f64()
	m.PixelWidth, m.PixelHeight = int(r.u32()), int(r.u32())
	m.Scale = r.f64()
	m.X, m.Y = r.f64(), r.f64()
	m.Focused, m.Visible = r.bool(), r.bool()
	m.State = r.u8()
	m.ColorMode = r.u8()
	m.MonitorWidth, m.MonitorHeight = r.f64(), r.f64()
	m.RefreshRate = int(r.u32())
}

func (m *Input) encode(w *writer) {
	w.u64(m.Seq)
	w.u8(uint8(m.Kind))
	w.f64(m.X)
	w.f64(m.Y)
	w.f64(m.Line)
	w.i32(m.Button)
	w.i32(m.Key)
	w.bool(m.Pressed)
	w.bool(m.Repeat)
	w.bool(m.Entered)
	w.bool(m.Precise)
	w.u8(uint8(m.Mods))
	w.i32(m.Phase)
	w.str(m.Text)
	w.i32(m.Start)
	w.i32(m.End)
	w.bool(m.Done)
	w.bool(m.HasReplacement)
	w.bool(m.Passthrough)
	w.u32(uint32(len(m.Paths)))
	for _, p := range m.Paths {
		w.str(p)
	}
}

func (m *Input) decode(r *reader) {
	m.Seq = r.u64()
	m.Kind = InputKind(r.u8())
	m.X, m.Y, m.Line = r.f64(), r.f64(), r.f64()
	m.Button, m.Key = r.i32(), r.i32()
	m.Pressed, m.Repeat, m.Entered, m.Precise = r.bool(), r.bool(), r.bool(), r.bool()
	m.Mods = Mods(r.u8())
	m.Phase = r.i32()
	m.Text = r.str()
	m.Start, m.End = r.i32(), r.i32()
	m.Done, m.HasReplacement, m.Passthrough = r.bool(), r.bool(), r.bool()
	if n := r.count(4); n > 0 {
		m.Paths = make([]string, n)
		for i := range m.Paths {
			m.Paths[i] = r.str()
		}
	}
}

func (*Close) encode(*writer) {}
func (*Close) decode(*reader) {}

func (m *Ack) encode(w *writer) { w.u64(m.Seq) }
func (m *Ack) decode(r *reader) { m.Seq = r.u64() }

func (m *Pixels) encode(w *writer) {
	w.u64(m.Seq)
	w.str(m.Err)
	w.u32(uint32(len(m.Pixels)))
	for _, p := range m.Pixels {
		w.bytes(p)
	}
}

func (m *Pixels) decode(r *reader) {
	m.Seq = r.u64()
	m.Err = r.str()
	n := r.count(4)
	m.Pixels = make([][]byte, n)
	for i := range m.Pixels {
		m.Pixels[i] = r.bytes()
	}
}

func (m *SetTitle) encode(w *writer) { w.str(m.Title) }
func (m *SetTitle) decode(r *reader) { m.Title = r.str() }

func (m *SetCursor) encode(w *writer) { w.u8(m.Shape); w.u8(m.Mode) }
func (m *SetCursor) decode(r *reader) { m.Shape, m.Mode = r.u8(), r.u8() }

func (m *SharedMemory) encode(w *writer) { w.u64(uint64(m.Size)); w.str(m.Name) }
func (m *SharedMemory) decode(r *reader) { m.Size = int64(r.u64()); m.Name = r.str() }

func (m *Batch) encode(w *writer) {
	w.u64(m.Seq)
	w.u64(m.InputSeq)
	w.bool(m.Present)
	w.u64(uint64(m.Offset))
	w.u64(uint64(m.Length))
	w.bool(m.Read != nil)
	if m.Read != nil {
		w.u32(m.Read.Image)
		w.u32(uint32(len(m.Read.Regions)))
		for _, rect := range m.Read.Regions {
			w.rect(rect)
		}
	}
}

func (m *Batch) decode(r *reader) {
	m.Seq, m.InputSeq = r.u64(), r.u64()
	m.Present = r.bool()
	m.Offset, m.Length = int64(r.u64()), int64(r.u64())
	if r.bool() {
		m.Read = &ReadRequest{Image: r.u32()}
		m.Read.Regions = make([]image.Rectangle, r.count(16))
		for i := range m.Read.Regions {
			m.Read.Regions[i] = r.rect()
		}
	}
}

func (m *Idle) encode(w *writer) { w.u64(m.InputSeq) }
func (m *Idle) decode(r *reader) { m.InputSeq = r.u64() }

func (*Closed) encode(*writer) {}
func (*Closed) decode(*reader) {}

// Encode returns m's frame.
func Encode(m Message) []byte {
	w := &writer{b: make([]byte, 5, 64)}
	w.b[4] = m.typ()
	m.encode(w)
	binary.LittleEndian.PutUint32(w.b, uint32(len(w.b)-4))
	return w.b
}

// Decode decodes the message of a frame without its length: its type byte and its payload.
func Decode(frame []byte) (Message, error) {
	if len(frame) == 0 {
		return nil, errors.New("remote: an empty frame")
	}
	m := newMessage(frame[0])
	if m == nil {
		return nil, fmt.Errorf("remote: unknown message type %d", frame[0])
	}
	r := &reader{b: frame[1:]}
	m.decode(r)
	if r.err != nil {
		return nil, fmt.Errorf("remote: a message of type %d: %w", frame[0], r.err)
	}
	if len(r.b) != r.p {
		return nil, fmt.Errorf("remote: a message of type %d has %d bytes past its end", frame[0], len(r.b)-r.p)
	}
	return m, nil
}

type writer struct{ b []byte }

func (w *writer) u8(v uint8)   { w.b = append(w.b, v) }
func (w *writer) u16(v uint16) { w.b = binary.LittleEndian.AppendUint16(w.b, v) }
func (w *writer) u32(v uint32) { w.b = binary.LittleEndian.AppendUint32(w.b, v) }
func (w *writer) u64(v uint64) { w.b = binary.LittleEndian.AppendUint64(w.b, v) }
func (w *writer) i32(v int)    { w.u32(uint32(int32(v))) }
func (w *writer) f64(v float64) {
	w.u64(math.Float64bits(v))
}
func (w *writer) bool(v bool) {
	if v {
		w.u8(1)
	} else {
		w.u8(0)
	}
}
func (w *writer) str(s string)   { w.u32(uint32(len(s))); w.b = append(w.b, s...) }
func (w *writer) bytes(b []byte) { w.u32(uint32(len(b))); w.b = append(w.b, b...) }
func (w *writer) rect(r image.Rectangle) {
	w.i32(r.Min.X)
	w.i32(r.Min.Y)
	w.i32(r.Max.X)
	w.i32(r.Max.Y)
}

var errShort = errors.New("cut short")

// reader reads a payload. A read past the end sets err and returns zeros.
type reader struct {
	b   []byte
	p   int
	err error
}

func (r *reader) take(n int) []byte {
	if r.err != nil {
		return nil
	}
	if n < 0 || n > len(r.b)-r.p {
		r.err = errShort
		return nil
	}
	v := r.b[r.p : r.p+n]
	r.p += n
	return v
}

func (r *reader) u8() uint8 {
	if b := r.take(1); b != nil {
		return b[0]
	}
	return 0
}

func (r *reader) u16() uint16 {
	if b := r.take(2); b != nil {
		return binary.LittleEndian.Uint16(b)
	}
	return 0
}

func (r *reader) u32() uint32 {
	if b := r.take(4); b != nil {
		return binary.LittleEndian.Uint32(b)
	}
	return 0
}

func (r *reader) u64() uint64 {
	if b := r.take(8); b != nil {
		return binary.LittleEndian.Uint64(b)
	}
	return 0
}

func (r *reader) i32() int { return int(int32(r.u32())) }

// f64 reads a float, which must be finite: no message has a use for a NaN or an infinity, and a
// viewer or an app sizing something by one would break.
func (r *reader) f64() float64 {
	v := math.Float64frombits(r.u64())
	if (math.IsNaN(v) || math.IsInf(v, 0)) && r.err == nil {
		r.err = fmt.Errorf("a float of %v", v)
	}
	return v
}

func (r *reader) bool() bool {
	switch v := r.u8(); v {
	case 0:
		return false
	case 1:
		return true
	default:
		if r.err == nil {
			r.err = fmt.Errorf("a bool of %d", v)
		}
		return false
	}
}

func (r *reader) bytes() []byte {
	n := r.u32()
	if b := r.take(int(n)); b != nil {
		return append([]byte(nil), b...)
	}
	return nil
}

func (r *reader) str() string { return string(r.bytes()) }

func (r *reader) rect() image.Rectangle {
	x0, y0, x1, y1 := r.i32(), r.i32(), r.i32(), r.i32()
	return image.Rect(x0, y0, x1, y1)
}

// count reads a count of items each at least size bytes long that must fit in what is left.
func (r *reader) count(size int) int {
	n := r.u32()
	if r.err != nil {
		return 0
	}
	if uint64(n)*uint64(size) > uint64(len(r.b)-r.p) {
		r.err = errShort
		return 0
	}
	return int(n)
}
