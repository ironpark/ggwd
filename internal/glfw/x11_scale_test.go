// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2026 The ggfx Authors

//go:build darwin || freebsd || linux || netbsd || windows

package glfw

import (
	"encoding/binary"
	"math"
	"testing"
)

type xsettingsEntry struct {
	typ  byte
	name string
	i32  int32
	str  string
	rgba [4]uint16
}

func buildXSettingsPayload(bo binary.AppendByteOrder, entries []xsettingsEntry) []byte {
	var buf []byte
	if bo == binary.LittleEndian {
		buf = append(buf, 'l')
	} else {
		buf = append(buf, 'B')
	}
	buf = append(buf, 0, 0, 0) // unused
	buf = bo.AppendUint32(buf, 1)
	buf = bo.AppendUint32(buf, uint32(len(entries)))

	for _, e := range entries {
		buf = append(buf, e.typ, 0)
		buf = bo.AppendUint16(buf, uint16(len(e.name)))
		buf = append(buf, e.name...)
		for len(buf)%4 != 0 {
			buf = append(buf, 0)
		}
		buf = bo.AppendUint32(buf, 1) // last-change-serial
		switch e.typ {
		case 0:
			buf = bo.AppendUint32(buf, uint32(e.i32))
		case 1:
			buf = bo.AppendUint32(buf, uint32(len(e.str)))
			buf = append(buf, e.str...)
			for len(buf)%4 != 0 {
				buf = append(buf, 0)
			}
		case 2:
			for _, c := range e.rgba {
				buf = bo.AppendUint16(buf, c)
			}
		}
	}
	return buf
}

func TestParsePositiveDPI(t *testing.T) {
	tests := []struct {
		in     string
		want   float32
		wantOK bool
	}{
		{"96", 1.0, true},
		{"120", 1.25, true},
		{"144", 1.5, true},
		{"192", 2.0, true},
		{" 168 \n", 1.75, true},
		{"", 0, false},
		{"   ", 0, false},
		{"0", 0, false},
		{"-96", 0, false},
		{"NaN", 0, false},
		{"+Inf", 0, false},
		{"invalid", 0, false},
		{"1e308", 0, false},
	}

	for _, tc := range tests {
		got, ok := parsePositiveDPI(tc.in)
		if ok != tc.wantOK {
			t.Errorf("parsePositiveDPI(%q) ok = %v, want %v", tc.in, ok, tc.wantOK)
			continue
		}
		if ok && math.Abs(float64(got-tc.want)) > 1e-6 {
			t.Errorf("parsePositiveDPI(%q) = %v, want %v", tc.in, got, tc.want)
		}
	}
}

func TestParseXSettingsScale(t *testing.T) {
	for _, tc := range []struct {
		name string
		bo   binary.AppendByteOrder
	}{
		{"little", binary.LittleEndian},
		{"big", binary.BigEndian},
	} {
		t.Run(tc.name, func(t *testing.T) {
			bo := tc.bo
			// Xft/DPI = 144 * 1024 (1.5x) after string and color entries.
			data := buildXSettingsPayload(bo, []xsettingsEntry{
				{typ: 1, name: "Net/ThemeName", str: "Adwaita"},
				{typ: 2, name: "Net/CursorColor", rgba: [4]uint16{0xffff, 0, 0, 0xffff}},
				{typ: 0, name: "Xft/DPI", i32: 144 * 1024},
			})
			got, ok := parseXSettingsScale(data)
			if !ok || math.Abs(float64(got-1.5)) > 1e-6 {
				t.Fatalf("parseXSettingsScale(Xft/DPI=144*1024) = (%v, %v), want (1.5, true)", got, ok)
			}

			// Gdk/WindowScalingFactor = 2 when Xft/DPI is unset (-1).
			data = buildXSettingsPayload(bo, []xsettingsEntry{
				{typ: 0, name: "Xft/DPI", i32: -1},
				{typ: 0, name: "Gdk/WindowScalingFactor", i32: 2},
			})
			got, ok = parseXSettingsScale(data)
			if !ok || got != 2 {
				t.Fatalf("parseXSettingsScale(Gdk/WindowScalingFactor=2) = (%v, %v), want (2, true)", got, ok)
			}

			// XFCE style: Xft/DPI = 96*1024 (1.0) with Gdk/WindowScalingFactor = 2.
			data = buildXSettingsPayload(bo, []xsettingsEntry{
				{typ: 0, name: "Xft/DPI", i32: 96 * 1024},
				{typ: 0, name: "Gdk/WindowScalingFactor", i32: 2},
			})
			got, ok = parseXSettingsScale(data)
			if !ok || got != 2 {
				t.Fatalf("parseXSettingsScale(Xft=96, Gdk=2) = (%v, %v), want (2, true)", got, ok)
			}

			// GNOME fractional: Xft/DPI = 144*1024 (1.5) with Gdk/WindowScalingFactor = 1.
			data = buildXSettingsPayload(bo, []xsettingsEntry{
				{typ: 0, name: "Xft/DPI", i32: 144 * 1024},
				{typ: 0, name: "Gdk/WindowScalingFactor", i32: 1},
			})
			got, ok = parseXSettingsScale(data)
			if !ok || math.Abs(float64(got-1.5)) > 1e-6 {
				t.Fatalf("parseXSettingsScale(Xft=144, Gdk=1) = (%v, %v), want (1.5, true)", got, ok)
			}
		})
	}

	// Malformed/truncated payloads must not panic and must return ok=false.
	malformed := [][]byte{
		nil,
		{},
		[]byte("short"),
		{'x', 0, 0, 0, 0, 0, 0, 0, 1, 0, 0, 0},
		{'l', 0, 0, 0, 0, 0, 0, 0, 0xff, 0xff, 0xff, 0xff},                                                   // huge count, no entries
		{'l', 0, 0, 0, 0, 0, 0, 0, 1, 0, 0, 0, 1, 0, 1, 0, 'a', 0, 0, 0, 0, 0, 0, 0, 0xff, 0xff, 0xff, 0xff}, // huge string length
		{'l', 0, 0, 0, 0, 0, 0, 0, 1, 0, 0, 0, 99, 0, 1, 0, 'a', 0, 0, 0, 0, 0, 0, 0},                        // unknown type
	}
	for i, bad := range malformed {
		if got, ok := parseXSettingsScale(bad); ok {
			t.Errorf("malformed[%d]: got (%v, true), want ok=false", i, got)
		}
	}
}

