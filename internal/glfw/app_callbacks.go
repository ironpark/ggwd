// Copyright 2026 The ggfx Authors
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

//go:build darwin || freebsd || linux || netbsd || windows

package glfw

// SetQuitRequestCallback sets the callback run when the user asks the app to quit, as with
// Command-Q or the Dock's Quit, before every window gets its close callback. It is how the
// app tells a quit from a window's close button. This is a ggwd extension; GLFW has no such
// callback, and only macOS reports it.
func SetQuitRequestCallback(cbfun func()) (func(), error) {
	if !_glfw.initialized {
		return nil, NotInitialized
	}
	old := _glfw.callbacks.quitRequest
	_glfw.callbacks.quitRequest = cbfun
	return old, nil
}

// SetReopenCallback sets the callback run when the user opens the running app again, as by
// clicking its Dock icon: the way back to a window the app hid. This is a ggwd extension; GLFW
// has no such callback, and only macOS reports it.
func SetReopenCallback(cbfun func()) (func(), error) {
	if !_glfw.initialized {
		return nil, NotInitialized
	}
	old := _glfw.callbacks.reopen
	_glfw.callbacks.reopen = cbfun
	return old, nil
}

func inputQuitRequest() {
	if cb := _glfw.callbacks.quitRequest; cb != nil {
		cb()
	}
}

func inputReopen() {
	if cb := _glfw.callbacks.reopen; cb != nil {
		cb()
	}
}
