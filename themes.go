package main

import (
	"github.com/egoist/mygo/plugins/terminal"
	"github.com/egoist/mygo/ui"
)

// A theme is one of the looks the settings offer: the window's colors and
// the terminals' sixteen, from a handful of base colors. Each theme is
// either dark or light; the settings pick one, or one of each to follow
// the system's appearance.
type theme struct {
	id, name string
	dark     bool
	// mood says what the theme is like, under the settings' themes.
	mood string

	// bg is the window's, card the panes', text their text, muted and
	// faint the secondary and the quietest text.
	bg, card, text, muted, faint string
	// accent marks what is chosen and second is the color of hover
	// faces; onAccent is text on the accent, as under the cursor.
	accent, second, onAccent string
	// tile and tileFg face the host's glyph; switchOff is the track of a
	// switch that is off.
	tile, tileFg, switchOff string
	busy, attention, danger string
	// pal are the terminals' sixteen colors.
	pal [16]string
}

// themes are the looks, in the order the settings show them. Aurora is
// patina's theme (github.com/patina-os/patina, themes/aurora), true black
// with neon green as the accent and violet as the second color, and its
// light mode on lavender white; the others are Lex's own.
var themes = []*theme{
	{
		id: "aurora", name: "Aurora", dark: true,
		mood: "True black, neon green and violet. The default.",
		bg:   "#000000", card: "#0b0a10", text: "#d8d4e6", muted: "#aea8c4", faint: "#8a83a3",
		accent: "#3cff9d", second: "#b77dff", onAccent: "#000000",
		tile: "#17141f", tileFg: "#d8d4e6", switchOff: "#17141f",
		busy: "#3cff9d", attention: "#ffa45c", danger: "#ff5c7a",
		pal: [16]string{
			"#000000", "#ff5c7a", "#3cff9d", "#f2e15c", "#7aa2ff", "#b77dff", "#4de8d4", "#d8d4e6",
			"#524a6b", "#ff5c7a", "#3cff9d", "#f2e15c", "#7aa2ff", "#b77dff", "#4de8d4", "#f7f6fb",
		},
	},
	{
		id: "aurora-light", name: "Aurora Light", dark: false,
		mood: "Lavender white with deep green and violet.",
		bg:   "#eeecf5", card: "#f7f6fb", text: "#2a2638", muted: "#4b4560", faint: "#6b6585",
		accent: "#077141", second: "#7c3aed", onAccent: "#f7f6fb",
		tile: "#2a2638", tileFg: "#f7f6fb", switchOff: "#e0dcec",
		busy: "#077141", attention: "#b2590e", danger: "#d1264a",
		pal: [16]string{
			"#f7f6fb", "#d1264a", "#077141", "#8b6e00", "#3559d6", "#7c3aed", "#0e7f75", "#2a2638",
			"#928bab", "#d1264a", "#077141", "#8b6e00", "#3559d6", "#7c3aed", "#0e7f75", "#0d0b14",
		},
	},
	{
		id: "ember", name: "Ember", dark: true,
		mood: "Warm amber on charcoal, easy on the eyes at night.",
		bg:   "#0c0907", card: "#15100c", text: "#eadccb", muted: "#c4b39f", faint: "#93826f",
		accent: "#ffb347", second: "#ff7a59", onAccent: "#15100c",
		tile: "#241b14", tileFg: "#eadccb", switchOff: "#241b14",
		busy: "#a9cc5a", attention: "#ff8a3d", danger: "#f2685a",
		pal: [16]string{
			"#15100c", "#f2685a", "#a9cc5a", "#ffb347", "#6fb3d9", "#e08fb4", "#79cfb3", "#eadccb",
			"#7a6655", "#ff8576", "#c1e07a", "#ffc96e", "#93c8e6", "#f0a9c8", "#9be0c8", "#fff4e6",
		},
	},
	{
		id: "tidepool", name: "Tidepool", dark: true,
		mood: "Aqua on deep ink, crisp and cool.",
		bg:   "#04080c", card: "#0a1219", text: "#d0e3ee", muted: "#a3bccc", faint: "#7590a3",
		accent: "#45d4e6", second: "#8c9cff", onAccent: "#04080c",
		tile: "#13202b", tileFg: "#d0e3ee", switchOff: "#13202b",
		busy: "#5ee0a0", attention: "#ffb86b", danger: "#ff6f91",
		pal: [16]string{
			"#0a1219", "#ff6f91", "#5ee0a0", "#f2d572", "#6ea6ff", "#b49cff", "#45d4e6", "#d0e3ee",
			"#5d7a8f", "#ff8fa9", "#86ecb8", "#f8e396", "#94bfff", "#cbb8ff", "#7ae3f0", "#f2fbff",
		},
	},
	{
		id: "orchid", name: "Orchid", dark: true,
		mood: "Soft rose and lilac, low glare for long sessions.",
		bg:   "#0f0b13", card: "#17111d", text: "#e9dfef", muted: "#c3b4cf", faint: "#978aa5",
		accent: "#ff9cc6", second: "#b8a3ff", onAccent: "#17111d",
		tile: "#241a2c", tileFg: "#e9dfef", switchOff: "#241a2c",
		busy: "#9de0b0", attention: "#ffc08a", danger: "#ff7088",
		pal: [16]string{
			"#17111d", "#ff7a93", "#9de0b0", "#f5d68f", "#8fb0ff", "#c9a8ff", "#8fe0dc", "#e9dfef",
			"#776486", "#ff9aae", "#b8ecc6", "#fae3ad", "#adc5ff", "#dcc3ff", "#ade9e6", "#fff7fd",
		},
	},
	{
		id: "paper", name: "Paper", dark: false,
		mood: "Ink on warm paper, for daylight and bright rooms.",
		bg:   "#e9e2d4", card: "#f6f1e6", text: "#2a251f", muted: "#4a4339", faint: "#6b6253",
		accent: "#a3460f", second: "#2f5d9e", onAccent: "#f6f1e6",
		tile: "#2a251f", tileFg: "#f6f1e6", switchOff: "#ddd5c5",
		busy: "#3f7a1d", attention: "#b25e00", danger: "#b3261e",
		pal: [16]string{
			"#e9e2d4", "#b3261e", "#3f7a1d", "#8a6400", "#2f5d9e", "#8b3f8b", "#17717a", "#2a251f",
			"#8f8574", "#c63a2f", "#4f8f26", "#a07600", "#3f70b5", "#a050a0", "#1f8590", "#14110d",
		},
	},
}

