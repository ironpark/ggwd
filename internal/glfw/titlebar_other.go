// Copyright 2026 The ggwd Authors
// SPDX-License-Identifier: Apache-2.0

//go:build freebsd || linux || netbsd || windows

package glfw

// Only a Cocoa window's title bar can be hidden; elsewhere these do nothing.

// SetTrafficLights does nothing: only macOS has traffic lights.
func (w *Window) SetTrafficLights(x, y float64) {}

// StartDrag does nothing: only a Cocoa window's title bar can be hidden.
func (w *Window) StartDrag() {}

// TitlebarDoubleClick does nothing: only a Cocoa window's title bar can be hidden.
func (w *Window) TitlebarDoubleClick() {}
