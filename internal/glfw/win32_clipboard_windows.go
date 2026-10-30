// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2002-2006 Marcus Geelnard
// SPDX-FileCopyrightText: 2006-2019 Camilla Löwy <elmindreda@glfw.org>
// SPDX-FileCopyrightText: 2026 The ggfx Authors

package glfw

import (
	"errors"
	"fmt"
	"time"
	"unsafe"

	"golang.org/x/sys/windows"
)

const (
	_CF_UNICODETEXT = 13
	_GMEM_MOVEABLE  = 0x0002
)

var (
	procGlobalAlloc  = kernel32.NewProc("GlobalAlloc")
	procGlobalFree   = kernel32.NewProc("GlobalFree")
	procGlobalLock   = kernel32.NewProc("GlobalLock")
	procGlobalUnlock = kernel32.NewProc("GlobalUnlock")

	procCloseClipboard   = user32.NewProc("CloseClipboard")
	procEmptyClipboard   = user32.NewProc("EmptyClipboard")
	procGetClipboardData = user32.NewProc("GetClipboardData")
	procOpenClipboard    = user32.NewProc("OpenClipboard")
	procSetClipboardData = user32.NewProc("SetClipboardData")
)

// globalLock locks a global memory object and returns its memory, which is not Go's.
func globalLock(hMem uintptr) (unsafe.Pointer, error) {
	r, _, e := procGlobalLock.Call(hMem)
	if r == 0 {
		return nil, fmt.Errorf("glfw: GlobalLock failed: %w", e)
	}
	// Convert through a pointer to r, so that vet does not take r for a Go pointer kept as a
	// uintptr.
	return *(*unsafe.Pointer)(unsafe.Pointer(&r)), nil
}

func globalUnlock(hMem uintptr) {
	_, _, _ = procGlobalUnlock.Call(hMem)
}

// openClipboard opens the clipboard for the helper window. Another application can have it open,
// and so does the Windows clipboard history after each update, so it tries a few times.
func openClipboard() error {
	var err error
	for range 3 {
		r, _, e := procOpenClipboard.Call(uintptr(_glfw.platformWindow.helperWindowHandle))
		if r != 0 {
			return nil
		}
		err = e
		time.Sleep(time.Millisecond)
	}
	return fmt.Errorf("glfw: failed to open clipboard: %w: %w", err, PlatformError)
}

func closeClipboard() {
	_, _, _ = procCloseClipboard.Call()
}

func platformSetClipboardString(str string) error {
	text, err := windows.UTF16FromString(str)
	if err != nil {
		return fmt.Errorf("glfw: the clipboard string contains a NUL character: %w", InvalidValue)
	}
	size := uintptr(len(text)) * unsafe.Sizeof(text[0])

	object, _, e := procGlobalAlloc.Call(_GMEM_MOVEABLE, size)
	if object == 0 {
		return fmt.Errorf("glfw: failed to allocate global handle for clipboard: %w: %w", e, PlatformError)
	}
	// The clipboard owns the object once SetClipboardData succeeds; free it otherwise.
	owned := false
	defer func() {
		if !owned {
			_, _, _ = procGlobalFree.Call(object)
		}
	}()

	buffer, err := globalLock(object)
	if err != nil {
		return errors.Join(err, PlatformError)
	}
	copy(unsafe.Slice((*uint16)(buffer), len(text)), text)
	globalUnlock(object)

	if err := openClipboard(); err != nil {
		return err
	}
	defer closeClipboard()

	_, _, _ = procEmptyClipboard.Call()
	if r, _, e := procSetClipboardData.Call(_CF_UNICODETEXT, object); r == 0 {
		return fmt.Errorf("glfw: failed to set clipboard data: %w: %w", e, PlatformError)
	}
	owned = true
	return nil
}

func platformGetClipboardString() (string, error) {
	if err := openClipboard(); err != nil {
		return "", err
	}
	defer closeClipboard()

	object, _, _ := procGetClipboardData.Call(_CF_UNICODETEXT)
	if object == 0 {
		return "", fmt.Errorf("glfw: failed to convert clipboard to string: %w", FormatUnavailable)
	}
	buffer, err := globalLock(object)
	if err != nil {
		return "", errors.Join(err, PlatformError)
	}
	defer globalUnlock(object)
	return windows.UTF16PtrToString((*uint16)(buffer)), nil
}
