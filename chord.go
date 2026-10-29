// Copyright 2026 The ggip Authors
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
	"errors"
	"maps"
	"strings"
	"sync"
)

// Chord is a key with modifiers, as a shortcut names it: "cmd+s",
// "ctrl+shift+z", "escape". Parse one with ParseChord.
type Chord struct {
	Key  Key
	Mods Mods
	// Cmd stands for the platform's command modifier: Meta on macOS, Ctrl
	// elsewhere. A chord written with "cmd" matches through Mods.Cmd.
	Cmd bool
}

// Modified reports whether the chord holds a modifier other than Shift,
// which is what makes it run before the focused widget.
func (c Chord) Modified() bool { return c.Cmd || c.Mods.Ctrl || c.Mods.Alt || c.Mods.Meta }

// String writes the chord back the way ParseChord reads it.
func (c Chord) String() string {
	var parts []string
	if c.Cmd {
		parts = append(parts, "cmd")
	}
	if c.Mods.Ctrl {
		parts = append(parts, "ctrl")
	}
	if c.Mods.Alt {
		parts = append(parts, "alt")
	}
	if c.Mods.Meta {
		parts = append(parts, "meta")
	}
	if c.Mods.Shift {
		parts = append(parts, "shift")
	}
	return strings.Join(append(parts, strings.ToLower(c.Key.String())), "+")
}

// Matches reports whether a press of key with mods held is c: the same key
// with the same modifiers, no more and no fewer.
func (c Chord) Matches(key Key, mods Mods) bool {
	return key == c.Key && mods == c.Held()
}

// Held is the modifiers a press of c holds on this platform, cmd turned
// into Meta on macOS and Ctrl elsewhere.
func (c Chord) Held() Mods {
	m := c.Mods
	if c.Cmd {
		if isDarwin {
			m.Meta = true
		} else {
			m.Ctrl = true
		}
	}
	return m
}

// ParseChord reads a chord such as "cmd+s", "ctrl+shift+z", "alt+enter"
// or "f1". Modifiers are cmd (the platform's command key), ctrl, alt (or
// option), shift and meta; the key is any key name, ignoring
// case, with esc, return, up, down, left, right, plus and minus accepted,
// and a digit or punctuation key also by the character it types on a US
// layout: "cmd+1" is "cmd+digit1", "cmd+," is "cmd+comma".
func ParseChord(s string) (Chord, error) {
	var c Chord
	s = strings.TrimSpace(s)
	parts := strings.Split(strings.ToLower(s), "+")
	if s == "" {
		return c, errors.New("ggwd: empty chord")
	}
	// "ctrl++" names the plus key.
	if strings.HasSuffix(s, "++") {
		parts = append(parts[:len(parts)-2], "plus")
	}
	for i, p := range parts {
		last := i == len(parts)-1
		switch p {
		case "cmd", "command", "super":
			c.Cmd = true
		case "ctrl", "control":
			c.Mods.Ctrl = true
		case "alt", "option", "opt":
			c.Mods.Alt = true
		case "shift":
			c.Mods.Shift = true
		case "meta", "win":
			c.Mods.Meta = true
		default:
			k, ok := keyNames()[p]
			if !ok || !last {
				return c, errors.New("ggwd: unknown key " + s)
			}
			c.Key = k
		}
		if last && c.Key == 0 && p != "a" {
			return c, errors.New("ggwd: chord " + s + " names no key")
		}
	}
	return c, nil
}

// MustChord is ParseChord for a chord written in the source; it panics on
// an error.
func MustChord(s string) Chord {
	c, err := ParseChord(s)
	if err != nil {
		panic(err)
	}
	return c
}

