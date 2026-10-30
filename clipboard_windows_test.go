// Copyright 2026 The ggfx Authors
// SPDX-License-Identifier: Apache-2.0

package ggwd_test

import (
	"testing"

	"github.com/ironpark/ggwd"
)

// TestClipboardRoundTrip puts text on the Windows clipboard and reads it back, and then puts back
// what was there.
func TestClipboardRoundTrip(t *testing.T) {
	old, err := ggwd.ClipboardText()
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := ggwd.SetClipboardText(old); err != nil {
			t.Error(err)
		}
	}()

	const want = "ggwd 클립보드 😀"
	if err := ggwd.SetClipboardText(want); err != nil {
		t.Fatal(err)
	}
	got, err := ggwd.ClipboardText()
	if err != nil {
		t.Fatal(err)
	}
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}

	if err := ggwd.SetClipboardText("a\x00b"); err == nil {
		t.Error("SetClipboardText with a NUL character: got no error")
	}
}
