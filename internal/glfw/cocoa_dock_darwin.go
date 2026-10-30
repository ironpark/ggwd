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

package glfw

// SetDockIconVisible puts the app's icon in the Dock and its menu in the menu bar, or takes
// both out, as an app that lives in the menu bar's status area does while it shows no window.
// It must be called on the main thread. This is a ggwd extension.
func SetDockIconVisible(visible bool) {
	policy := _NSApplicationActivationPolicyAccessory
	if visible {
		policy = _NSApplicationActivationPolicyRegular
	}
	nsApp().Send(sel_setActivationPolicy, policy)
}
