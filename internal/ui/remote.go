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

package ui

// remoteBackend runs the app with its window shown by a viewer process (see package
// ggwd/remote). It is chosen at run time, by RunApp, among the backends of the build: the desktop
// backend's methods that an app reaches once it runs hand over to it first, which is all the
// desktop files know of it. A build without the ggwd_remote tag links none, and those hand-overs
// are never taken.
type remoteBackend interface {
	runApp(app App, options *RunOptions) error
	newWindow(o *WindowOptions, handle any) (AppWindow, error)
	wakeLoop()
	runOnMainThread(f func())
	appendMonitors(monitors []*Monitor) []*Monitor
	monitorDeviceScaleFactor() float64
	clipboardText() (string, error)
	setClipboardText(text string) error
}

// newRemoteBackend is set by the remote backend, when it is linked in. It returns nil unless the
// environment asks for the app to run remotely.
var newRemoteBackend func(u *UserInterface) remoteBackend

// startRemote makes u run remotely if the remote backend is linked in and the environment asks
// for it, and reports whether it does. RunApp calls it before anything of the desktop is set up.
func (u *UserInterface) startRemote() bool {
	if newRemoteBackend == nil {
		return false
	}
	u.remote = newRemoteBackend(u)
	return u.remote != nil
}
