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

// Package remote is the protocol between an app whose window is shown by another process, the
// viewer, and that viewer. The viewer owns the real window and the GPU; the app opens neither.
// Its frames are drawn by ggfx's remote driver (github.com/ironpark/ggfx/remote), whose batches
// of driver calls this protocol carries, and its input comes from the viewer's window.
//
// An app runs remotely when it is built with the ggwd_remote build tag and EnvSocket names the
// viewer's socket when it starts; otherwise it opens its windows as usual. The tag only links the
// remote backend in: a build with it and the variable unset is an ordinary desktop app.
//
// # Connection
//
// The app connects to the unix socket EnvSocket names, retrying for DialTimeout while the viewer
// starts. It does not reconnect: an app that finds no viewer, or a viewer of another protocol
// version, fails to start with an error, and an app whose viewer goes away is closed, its window
// gone, and ends with ErrViewerGone. A new app process connects anew; the viewer keeps its window
// and shows the new app in it.
//
//	viewer                          app
//	Welcome (version, driver info) →
//	                               ← Hello (version, pid)
//	                               ← OpenWindow (title, size, ...)
//	Config (size, scale, ...)      →
//	                               ← SharedMemory (size, the file)
//	                               ← Batch (seq, where in the shared memory)
//	Ack (seq)                      →
//	Input, Config, Close           →
//	                               ← Batch, Idle, SetTitle, SetCursor, Closed
//
// The viewer acks a Batch once it has replayed it into the image it shows. The app draws a frame
// when the viewer has acked every batch, so that an animation runs no faster than the viewer
// shows it, or, with one batch in flight, when there is input that batch does not answer, which
// is then drawn at once rather than a viewer's frame later; never with two batches in flight. The
// shared memory has a slot for each: batch n is at offset (n%2)·size/2. A Batch with a
// ReadRequest asks the viewer to replay it and read pixels back, which the viewer answers with
// Pixels instead of Ack.
//
// The viewer's Close asks the app to close its window; the app may refuse, as on a window of its
// own, and says Closed when it does close.
//
// # Wire format
//
// Every message is a frame: its length as a u32 (the type byte and the payload), the type byte,
// the payload. Numbers are little-endian; a float is an IEEE 754 float64; a bool is a byte, 0 or
// 1; a string or a byte array is its length as a u32 and its bytes; a rectangle is four i32. The
// payload of each type is its struct's fields in order, as Encode writes them, and Version
// changes whenever any of them does. SharedMemory passes its file as an SCM_RIGHTS control message
// with its frame. A frame is at most MaxFrameSize bytes.
//
// The batches themselves are ggfx's wire format, versioned on their own: the Welcome's driver info
// carries that version, and the app refuses a viewer of another.
package remote
