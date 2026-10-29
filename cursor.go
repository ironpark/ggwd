// Copyright 2019 The Ebiten Authors
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

import (
	"github.com/ironpark/ggwd/internal/ui"
)

// CursorMode is how a window shows the mouse cursor and reports its moves.
type CursorMode int

// The cursor modes.
const (
	CursorModeVisible  CursorMode = CursorMode(ui.CursorModeVisible)
	CursorModeHidden   CursorMode = CursorMode(ui.CursorModeHidden)
	CursorModeCaptured CursorMode = CursorMode(ui.CursorModeCaptured)
)

// CursorShape is a shape of the mouse cursor.
type CursorShape int

// The cursor shapes.
const (
	CursorShapeDefault    CursorShape = CursorShape(ui.CursorShapeDefault)
	CursorShapeText       CursorShape = CursorShape(ui.CursorShapeText)
	CursorShapeCrosshair  CursorShape = CursorShape(ui.CursorShapeCrosshair)
	CursorShapePointer    CursorShape = CursorShape(ui.CursorShapePointer)
	CursorShapeEWResize   CursorShape = CursorShape(ui.CursorShapeEWResize)
	CursorShapeNSResize   CursorShape = CursorShape(ui.CursorShapeNSResize)
	CursorShapeNESWResize CursorShape = CursorShape(ui.CursorShapeNESWResize)
	CursorShapeNWSEResize CursorShape = CursorShape(ui.CursorShapeNWSEResize)
	CursorShapeMove       CursorShape = CursorShape(ui.CursorShapeMove)
	CursorShapeNotAllowed CursorShape = CursorShape(ui.CursorShapeNotAllowed)
)
