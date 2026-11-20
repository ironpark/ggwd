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

import "github.com/ironpark/ggwd/internal/colormode"

// RemoteMenuItem describes one entry of an app's native menu bar forwarded to a remote viewer.
type RemoteMenuItem struct {
	Title                 string
	Separator             bool
	Items                 []RemoteMenuItem
	ID                    int
	Key                   string
	Cmd, Shift, Alt, Ctrl bool
	Enabled               bool
	Checked               bool
}

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
	setPreferredColorMode(mode colormode.ColorMode)
	setDockIcon(visible bool)
	setMenu(items []RemoteMenuItem)
	setMenuItemEnabled(id int, enabled bool)
	setMenuItemChecked(id int, checked bool)
}

// newRemoteBackend is set by the remote backend, when it is linked in. It returns nil unless the
// environment asks for the app to run remotely.
var newRemoteBackend func(u *UserInterface) remoteBackend

// Remote reports whether the app runs remotely.
func (u *UserInterface) Remote() bool {
	return u.remote != nil
}

// startRemote makes u run remotely if the remote backend is linked in and the environment asks
// for it, and reports whether it does. RunApp calls it before anything of the desktop is set up.
func (u *UserInterface) startRemote() bool {
	if newRemoteBackend == nil {
		return false
	}
	u.remote = newRemoteBackend(u)
	return u.remote != nil
}

func cloneRemoteMenuItems(items []RemoteMenuItem) []RemoteMenuItem {
	if items == nil {
		return nil
	}
	out := make([]RemoteMenuItem, len(items))
	for i := range items {
		out[i] = items[i]
		out[i].Items = cloneRemoteMenuItems(items[i].Items)
	}
	return out
}

// SetRemoteMenu replaces the remote viewer's native menu bar. It does nothing when not running
// remotely.
func (u *UserInterface) SetRemoteMenu(items []RemoteMenuItem) {
	if u.remote != nil {
		u.remote.setMenu(items)
		return
	}
	cp := cloneRemoteMenuItems(items)
	u.remoteMenuMu.Lock()
	u.pendingRemoteMenu = &cp
	u.remoteMenuMu.Unlock()
}

// SetRemoteMenuItemEnabled updates whether menu action id is enabled in the remote viewer.
func (u *UserInterface) SetRemoteMenuItemEnabled(id int, enabled bool) {
	if u.remote != nil {
		u.remote.setMenuItemEnabled(id, enabled)
	}
}

// SetRemoteMenuItemChecked updates whether menu action id is checked in the remote viewer.
func (u *UserInterface) SetRemoteMenuItemChecked(id int, checked bool) {
	if u.remote != nil {
		u.remote.setMenuItemChecked(id, checked)
	}
}

// SetRemoteMenuHandler registers fn to be called when the remote viewer activates a menu action.
func (u *UserInterface) SetRemoteMenuHandler(fn func(id int)) {
	u.remoteMenuMu.Lock()
	u.remoteMenuFn = fn
	u.remoteMenuMu.Unlock()
}
