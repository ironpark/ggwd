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

// Package windowbridge carries to ggwd/native the window system's side of a *ggwd.Window.
package windowbridge

import "github.com/ironpark/ggwd/internal/ui"

// UI returns the window system's window of w, a *ggwd.Window. ggwd sets it.
var UI func(w any) ui.AppWindow
