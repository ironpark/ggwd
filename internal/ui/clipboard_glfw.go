// Copyright 2026 The ggfx Authors
// SPDX-License-Identifier: Apache-2.0

//go:build !android && !ios && !js && !nintendosdk && !playstation5

package ui

import (
	"errors"

	"github.com/ironpark/ggwd/internal/glfw"
)

var errClipboardNotRunning = errors.New("ui: the clipboard needs the loop to run")

// ClipboardText returns the text on the system clipboard.
func (u *UserInterface) ClipboardText() (string, error) {
	text, err := "", errClipboardNotRunning
	u.RunOnMainThread(func() { text, err = glfw.GetClipboardString() })
	if errors.Is(err, glfw.FormatUnavailable) {
		// The clipboard holds no text.
		return "", nil
	}
	return text, err
}

// SetClipboardText puts text on the system clipboard.
func (u *UserInterface) SetClipboardText(text string) error {
	err := errClipboardNotRunning
	// The clipboard belongs to the process, not to the window the call names.
	u.RunOnMainThread(func() { err = (*glfw.Window)(nil).SetClipboardString(text) })
	return err
}
