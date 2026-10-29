// Copyright 2026 The ggip Authors
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

package ggwd_test

import (
	"testing"

	"github.com/ironpark/ggwd"
)

func TestKeyCodeRoundTrip(t *testing.T) {
	for k := ggwd.Key(0); k <= ggwd.KeyMax; k++ {
		code := k.Code()
		switch k {
		case ggwd.KeyAlt, ggwd.KeyControl, ggwd.KeyShift, ggwd.KeyMeta:
			if code != "" {
				t.Errorf("%v.Code() = %q; want none for a key that stands for either side", k, code)
			}
			continue
		}
		if code == "" {
			t.Errorf("%v has no code", k)
			continue
		}
		if got, ok := ggwd.KeyFromCode(code); !ok || got != k {
			t.Errorf("KeyFromCode(%q) = %v, %v; want %v", code, got, ok, k)
		}
	}
	if code := ggwd.KeyA.Code(); code != "KeyA" {
		t.Errorf("KeyA.Code() = %q; want KeyA", code)
	}
}

func TestKeyNameRoundTrip(t *testing.T) {
	for k := ggwd.Key(0); k <= ggwd.KeyMax; k++ {
		text, err := k.MarshalText()
		if err != nil {
			t.Fatal(err)
		}
		var got ggwd.Key
		if err := got.UnmarshalText(text); err != nil {
			t.Fatal(err)
		}
		if got != k {
			t.Errorf("text %q unmarshals to %v; want %v", text, got, k)
		}
		if got, ok := ggwd.KeyFromName(k.String()); !ok || got != k {
			t.Errorf("KeyFromName(%q) = %v, %v; want %v", k.String(), got, ok, k)
		}
	}
}
