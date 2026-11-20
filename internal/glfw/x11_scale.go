// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2026 The ggfx Authors

//go:build darwin || freebsd || linux || netbsd || windows

package glfw

import (
	"encoding/binary"
	"math"
	"strconv"
	"strings"
)

// parsePositiveScale parses a positive finite floating-point scale factor.
func parsePositiveScale(raw string) (float32, bool) {
	s := strings.TrimSpace(raw)
	if s == "" {
		return 0, false
	}
	v, err := strconv.ParseFloat(s, 64)
	if err != nil || math.IsNaN(v) || math.IsInf(v, 0) || v <= 0 {
		return 0, false
	}
	scale := float32(v)
	if math.IsInf(float64(scale), 0) || scale <= 0 {
		return 0, false
	}
	return scale, true
}

// parsePositiveDPI parses a positive finite X11 DPI value and converts it to a
// content scale relative to the standard 96 DPI baseline.
func parsePositiveDPI(raw string) (float32, bool) {
	s := strings.TrimSpace(raw)
	if s == "" {
		return 0, false
	}
	dpi, err := strconv.ParseFloat(s, 64)
	if err != nil || math.IsNaN(dpi) || math.IsInf(dpi, 0) || dpi <= 0 {
		return 0, false
	}
	scale := float32(dpi / 96.0)
	if math.IsInf(float64(scale), 0) || scale <= 0 {
		return 0, false
	}
	return scale, true
}

// parseXSettingsScale parses a freedesktop _XSETTINGS_SETTINGS binary property
// payload and extracts the display scale from Xft/DPI (stored as DPI * 1024)
// and/or Gdk/WindowScalingFactor.
func parseXSettingsScale(data []byte) (float32, bool) {
	// Header: byte-order (1), unused (3), serial (4), n_settings (4).
	if len(data) < 12 {
		return 0, false
	}

	var bo binary.ByteOrder
	switch data[0] {
	case 'l':
		bo = binary.LittleEndian
	case 'B':
		bo = binary.BigEndian
	default:
		return 0, false
	}

	nSettings := bo.Uint32(data[8:12])
	offset := 12

	var (
		xftScale float32
		hasXft   bool
		gdkScale float32
		hasGdk   bool
	)

	const (
		xsettingsTypeInteger = 0
		xsettingsTypeString  = 1
		xsettingsTypeColor   = 2
	)

	for range nSettings {
		if offset > len(data)-4 {
			break
		}
		typ := data[offset]
		nameLen := int(bo.Uint16(data[offset+2 : offset+4]))
		offset += 4

		paddedNameLen := (nameLen + 3) &^ 3
		if paddedNameLen > len(data)-offset {
			break
		}
		name := string(data[offset : offset+nameLen])
		offset += paddedNameLen

		// last-change-serial (CARD32).
		if 4 > len(data)-offset {
			break
		}
		offset += 4

		switch typ {
		case xsettingsTypeInteger:
			if 4 > len(data)-offset {
				return combineXSettingsScale(xftScale, hasXft, gdkScale, hasGdk)
			}
			val := int32(bo.Uint32(data[offset : offset+4]))
			offset += 4
			switch name {
			case "Xft/DPI":
				if val > 0 {
					xftScale = float32(float64(val) / (1024.0 * 96.0))
					hasXft = xftScale > 0
				}
			case "Gdk/WindowScalingFactor":
				if val > 0 {
					gdkScale = float32(val)
					hasGdk = gdkScale > 0
				}
			}

		case xsettingsTypeString:
			if 4 > len(data)-offset {
				return combineXSettingsScale(xftScale, hasXft, gdkScale, hasGdk)
			}
			strLen := bo.Uint32(data[offset : offset+4])
			offset += 4
			rem := uint64(len(data) - offset)
			if uint64(strLen) > rem {
				return combineXSettingsScale(xftScale, hasXft, gdkScale, hasGdk)
			}
			paddedStrLen := (uint64(strLen) + 3) &^ 3
			if paddedStrLen > rem {
				return combineXSettingsScale(xftScale, hasXft, gdkScale, hasGdk)
			}
			offset += int(paddedStrLen)

		case xsettingsTypeColor:
			if 8 > len(data)-offset {
				return combineXSettingsScale(xftScale, hasXft, gdkScale, hasGdk)
			}
			offset += 8

		default:
			return combineXSettingsScale(xftScale, hasXft, gdkScale, hasGdk)
		}
	}

	return combineXSettingsScale(xftScale, hasXft, gdkScale, hasGdk)
}

func combineXSettingsScale(xftScale float32, hasXft bool, gdkScale float32, hasGdk bool) (float32, bool) {
	if hasXft && xftScale > 0 {
		// Some desktops (e.g. XFCE) set Gdk/WindowScalingFactor=2 while leaving
		// Xft/DPI at 96*1024; prefer the integer window scale in that case.
		if hasGdk && gdkScale > 1 && xftScale <= 1 {
			return gdkScale, true
		}
		return xftScale, true
	}
	if hasGdk && gdkScale > 0 {
		return gdkScale, true
	}
	return 0, false
}

// resolveX11ContentScale chooses the system content scale from X11 sources:
//  1. Xft.dpi from RESOURCE_MANAGER (when set and valid)
//  2. XSETTINGS _XSETTINGS_SETTINGS (Xft/DPI or Gdk/WindowScalingFactor) when non-1
//  3. GDK_SCALE / QT_SCALE_FACTOR environment variables
//  4. XSETTINGS scale (if 1.0) or default 1.0 (96 DPI)
func resolveX11ContentScale(xftDPI string, xsettings []byte, getenv func(string) string) float32 {
	if scale, ok := parsePositiveDPI(xftDPI); ok {
		return scale
	}

	xsettingsScale, hasXSettings := parseXSettingsScale(xsettings)
	if hasXSettings && xsettingsScale != 1 {
		return xsettingsScale
	}

	if getenv != nil {
		if scale, ok := parsePositiveScale(getenv("GDK_SCALE")); ok {
			return scale
		}
		if scale, ok := parsePositiveScale(getenv("QT_SCALE_FACTOR")); ok {
			return scale
		}
	}

	if hasXSettings {
		return xsettingsScale
	}

	return 1
}
