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

package ggwd

import "github.com/ironpark/ggwd/internal/ui"

// SetDockIcon puts the app's icon in the Dock and its menu in the menu bar, or takes both out,
// as an app that lives in the menu bar's status area does while it shows no window. Without
// them the app is not in the Command-Tab switcher either, and its windows take no menu bar of
// their own. It does nothing but on macOS.
//
// SetDockIcon can be called from any goroutine while Run runs.
func SetDockIcon(visible bool) {
	ui.Get().SetDockIcon(visible)
}
