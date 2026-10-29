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

package ui

import "sync/atomic"

var tick atomic.Int64

// Tick returns the number of iterations of the loop so far: the clock that the current monitor's
// cache and the text input's state queue expire against.
func Tick() int64 {
	return tick.Load()
}

// advanceTick advances Tick. The loop calls it once per iteration.
func advanceTick() {
	tick.Add(1)
}
