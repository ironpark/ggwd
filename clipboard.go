// Copyright 2026 The ggfx Authors
// SPDX-License-Identifier: Apache-2.0

package ggwd

import "github.com/ironpark/ggwd/internal/ui"

// ClipboardText returns the text on the system clipboard: the CLIPBOARD
// selection on X11, the general pasteboard on macOS and the Unicode text on
// Windows, or "" when it holds none. It needs Run to have started. In the
// browser it returns an error that wraps errors.ErrUnsupported.
//
// ClipboardText can be called from any goroutine.
func ClipboardText() (string, error) {
	return ui.Get().ClipboardText()
}

// SetClipboardText puts text on the system clipboard; see ClipboardText. On
// X11 the app owns what it puts there, so it is gone once the app ends unless
// a clipboard manager took a copy, as desktops' usually do. Text with a NUL
// character cannot be put on the Windows clipboard.
//
// SetClipboardText can be called from any goroutine.
func SetClipboardText(text string) error {
	return ui.Get().SetClipboardText(text)
}