const (
	defaultDarkTheme  = "aurora"
	defaultLightTheme = "aurora-light"
)

// themeByID returns the theme of an id, or nil.
func themeByID(id string) *theme {
	for _, t := range themes {
		if t.id == id {
			return t
		}
	}
	return nil
}

// colors derives the window's colors from the theme's base colors, as
// aurora's are: rims and separators are its text at 12 %, the focused
// rim at 28 %, the selection the accent at 10 % with an edge four times
// as strong, and hover faces the second color.
func (t *theme) colors() *colors {
	text, second, accent := ui.Hex(t.text), ui.Hex(t.second), ui.Hex(t.accent)
	k := &colors{
		bg:                ui.Hex(t.bg),
		card:              ui.Hex(t.card),
		cardBorder:        text.Alpha(0.12),
		cardBorderFocused: text.Alpha(0.28),
		text:              text,
		textMuted:         ui.Hex(t.muted),
		textFaint:         ui.Hex(t.faint),
		iconMuted:         ui.Hex(t.faint),
		accent:            accent,
		onAccent:          ui.Hex(t.onAccent),
		selFill:           accent.Alpha(0.10),
		selEdge:           accent.Alpha(0.40),
		sep:               text.Alpha(0.12),
		tile:              ui.Hex(t.tile),
		tileFg:            ui.Hex(t.tileFg),
		switchOff:         ui.Hex(t.switchOff),
		busy:              ui.Hex(t.busy),
		attention:         ui.Hex(t.attention),
		danger:            ui.Hex(t.danger),
		blue:              ui.Hex(t.pal[4]),
		magenta:           ui.Hex(t.pal[5]),
		panel:             ui.Hex(t.card),
		panelBorder:       text.Alpha(0.12),
	}
	if t.dark {
		k.hover, k.pressed = second.Alpha(0.12), second.Alpha(0.22)
		k.tileRim = ui.RGBA(255, 255, 255, 0.22)
		k.shadow, k.backdrop = ui.RGBA(0, 0, 0, 0.6), ui.RGBA(0, 0, 0, 0.45)
	} else {
		k.hover, k.pressed = second.Alpha(0.08), second.Alpha(0.16)
		k.tileRim = ui.RGBA(255, 255, 255, 0.95)
		k.shadow, k.backdrop = ui.RGBA(13, 11, 20, 0.14), ui.RGBA(13, 11, 20, 0.12)
	}
	return k
}

// terminal derives the terminals' colors: the sixteen, the text and the
// pane's face, and the cursor and the selection in the accent, with the
// text under them in onAccent.
func (t *theme) terminal() *terminal.Theme {
	th := &terminal.Theme{
		Foreground:    ui.Hex(t.text),
		Background:    ui.Hex(t.card),
		Cursor:        ui.Hex(t.accent),
		CursorText:    ui.Hex(t.onAccent),
		Selection:     ui.Hex(t.accent),
		SelectionText: ui.Hex(t.onAccent),
	}
	for i, c := range t.pal {
		th.Palette[i] = ui.Hex(c)
	}
	return th
}

// themeColors and themeTerms keep each theme's derived colors.
var (
	themeColors = map[string]*colors{}
	themeTerms  = map[string]*terminal.Theme{}
)

func init() {
	for _, t := range themes {
		themeColors[t.id] = t.colors()
		themeTerms[t.id] = t.terminal()
	}
}
