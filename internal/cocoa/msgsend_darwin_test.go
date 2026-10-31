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

package cocoa

import (
	"testing"

	"github.com/ebitengine/purego/objc"
)

// Send and SendRect give what objc.Send does, without registering a function at each call.
func TestSendGivesWhatObjcSendDoes(t *testing.T) {
	screen := objc.ID(objc.GetClass("NSScreen")).Send(objc.RegisterName("mainScreen"))
	if screen == 0 {
		t.Skip("no screen")
	}
	frame, backing := objc.RegisterName("frame"), objc.RegisterName("convertRectToBacking:")
	want := objc.Send[NSRect](screen, frame)
	if got := SendRect(screen, frame); got != want {
		t.Fatalf("SendRect %+v, objc.Send %+v", got, want)
	}
	if got := Send[NSRect](screen, frame); got != want {
		t.Fatalf("Send %+v, objc.Send %+v", got, want)
	}
	wantPixels := objc.Send[NSRect](screen, backing, want)
	if got := SendRectRect(screen, backing, want); got != wantPixels {
		t.Fatalf("SendRectRect %+v, objc.Send %+v", got, wantPixels)
	}
	scale := objc.RegisterName("backingScaleFactor")
	if got, want := Send[float64](screen, scale), objc.Send[float64](screen, scale); got != want {
		t.Fatalf("Send[float64] %v, objc.Send %v", got, want)
	}
	cached := testing.AllocsPerRun(100, func() { SendRect(screen, frame) })
	fresh := testing.AllocsPerRun(100, func() { objc.Send[NSRect](screen, frame) })
	if cached >= fresh {
		t.Fatalf("SendRect allocates %v a call, objc.Send %v", cached, fresh)
	}
	t.Logf("allocations a call: SendRect %v, Send %v, objc.Send %v", cached,
		testing.AllocsPerRun(100, func() { Send[NSRect](screen, frame) }), fresh)
}
