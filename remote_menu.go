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

package ggwd

import "github.com/ironpark/ggwd/internal/ui"

// RemoteMenuItem describes one entry of an app's native menu bar when running under a remote
// viewer: an action, a separator, or a submenu when Items is non-empty.
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

func toUIRemoteMenuItems(items []RemoteMenuItem) []ui.RemoteMenuItem {
	if items == nil {
		return nil
	}
	out := make([]ui.RemoteMenuItem, len(items))
	for i := range items {
		it := &items[i]
		out[i] = ui.RemoteMenuItem{
			Title:     it.Title,
			Separator: it.Separator,
			Items:     toUIRemoteMenuItems(it.Items),
			ID:        it.ID,
			Key:       it.Key,
			Cmd:       it.Cmd,
			Shift:     it.Shift,
			Alt:       it.Alt,
			Ctrl:      it.Ctrl,
			Enabled:   it.Enabled,
			Checked:   it.Checked,
		}
	}
	return out
}

// SetRemoteMenu replaces the remote viewer's native menu bar with items. An empty slice clears
// the menu bar. It does nothing when the app is not running remotely.
func SetRemoteMenu(items []RemoteMenuItem) {
	ui.Get().SetRemoteMenu(toUIRemoteMenuItems(items))
}

// SetRemoteMenuItemEnabled enables or disables the remote menu action with the given ID. It does
// nothing when the app is not running remotely.
func SetRemoteMenuItemEnabled(id int, enabled bool) {
	ui.Get().SetRemoteMenuItemEnabled(id, enabled)
}

// SetRemoteMenuItemChecked sets or clears the check mark on the remote menu action with the given
// ID. It does nothing when the app is not running remotely.
func SetRemoteMenuItemChecked(id int, checked bool) {
	ui.Get().SetRemoteMenuItemChecked(id, checked)
}

// SetRemoteMenuHandler registers fn to run when the user activates a menu action in the remote
// viewer. It can be called before or during Run; passing nil removes the handler.
func SetRemoteMenuHandler(fn func(id int)) {
	ui.Get().SetRemoteMenuHandler(fn)
}
