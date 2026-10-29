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

// Package gamepad reads the gamepads: IOKit and the GameController framework on macOS,
// DirectInput and XInput on Windows, evdev on Linux and the Gamepad API in the browser.
//
// Buttons and axes can only be polled. [Update] reads every gamepad, and a [Tracker] compares the
// poll with the previous one and reports the differences as events. Poll every
// [Tracker.WaitTimeout]: often while a gamepad is connected, once a second otherwise.
//
// The package does not know about windows or event loops; the caller wires it to its own:
//
//   - Call [Update] on the main thread. macOS schedules its device callbacks on the main run
//     loop and runs it once while connecting, and Windows subclasses a window of that thread.
//   - On Windows, pass a window handle (an HWND) to [Update]. Its window procedure is
//     subclassed once, on the first Update, to learn of devices arriving and leaving
//     (WM_DEVICECHANGE), so pass a window that lives as long as the gamepads are read. Other
//     platforms ignore the handle.
//   - Hand the events to the loop that handles input, and wake it if it sleeps.
package gamepad
