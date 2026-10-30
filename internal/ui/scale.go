// Copyright 2022 The Ebiten Authors
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

// LogicalPositionToClientPositionInNativePixels converts a logical position in the focused window
// to a client-area position in native pixels. A logical position is already a client position in
// device-independent pixels: an app window has no letterboxing.
func (u *UserInterface) LogicalPositionToClientPositionInNativePixels(x, y float64) (float64, float64) {
	s := u.monitorDeviceScaleFactor()
	x = dipToNativePixels(x, s)
	y = dipToNativePixels(y, s)
	return x, y
}
