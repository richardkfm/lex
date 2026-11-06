package main

import (
	_ "embed"

	"github.com/egoist/mygo/plugins/terminal"
	"github.com/egoist/mygo/ui"
)

// colors are the window's, derived from a theme (themes.go): the comments
// say what each colors.
type colors struct {
	// bg is the window's, flat: the title bar and the gaps between panes.
	bg ui.Color

	// The panes: a flat card with a hairline rim, a brighter one round the
	// focused pane.
	card, cardBorder, cardBorderFocused ui.Color

	text, textMuted, textFaint, iconMuted ui.Color
	// hover and pressed are faces of the second color under the pointer.
	hover, pressed ui.Color

	// accent marks what is chosen: the active tab, the selected row, the
	// focused pane, the front tile. selFill and selEdge are a faint fill
	// of it and its thin edge; onAccent is text on it.
	accent, onAccent, selFill, selEdge ui.Color
	sep                                ui.Color
	// tile and tileFg face the host's glyph in its popover; tileRim rims
	// the program tiles of tabs.
	tile, tileFg, tileRim ui.Color
	// switchOff is the track of a switch that is off.
	switchOff ui.Color

	busy, attention, danger ui.Color
	// blue and magenta are its terminals': with danger and attention,
	// the colors of pane tints.
	blue, magenta ui.Color

	panel, panelBorder, shadow, backdrop ui.Color
}

// colorsOf returns the colors of the theme the window shows: the one the
// settings chose, or, following the system, its dark or its light one.
func colorsOf(c *ui.Context) *colors {
	return themeColors[prefs.themeFor(c.Theme().Dark).id]
}

//go:embed assets/fonts/JetBrainsMono-Regular.ttf
var fontRegular []byte

//go:embed assets/fonts/JetBrainsMono-Bold.ttf
var fontBold []byte

//go:embed assets/fonts/JetBrainsMono-Italic.ttf
var fontItalic []byte

//go:embed assets/fonts/JetBrainsMono-BoldItalic.ttf
var fontBoldItalic []byte

func registerFonts() {
	for _, f := range [][]byte{fontRegular, fontBold, fontItalic, fontBoldItalic} {
		ui.RegisterFont(f, "JetBrains Mono")
	}
}

// builtinFont is the font Lex embeds, the terminals' unless the settings
// choose another of the system's.
const builtinFont = "JetBrains Mono"

var termFont = terminal.Font{Family: builtinFont + ", monospace", Size: 11.6, LineHeight: 1.0}