// keyNames maps every key's lower-cased name and a few aliases
// to the key, built on first use, from whichever goroutine parses first.
var keyNames = sync.OnceValue(func() map[string]Key {
	table := map[string]Key{}
	for k := Key(0); k <= KeyMax; k++ {
		if name := k.String(); name != "" {
			table[strings.ToLower(name)] = k
		}
	}
	maps.Copy(table, map[string]Key{
		"esc": KeyEscape, "return": KeyEnter, "up": KeyArrowUp,
		"down": KeyArrowDown, "left": KeyArrowLeft, "right": KeyArrowRight,
		"plus": KeyEqual, "minus": KeyMinus, "del": KeyDelete,
		"pgup": KeyPageUp, "pgdn": KeyPageDown, "bksp": KeyBackspace,
		"-": KeyMinus, "=": KeyEqual, ",": KeyComma, ".": KeyPeriod,
		"/": KeySlash, ";": KeySemicolon, "'": KeyQuote, "[": KeyBracketLeft,
		"]": KeyBracketRight, "\\": KeyBackslash, "`": KeyBackquote,
	})
	for k := KeyDigit0; k <= KeyDigit9; k++ {
		table[string(rune('0'+(k-KeyDigit0)))] = k
	}
	return table
})

// Label writes the chord the way the platform shows it to a user, in a
// menu or a tooltip: "⇧⌘S" on macOS, "Ctrl+Shift+S" elsewhere.
func (c Chord) Label() string {
	ctrl, alt, shift, meta := c.Mods.Ctrl, c.Mods.Alt, c.Mods.Shift, c.Mods.Meta
	mac := isDarwin
	if c.Cmd {
		if mac {
			meta = true
		} else {
			ctrl = true
		}
	}
	key := keyLabel(c.Key, mac)
	if mac {
		var b strings.Builder
		for _, m := range []struct {
			on  bool
			sym string
		}{{ctrl, "⌃"}, {alt, "⌥"}, {shift, "⇧"}, {meta, "⌘"}} {
			if m.on {
				b.WriteString(m.sym)
			}
		}
		return b.String() + key
	}
	var parts []string
	for _, m := range []struct {
		on   bool
		name string
	}{{ctrl, "Ctrl"}, {alt, "Alt"}, {shift, "Shift"}, {meta, "Win"}} {
		if m.on {
			parts = append(parts, m.name)
		}
	}
	return strings.Join(append(parts, key), "+")
}

// keyLabel is how a key is written in a chord's label.
func keyLabel(k Key, mac bool) string {
	switch {
	case k >= KeyA && k <= KeyZ:
		return string(rune('A' + (k - KeyA)))
	case k >= KeyDigit0 && k <= KeyDigit9:
		return string(rune('0' + (k - KeyDigit0)))
	}
	symbols := map[Key][2]string{
		KeyEnter: {"Enter", "↩"}, KeyEscape: {"Esc", "⎋"}, KeyTab: {"Tab", "⇥"},
		KeyBackspace: {"Backspace", "⌫"}, KeyDelete: {"Del", "⌦"}, KeySpace: {"Space", "Space"},
		KeyArrowUp: {"Up", "↑"}, KeyArrowDown: {"Down", "↓"}, KeyArrowLeft: {"Left", "←"}, KeyArrowRight: {"Right", "→"},
		KeyPageUp: {"PgUp", "⇞"}, KeyPageDown: {"PgDn", "⇟"}, KeyHome: {"Home", "↖"}, KeyEnd: {"End", "↘"},
		KeyComma: {",", ","}, KeyPeriod: {".", "."}, KeySlash: {"/", "/"}, KeySemicolon: {";", ";"},
		KeyQuote: {"'", "'"}, KeyBracketLeft: {"[", "["}, KeyBracketRight: {"]", "]"}, KeyBackslash: {"\\", "\\"},
		KeyMinus: {"-", "-"}, KeyEqual: {"=", "="}, KeyBackquote: {"`", "`"},
	}
	if s, ok := symbols[k]; ok {
		if mac {
			return s[1]
		}
		return s[0]
	}
	return k.String()
}