func TestResolveX11ContentScale(t *testing.T) {
	xsettings2x := buildXSettingsPayload(binary.LittleEndian, []xsettingsEntry{
		{typ: 0, name: "Xft/DPI", i32: 192 * 1024},
	})
	xsettings1x := buildXSettingsPayload(binary.LittleEndian, []xsettingsEntry{
		{typ: 0, name: "Xft/DPI", i32: 96 * 1024},
	})

	envMap := func(m map[string]string) func(string) string {
		return func(k string) string { return m[k] }
	}

	tests := []struct {
		name      string
		xftDPI    string
		xsettings []byte
		env       map[string]string
		want      float32
	}{
		{
			name:      "Xft.dpi wins over XSETTINGS and env",
			xftDPI:    "144",
			xsettings: xsettings2x,
			env: map[string]string{
				"GDK_SCALE":       "3",
				"QT_SCALE_FACTOR": "2.5",
			},
			want: 1.5,
		},
		{
			name:      "Xft.dpi unset falls back to non-1 XSETTINGS",
			xsettings: xsettings2x,
			env:       map[string]string{},
			want:      2.0,
		},
		{
			name: "Xft.dpi unset and no XSETTINGS falls back to GDK_SCALE",
			env: map[string]string{
				"GDK_SCALE":       "2",
				"QT_SCALE_FACTOR": "1.5",
			},
			want: 2.0,
		},
		{
			name: "Xft.dpi unset and no GDK_SCALE falls back to QT_SCALE_FACTOR",
			env: map[string]string{
				"QT_SCALE_FACTOR": "1.25",
			},
			want: 1.25,
		},
		{
			name:      "XSETTINGS 1.0 allows explicit QT_SCALE_FACTOR override",
			xsettings: xsettings1x,
			env: map[string]string{
				"QT_SCALE_FACTOR": "1.5",
			},
			want: 1.5,
		},
		{
			name:      "Invalid Xft.dpi and invalid GDK_SCALE fall back to QT_SCALE_FACTOR",
			xftDPI:    "0",
			xsettings: nil,
			env: map[string]string{
				"GDK_SCALE":       "-2",
				"QT_SCALE_FACTOR": "1.75",
			},
			want: 1.75,
		},
		{
			name:      "KDE removes Xft.dpi when reset to 96 DPI",
			xftDPI:    "",
			xsettings: xsettings1x,
			env:       map[string]string{},
			want:      1.0,
		},
		{
			name: "Nothing set defaults to 1.0",
			want: 1.0,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := resolveX11ContentScale(tc.xftDPI, tc.xsettings, envMap(tc.env))
			if math.Abs(float64(got-tc.want)) > 1e-6 {
				t.Errorf("resolveX11ContentScale() = %v, want %v", got, tc.want)
			}
		})
	}
}
