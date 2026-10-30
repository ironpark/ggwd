// Copyright 2026 The ggfx Authors
// SPDX-License-Identifier: Apache-2.0

package ggwd

import (
	"go/ast"
	"go/parser"
	"go/token"
	"reflect"
	"slices"
	"testing"
	"testing/fstest"

	"github.com/ironpark/ggwd/internal/ui"
)

// uiEvents has one of every event of internal/ui. TestEventTypesMatch fails when one is missing.
var uiEvents = []ui.Event{
	ui.StartEvent{},
	ui.WakeEvent{},
	ui.SystemColorModeEvent{},
	ui.MonitorEvent{},
	ui.QuitEvent{},
	ui.ReopenEvent{},
	ui.FrameEvent{},
	ui.ResizeEvent{},
	ui.MoveEvent{},
	ui.FocusEvent{},
	ui.WindowStateEvent{},
	ui.CloseEvent{},
	ui.ClosedEvent{},
	ui.KeyEvent{},
	ui.TextEvent{},
	ui.CompositionEvent{},
	ui.DragEvent{},
	ui.MouseMoveEvent{},
	ui.MouseEnterEvent{},
	ui.MouseButtonEvent{},
	ui.ScrollEvent{},
	ui.TouchEvent{},
	ui.DropEvent{},
}

// renamedFields are the fields of a ui event that its public event has under other names.
var renamedFields = map[string]map[string]string{
	"TextEvent": {
		"HasReplacement":   "Replacement",
		"ReplacementStart": "Replacement",
		"ReplacementEnd":   "Replacement",
	},
}

// eventFields returns the exported fields of the types in the given files that have an isEvent
// method, by type name.
func eventFields(t *testing.T, files ...string) map[string][]string {
	t.Helper()
	fset := token.NewFileSet()
	structs := map[string][]string{}
	events := map[string]bool{}
	for _, name := range files {
		f, err := parser.ParseFile(fset, name, nil, 0)
		if err != nil {
			t.Fatal(err)
		}
		for _, d := range f.Decls {
			switch d := d.(type) {
			case *ast.FuncDecl:
				if d.Name.Name != "isEvent" || d.Recv == nil {
					continue
				}
				if id, ok := d.Recv.List[0].Type.(*ast.Ident); ok {
					events[id.Name] = true
				}
			case *ast.GenDecl:
				for _, s := range d.Specs {
					ts, ok := s.(*ast.TypeSpec)
					if !ok {
						continue
					}
					st, ok := ts.Type.(*ast.StructType)
					if !ok {
						continue
					}
					fields := []string{}
					for _, f := range st.Fields.List {
						for _, n := range f.Names {
							if n.IsExported() {
								fields = append(fields, n.Name)
							}
						}
					}
					structs[ts.Name.Name] = fields
				}
			}
		}
	}
	out := map[string][]string{}
	for name := range events {
		out[name] = structs[name]
	}
	return out
}

// TestEventTypesMatch checks that the public events and internal/ui's are the same events with
// the same fields, so that an event or a field added to one side is not lost on the other.
func TestEventTypesMatch(t *testing.T) {
	internal := eventFields(t, "internal/ui/event.go")
	public := eventFields(t, "events.go")

	for name, fields := range internal {
		pub, ok := public[name]
		if !ok {
			t.Errorf("ui.%s has no public event", name)
			continue
		}
		for _, f := range fields {
			if r, ok := renamedFields[name][f]; ok {
				f = r
			}
			if !slices.Contains(pub, f) {
				t.Errorf("ui.%s.%s has no field in %s", name, f, name)
			}
		}
		for _, f := range pub {
			renamed := false
			for _, r := range renamedFields[name] {
				renamed = renamed || r == f
			}
			if !renamed && !slices.Contains(fields, f) {
				t.Errorf("%s.%s has no field in ui.%s", name, f, name)
			}
		}
	}
	for name := range public {
		if _, ok := internal[name]; !ok {
			t.Errorf("%s has no ui event", name)
		}
	}

	var listed []string
	for _, ev := range uiEvents {
		listed = append(listed, reflect.TypeOf(ev).Name())
	}
	for name := range internal {
		if !slices.Contains(listed, name) {
			t.Errorf("ui.%s is missing in uiEvents", name)
		}
	}
}

// TestEventConversionKeepsFields checks that eventFromUI carries every field of every ui event
// over: each one is set, and must not come out as its zero value.
func TestEventConversionKeepsFields(t *testing.T) {
	w := new(Window)
	window := eventTestWindow{public: w}
	for _, ev := range uiEvents {
		in := reflect.New(reflect.TypeOf(ev)).Elem()
		for i := range in.NumField() {
			f := in.Field(i)
			if !f.CanSet() {
				continue
			}
			switch {
			case f.Type() == reflect.TypeFor[ui.AppWindow]():
				f.Set(reflect.ValueOf(window))
			case f.Kind() == reflect.String:
				f.SetString("x")
			case f.Kind() == reflect.Bool:
				f.SetBool(true)
			case f.CanInt():
				f.SetInt(1)
			case f.CanFloat():
				f.SetFloat(1.5)
			case f.Kind() == reflect.Struct:
				// KeyModifiers: every modifier held.
				for j := range f.NumField() {
					f.Field(j).SetBool(true)
				}
			case f.Type().String() == "fs.FS":
				f.Set(reflect.ValueOf(fstest.MapFS{}))
			}
		}

		out := reflect.ValueOf(eventFromUI(in.Interface().(ui.Event)))
		name := in.Type().Name()
		if out.Type().Name() != name {
			t.Errorf("ui.%s became %s", name, out.Type().Name())
			continue
		}
		for i := range in.NumField() {
			sf := in.Type().Field(i)
			if !sf.IsExported() || in.Field(i).IsZero() {
				continue
			}
			field := sf.Name
			if r, ok := renamedFields[name][field]; ok {
				field = r
			}
			if o := out.FieldByName(field); !o.IsValid() || o.IsZero() {
				t.Errorf("%s.%s is lost in the conversion", name, field)
			}
		}
		if wf := out.FieldByName("Window"); wf.IsValid() && wf.Interface() != w {
			t.Errorf("%s.Window is not the public window", name)
		}
	}
}
