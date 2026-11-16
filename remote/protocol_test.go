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
	"image"
	"reflect"
	"testing"
)

var messages = []Message{
	&Welcome{Version: Version, Driver: []byte{1, 2, 3}},
	&Hello{Version: Version, PID: 4242},
	&OpenWindow{Title: "héllo", Width: 640, Height: 480.5, Resizable: true, MinWidth: -1, MinHeight: 100, MaxWidth: 2000, MaxHeight: -1},
	&Config{Width: 640, Height: 480, PixelWidth: 1280, PixelHeight: 960, Scale: 2, Focused: true, ColorMode: 2, MonitorWidth: 1512, MonitorHeight: 982, RefreshRate: 120},
	&Input{Seq: 7, Kind: InputScroll, X: 1.5, Y: -2, Line: 40, Button: 2, Key: 34, Pressed: true, Repeat: true, Entered: true, Precise: true, Mods: ModShift | ModMeta, Phase: 3, Text: "가"},
	&Close{},
	&Ack{Seq: 1 << 40},
	&Pixels{Seq: 3, Err: "no", Pixels: [][]byte{{1, 2}, {}, {3}}},
	&SetTitle{Title: "t"},
	&SetCursor{Shape: 3, Mode: 1},
	&SharedMemory{Size: 1 << 22},
	&Batch{Seq: 9, InputSeq: 8, Present: true, Offset: 16, Length: 1000},
	&Batch{Seq: 10, Read: &ReadRequest{Image: 5, Regions: []image.Rectangle{image.Rect(1, 2, 3, 4), image.Rect(-1, 0, 5, 9)}}},
	&Idle{InputSeq: 11},
	&Closed{},
}

func TestRoundTrip(t *testing.T) {
	for _, m := range messages {
		b := Encode(m)
		got, err := Decode(b[4:])
		if err != nil {
			t.Fatalf("%T: %v", m, err)
		}
		if p, ok := m.(*Pixels); ok {
			// An empty array decodes as nil.
			want := *p
			want.Pixels = [][]byte{{1, 2}, nil, {3}}
			m = &want
		}
		if !reflect.DeepEqual(got, m) {
			t.Errorf("%T: got %+v, want %+v", m, got, m)
		}
	}
}

func TestDecodeRejects(t *testing.T) {
	for _, m := range messages {
		b := Encode(m)[4:]
		for n := 1; n < len(b); n++ {
			if _, err := Decode(b[:n]); err == nil {
				t.Errorf("%T cut to %d of %d bytes decoded", m, n, len(b))
			}
		}
		if _, err := Decode(append(b, 0)); err == nil {
			t.Errorf("%T with a byte past its end decoded", m)
		}
	}
	if _, err := Decode([]byte{200}); err == nil {
		t.Error("an unknown type decoded")
	}
}

func FuzzDecode(f *testing.F) {
	for _, m := range messages {
		f.Add(Encode(m)[4:])
	}
	f.Fuzz(func(t *testing.T, b []byte) {
		m, err := Decode(b)
		if err != nil {
			return
		}
		// What decodes encodes back to a frame that decodes the same.
		again, err := Decode(Encode(m)[4:])
		if err != nil || !reflect.DeepEqual(again, m) {
			t.Errorf("%T does not survive a round trip: %v", m, err)
		}
	})
}
