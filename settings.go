package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/plugins/terminal"

	"lex/internal/rex"
)

// settings are the user's choices, kept in settings.json beside the
// server's state. The settings panel (settingsview.go) changes them.
type settings struct {
	// Appearance is "light", "dark", or "system" to follow the system;
	// "dark" when empty.
	Appearance string  `json:"appearance,omitempty"`
	FontSize   float32 `json:"fontSize,omitempty"`
	// LineHeight is the height of the terminals' rows, as a multiple of
	// the font's line height: 1 when zero.
	LineHeight float32 `json:"lineHeight,omitempty"`
	// Cursor is the cursor's shape, "block", "bar" or "underline": a block
	// when empty. NoBlink keeps it from blinking.
	Cursor  string `json:"cursor,omitempty"`
	NoBlink bool   `json:"noBlink,omitempty"`
	// NoLigatures turns off the font's coding ligatures (-> != >=).
	NoLigatures bool `json:"noLigatures,omitempty"`
	// NoNotify keeps a long command that finishes while the window is in
	// the background from posting a notification.
	NoNotify bool `json:"noNotify,omitempty"`
}

const defaultFontSize = 11.6

var prefs settings

func settingsPath() string { return filepath.Join(rex.Dir(), "settings.json") }

func loadSettings() {
	if b, err := os.ReadFile(settingsPath()); err == nil {
		json.Unmarshal(b, &prefs)
	}
	if prefs.FontSize < 6 || prefs.FontSize > 40 {
		prefs.FontSize = defaultFontSize
	}
	applyFont()
	applyAppearance()
}

func saveSettings() {
	b, _ := json.MarshalIndent(prefs, "", "  ")
	os.MkdirAll(rex.Dir(), 0o700)
	os.WriteFile(settingsPath(), b, 0o600)
}

// appearance is the appearance chosen: dark unless the user chose
// another, as aurora, the theme of Lex's colors, is made dark first.
func appearance() string {
	switch prefs.Appearance {
	case "light", "system":
		return prefs.Appearance
	}
	return "dark"
}

func applyAppearance() {
	switch appearance() {
	case "light":
		mygo.Theme.SetSource(mygo.ThemeLight)
	case "system":
		mygo.Theme.SetSource(mygo.ThemeSystem)
	default:
		mygo.Theme.SetSource(mygo.ThemeDark)
	}
}

func (a *App) setAppearance(v string) {
	prefs.Appearance = v
	applyAppearance()
	saveSettings()
}

// lineHeight is the rows' height the settings ask for, within reason.
func (s settings) lineHeight() float32 {
	if s.LineHeight == 0 {
		return 1
	}
	return min(max(s.LineHeight, 0.8), 2)
}

// cursorStyle is the cursor's shape the settings ask for.
func (s settings) cursorStyle() terminal.CursorStyle {
	switch s.Cursor {
	case "bar":
		return terminal.CursorBar
	case "underline":
		return terminal.CursorUnderline
	}
	return terminal.CursorBlock
}

// cursorSequence sets the cursor's shape and blinking in a terminal that
// is already open, as a program does (DECSCUSR); new terminals take them
// from their options.
func (s settings) cursorSequence() []byte {
	n := 1 // a blinking block; the steady kind is the next number
	switch s.cursorStyle() {
	case terminal.CursorUnderline:
		n = 3
	case terminal.CursorBar:
		n = 5
	}
	if s.NoBlink {
		n++
	}
	return fmt.Appendf(nil, "\x1b[%d q", n)
}

// applyFont gives the terminals' font the settings' size, row height and
// ligatures.
func applyFont() {
	termFont.Size = prefs.FontSize
	termFont.LineHeight = prefs.lineHeight()
	termFont.Features = nil
	if prefs.NoLigatures {
		termFont.Features = []string{"-calt", "-liga"}
	}
}

// changeSettings changes the settings with fn, keeps them, and shows
// them in every pane at once.
func (a *App) changeSettings(fn func(*settings)) {
	fn(&prefs)
	prefs.FontSize = min(max(prefs.FontSize, 8), 32)
	applyFont()
	cursor := prefs.cursorSequence()
	for _, t := range a.tabs {
		for _, p := range t.panes() {
			if p.term != nil {
				p.term.SetFont(termFont)
				p.term.Feed(cursor)
			}
		}
	}
	saveSettings()
}

// setFontSize changes the size of the terminals' text, in every pane.
func (a *App) setFontSize(size float32) {
	a.changeSettings(func(s *settings) { s.FontSize = size })
}
