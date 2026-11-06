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
	// Theme is the id of the theme the window shows (themes.go): Aurora
	// when empty. With FollowSystem, the window follows the system's
	// appearance instead, in DarkTheme while it is dark and LightTheme
	// while it is light (Aurora and Aurora Light when empty).
	Theme        string `json:"theme,omitempty"`
	FollowSystem bool   `json:"followSystem,omitempty"`
	DarkTheme    string `json:"darkTheme,omitempty"`
	LightTheme   string `json:"lightTheme,omitempty"`
	// Appearance is what earlier versions kept: "light", "dark", or
	// "system". loadSettings turns it into the theme and drops it.
	Appearance string `json:"appearance,omitempty"`

	// Background is the path of an image the window shows behind its
	// panes, none when empty. BackgroundFit is how it fills the window:
	// "fill" (cropped to cover it; the default), "fit" (whole, inside
	// it), "tile" or "center" (at its own size). PaneOpacity is how
	// opaque the panes are over it, from 0.3 to 1 (0.85 when zero);
	// BackgroundDim darkens it, or lightens it in a light theme, from 0
	// to 0.8; BackgroundBlur blurs it by that many DIPs, up to 24.
	Background     string  `json:"background,omitempty"`
	BackgroundFit  string  `json:"backgroundFit,omitempty"`
	PaneOpacity    float32 `json:"paneOpacity,omitempty"`
	BackgroundDim  float32 `json:"backgroundDim,omitempty"`
	BackgroundBlur float32 `json:"backgroundBlur,omitempty"`

	// Font is the family of the terminals' text, one of the system's:
	// JetBrains Mono, which Lex embeds, when empty. UIFont is the family
	// of the rest of the window's text: the system's when empty.
	Font     string  `json:"font,omitempty"`
	UIFont   string  `json:"uiFont,omitempty"`
	FontSize float32 `json:"fontSize,omitempty"`
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

const (
	defaultFontSize    = 11.6
	defaultPaneOpacity = 0.85
)

var prefs settings

func settingsPath() string { return filepath.Join(rex.Dir(), "settings.json") }

func loadSettings() {
	if b, err := os.ReadFile(settingsPath()); err == nil {
		json.Unmarshal(b, &prefs)
	}
	if prefs.FontSize < 6 || prefs.FontSize > 40 {
		prefs.FontSize = defaultFontSize
	}
	prefs.migrate()
	applyFont()
	applyAppearance()
}

// migrate turns the appearance of earlier versions into a theme: light
// into Aurora Light, and the system's into following it with Aurora and
// Aurora Light.
func (s *settings) migrate() {
	if s.Appearance == "" {
		return
	}
	if s.Theme == "" && !s.FollowSystem {
		switch s.Appearance {
		case "light":
			s.Theme = defaultLightTheme
		case "system":
			s.FollowSystem = true
		}
	}
	s.Appearance = ""
}

func saveSettings() {
	b, _ := json.MarshalIndent(prefs, "", "  ")
	os.MkdirAll(rex.Dir(), 0o700)
	os.WriteFile(settingsPath(), b, 0o600)
}

// themeFor returns the theme the window shows while the system's
// appearance is dark, or light: the one theme chosen, unless the
// settings follow the system.
func (s settings) themeFor(dark bool) *theme {
	if !s.FollowSystem {
		if t := themeByID(s.Theme); t != nil {
			return t
		}
		return themeByID(defaultDarkTheme)
	}
	id, def := s.LightTheme, defaultLightTheme
	if dark {
		id, def = s.DarkTheme, defaultDarkTheme
	}
	if t := themeByID(id); t != nil && t.dark == dark {
		return t
	}
	return themeByID(def)
}

// applyAppearance makes the window, its native parts and its terminals
// as dark or as light as the theme, or follow the system.
func applyAppearance() {
	switch {
	case prefs.FollowSystem:
		setThemeSource(mygo.ThemeSystem)
	case prefs.themeFor(true).dark:
		setThemeSource(mygo.ThemeDark)
	default:
		setThemeSource(mygo.ThemeLight)
	}
}

// setThemeSource makes the app light, dark or the system's, when it is
// not already: every window and its native parts take it. Tests, which
// run no app, replace it.
var setThemeSource = func(s mygo.ThemeSource) {
	if mygo.Theme.Source() != s {
		mygo.Theme.SetSource(s)
	}
}

// termThemes are the terminals' colors in light and in dark windows.
func termThemes() (light, dark *terminal.Theme) {
	return themeTerms[prefs.themeFor(false).id], themeTerms[prefs.themeFor(true).id]
}

// paneOpacity is how opaque the panes are over the background image.
func (s settings) paneOpacity() float32 {
	if s.PaneOpacity == 0 {
		return defaultPaneOpacity
	}
	return min(max(s.PaneOpacity, 0.3), 1)
}

// termFamily is the family of the terminals' text, with the embedded
// font and the system's monospaced one after it for what it lacks.
func (s settings) termFamily() string {
	if s.Font == "" || s.Font == builtinFont {
		return builtinFont + ", monospace"
	}
	return s.Font + ", " + builtinFont + ", monospace"
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
	termFont.Family = prefs.termFamily()
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
	applyAppearance()
	cursor := prefs.cursorSequence()
	light, dark := termThemes()
	for _, t := range a.tabs {
		for _, p := range t.panes() {
			if p.term != nil {
				p.term.SetFont(termFont)
				p.term.SetTheme(light, dark)
				p.term.Feed(cursor)
			}
		}
	}
	saveSettings()
}

// pickTheme shows a theme: the one theme, or, following the system, the
// one of its side, dark or light.
func (a *App) pickTheme(t *theme) {
	a.changeSettings(func(s *settings) {
		switch {
		case !s.FollowSystem:
			s.Theme = t.id
		case t.dark:
			s.DarkTheme = t.id
		default:
			s.LightTheme = t.id
		}
	})
}

// toggleFollowSystem starts or stops following the system's appearance,
// dark now or not, keeping the theme shown: it becomes the theme of its
// side, or the one theme.
func (a *App) toggleFollowSystem(dark bool) {
	a.changeSettings(func(s *settings) {
		shown := s.themeFor(dark)
		s.FollowSystem = !s.FollowSystem
		switch {
		case !s.FollowSystem:
			s.Theme = shown.id
		case shown.dark:
			s.DarkTheme = shown.id
		default:
			s.LightTheme = shown.id
		}
	})
}

// changeLook changes settings only the window's drawing reads, as the
// background's, and keeps them: the terminals need not hear of them.
func (a *App) changeLook(fn func(*settings)) {
	fn(&prefs)
	saveSettings()
}

// setFontSize changes the size of the terminals' text, in every pane.
func (a *App) setFontSize(size float32) {
	a.changeSettings(func(s *settings) { s.FontSize = size })
}
