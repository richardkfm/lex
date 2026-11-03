package main

import (
	_ "embed"

	"github.com/egoist/mygo/plugins/terminal"
	"github.com/egoist/mygo/ui"
)

// colors are the app's own, in light and dark windows: patina's aurora
// theme (github.com/patina-os/patina, themes/aurora), true black with neon
// green as the accent and violet as the second color, and its light mode
// on lavender white. The comments name aurora's base16 slots.
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
	// of it and its thin edge.
	accent, selFill, selEdge ui.Color
	sep                      ui.Color
	// tile and tileFg face the host's glyph in its popover; tileRim rims
	// the program tiles of tabs.
	tile, tileFg, tileRim ui.Color
	// switchOff is the track of a switch that is off.
	switchOff ui.Color

	busy, attention, danger ui.Color

	panel, panelBorder, shadow, backdrop ui.Color
}

var darkColors = colors{
	bg: ui.Hex("#000000"), // base00: an OLED panel's pixels stay off

	card:              ui.Hex("#0b0a10"),            // base01
	cardBorder:        ui.RGBA(216, 212, 230, 0.12), // base05 at 12 %, patina's rim
	cardBorderFocused: ui.RGBA(216, 212, 230, 0.28), // base05 at 28 %

	text:      ui.Hex("#d8d4e6"), // base05
	textMuted: ui.Hex("#aea8c4"), // between base04 and base05
	textFaint: ui.Hex("#8a83a3"), // base04
	iconMuted: ui.Hex("#8a83a3"),
	hover:     ui.RGBA(183, 125, 255, 0.12), // base0E
	pressed:   ui.RGBA(183, 125, 255, 0.22),

	accent:  ui.Hex("#3cff9d"),           // base0B
	selFill: ui.RGBA(60, 255, 157, 0.10), // patina's selection: 10 %
	selEdge: ui.RGBA(60, 255, 157, 0.40), // and four times as much on its edge
	sep:     ui.RGBA(216, 212, 230, 0.12),

	tile:    ui.Hex("#17141f"), // base02
	tileFg:  ui.Hex("#d8d4e6"),
	tileRim: ui.RGBA(255, 255, 255, 0.22),

	switchOff: ui.Hex("#17141f"), // base02

	busy:      ui.Hex("#3cff9d"), // base0B
	attention: ui.Hex("#ffa45c"), // base09, kitty's bell border
	danger:    ui.Hex("#ff5c7a"), // base08

	panel:       ui.Hex("#0b0a10"),
	panelBorder: ui.RGBA(216, 212, 230, 0.12),
	shadow:      ui.RGBA(0, 0, 0, 0.6),
	backdrop:    ui.RGBA(0, 0, 0, 0.45),
}

var lightColors = colors{
	bg: ui.Hex("#eeecf5"), // base01: panels and title bars

	card:              ui.Hex("#f7f6fb"),         // base00: lavender white
	cardBorder:        ui.RGBA(42, 38, 56, 0.12), // base05 at 12 %
	cardBorderFocused: ui.RGBA(42, 38, 56, 0.28), // base05 at 28 %

	text:      ui.Hex("#2a2638"), // base05
	textMuted: ui.Hex("#4b4560"), // between base04 and base05
	textFaint: ui.Hex("#6b6585"), // base04
	iconMuted: ui.Hex("#6b6585"),
	hover:     ui.RGBA(124, 58, 237, 0.08), // base0E
	pressed:   ui.RGBA(124, 58, 237, 0.16),

	accent:  ui.Hex("#077141"), // base0B
	selFill: ui.RGBA(7, 113, 65, 0.10),
	selEdge: ui.RGBA(7, 113, 65, 0.40),
	sep:     ui.RGBA(42, 38, 56, 0.12),

	tile:    ui.Hex("#2a2638"), // base05: dark tiles, as aurora's app glyphs
	tileFg:  ui.Hex("#f7f6fb"),
	tileRim: ui.RGBA(255, 255, 255, 0.95),

	switchOff: ui.Hex("#e0dcec"), // base02

	busy:      ui.Hex("#077141"), // base0B
	attention: ui.Hex("#b2590e"), // base09
	danger:    ui.Hex("#d1264a"), // base08

	panel:       ui.Hex("#f7f6fb"),
	panelBorder: ui.RGBA(42, 38, 56, 0.12),
	shadow:      ui.RGBA(13, 11, 20, 0.14),
	backdrop:    ui.RGBA(13, 11, 20, 0.12),
}

func colorsOf(c *ui.Context) *colors {
	if c.Theme().Dark {
		return &darkColors
	}
	return &lightColors
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

var termFont = terminal.Font{Family: "JetBrains Mono, monospace", Size: 11.6, LineHeight: 1.0}

// The terminals' colors: aurora's, as patina gives them to kitty. The
// sixteen colors are the base16 slots, the cursor and the selection the
// accent, the selected text the background.
var darkTerm = &terminal.Theme{
	Foreground:    ui.Hex("#d8d4e6"),
	Background:    ui.Hex("#0b0a10"),
	Cursor:        ui.Hex("#3cff9d"),
	CursorText:    ui.Hex("#000000"),
	Selection:     ui.Hex("#3cff9d"),
	SelectionText: ui.Hex("#000000"),
	Palette: [16]ui.Color{
		ui.Hex("#000000"), ui.Hex("#ff5c7a"), ui.Hex("#3cff9d"), ui.Hex("#f2e15c"),
		ui.Hex("#7aa2ff"), ui.Hex("#b77dff"), ui.Hex("#4de8d4"), ui.Hex("#d8d4e6"),
		ui.Hex("#524a6b"), ui.Hex("#ff5c7a"), ui.Hex("#3cff9d"), ui.Hex("#f2e15c"),
		ui.Hex("#7aa2ff"), ui.Hex("#b77dff"), ui.Hex("#4de8d4"), ui.Hex("#f7f6fb"),
	},
}

var lightTerm = &terminal.Theme{
	Foreground:    ui.Hex("#2a2638"),
	Background:    ui.Hex("#f7f6fb"),
	Cursor:        ui.Hex("#077141"),
	CursorText:    ui.Hex("#f7f6fb"),
	Selection:     ui.Hex("#077141"),
	SelectionText: ui.Hex("#f7f6fb"),
	Palette: [16]ui.Color{
		ui.Hex("#f7f6fb"), ui.Hex("#d1264a"), ui.Hex("#077141"), ui.Hex("#8b6e00"),
		ui.Hex("#3559d6"), ui.Hex("#7c3aed"), ui.Hex("#0e7f75"), ui.Hex("#2a2638"),
		ui.Hex("#928bab"), ui.Hex("#d1264a"), ui.Hex("#077141"), ui.Hex("#8b6e00"),
		ui.Hex("#3559d6"), ui.Hex("#7c3aed"), ui.Hex("#0e7f75"), ui.Hex("#0d0b14"),
	},
}
