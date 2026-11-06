package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/egoist/mygo/ui"
)

// settingsButton draws the gear of the title bar, and below it, while it
// is open, the settings panel. Every change shows at once in every pane
// and is kept in settings.json.
func (a *App) settingsButton(c *ui.Context, k *colors) {
	gear := iconButton(c, k, "configure", "Settings", 30, 16).Tooltip(tip("Settings", &cmdSettings))
	if a.settingsOpen {
		gear.Background(k.hover)
	}
	if gear.Clicked() {
		a.settingsOpen = !a.settingsOpen
	}
	was := a.settingsOpen
	_, winH := c.Size()
	ui.PopoverBase(c, gear, &a.settingsOpen, func(panel ui.Element) {
		panel.AttachTo(gear, ui.AnchorBottomRight, ui.AnchorTopRight).Margin(8, 0, 0, 0).
			Width(360).Radius(8).Background(k.panel).Border(1, k.panelBorder).Clip().
			Shadow(0, 12, 32, 0, k.shadow).Role(ui.RoleDialog).Label("Settings")
		// The panel scrolls in a window too low for all of it.
		ui.Scroll(c).MaxHeight(max(winH-titleH-24, 200)).Padding(14).Gap(14).Children(func() {
			a.settingsPanel(c, k)
		})
	})
	// The focus goes back to the pane the panel took it from.
	if was && !a.settingsOpen {
		a.fontOpen = ""
		if t := a.tab(); t != nil {
			a.focusReq = t.Focus
		}
	}
}

// lineSpacings are the rows' heights the panel offers.
var lineSpacings = []struct {
	label  string
	height float32
}{{"Compact", 1}, {"Normal", 1.1}, {"Relaxed", 1.2}}

// settingsTabs are the panel's sections.
var settingsTabs = []struct{ id, label string }{{"look", "Look"}, {"text", "Text"}, {"general", "General"}}

func (a *App) settingsPanel(c *ui.Context, k *colors) {
	if a.settingsTab == "" {
		a.settingsTab = "look"
	}
	ui.Row(c).Gap(8).AlignItems(ui.Center).Children(func() {
		ui.Text(c, "Settings").FontSize(14).FontWeight(700).TextColor(k.text).Grow(1)
		at := 0
		labels := make([]string, len(settingsTabs))
		for i, s := range settingsTabs {
			labels[i] = s.label
			if s.id == a.settingsTab {
				at = i
			}
		}
		if i, ok := tabRow(c, k, "Sections", labels, at); ok {
			a.settingsTab = settingsTabs[i].id
			a.fontOpen = ""
		}
	})
	switch a.settingsTab {
	case "text":
		a.textSettings(c, k)
	case "general":
		if toggleRow(c, k, "Done notifications", "When a long command finishes in the background", !prefs.NoNotify) {
			a.changeSettings(func(s *settings) { s.NoNotify = !s.NoNotify })
		}
	default:
		a.themeSettings(c, k)
		ui.Box(c).Height(1).Background(k.sep)
		a.backgroundSettings(c, k)
	}
	ui.Box(c).Height(1).Background(k.sep)
	ui.Text(c, "Saved in "+tildePath(settingsPath())).FontSize(11).TextColor(k.textFaint).
		SingleLine().Ellipsis("…").MinWidth(0)
}

// themeSettings draws the themes, three to a row, and the choice to
// follow the system's appearance with a dark and a light one.
func (a *App) themeSettings(c *ui.Context, k *colors) {
	dark := c.Theme().Dark
	shown := prefs.themeFor(dark)
	ui.Column(c).Gap(8).Children(func() {
		ui.Text(c, "Theme").FontSize(12.5).TextColor(k.text)
		for row := 0; row < len(themes); row += 3 {
			ui.Row(c.Key(row)).Gap(8).Role(ui.RoleRadioGroup).Label("Theme").Children(func() {
				for _, t := range themes[row:min(row+3, len(themes))] {
					on := t == shown
					// Following the system, the theme of the other side
					// is marked too, in a dashed rim.
					paired := prefs.FollowSystem && !on && prefs.themeFor(!dark) == t
					if themeCard(c, k, t, on, paired).Clicked() && !on {
						a.pickTheme(t)
					}
				}
			})
		}
		hint := shown.mood
		if prefs.FollowSystem {
			side := map[bool]string{true: "dark", false: "light"}
			hint = fmt.Sprintf("The system is %s now, so %s; %s while it is %s. A theme you click takes its side.",
				side[dark], shown.name, prefs.themeFor(!dark).name, side[!dark])
		}
		ui.Text(c, hint).FontSize(11.5).TextColor(k.textFaint)
	})
	if toggleRow(c, k, "Match system appearance", "A dark theme while the system is dark, a light one while it is light", prefs.FollowSystem) {
		a.toggleFollowSystem(dark)
	}
}

// themeCard draws a theme's card: a little window in its colors over its
// name, in the selection while it is the theme shown.
func themeCard(c *ui.Context, k *colors, t *theme, on, paired bool) ui.Element {
	tk := themeColors[t.id]
	card := ui.Column(c.Key(t.id)).Grow(1).Basis(0).MinWidth(0).Radius(8).Clip().Cursor(ui.CursorPointer).
		Role(ui.RoleRadio).Label(t.name).Checked(on)
	switch {
	case on:
		card.Border(1, k.selEdge).Background(k.selFill)
	case paired:
		card.Border(1, k.selEdge).BorderStyle(ui.BorderDashed)
	case card.Hovered():
		card.Border(1, k.cardBorderFocused).Background(k.hover)
	default:
		card.Border(1, k.sep)
	}
	card.Transition(ui.ElementTransition{Colors: true, Duration: 120 * time.Millisecond})
	card.Children(func() {
		ui.Box(c).Height(46).Draw(func(p *ui.Painter, r ui.Rect) {
			p.Fill(r, tk.bg, 0)
			in := ui.Rect{X: r.X + 6, Y: r.Y + 6, W: r.W - 12, H: r.H - 12}
			p.Fill(in, tk.cardBorderFocused, 4)
			p.Fill(ui.Rect{X: in.X + 1, Y: in.Y + 1, W: in.W - 2, H: in.H - 2}, tk.card, 3)
			x, w := in.X+7, in.W-14
			for i, bar := range []struct {
				c ui.Color
				w float32
			}{{tk.accent, 0.7}, {ui.Hex(t.second), 0.46}, {tk.text.Alpha(0.45), 0.84}} {
				p.Fill(ui.Rect{X: x, Y: in.Y + 6 + float32(i)*7, W: w * bar.w, H: 3}, bar.c, 1.5)
			}
		})
		ui.Row(c).Padding(6, 8).Gap(6).AlignItems(ui.Center).Children(func() {
			name := ui.Text(c, t.name).FontSize(12).TextColor(k.text).SingleLine().Ellipsis("…").Grow(1).MinWidth(0)
			if on {
				name.FontWeight(600)
				ui.Box(c).Size(6, 6).Radius(3).Background(k.accent).Shrink(0)
			}
		})
	})
	return card
}

// backgroundSettings draws the background: none, Lex's patterns or the
// user's own picture, as thumbnails, the button choosing a picture, and
// while there is a background, the panes' opacity over it, how dim and
// how blurred it is, and for a picture, its fit.
func (a *App) backgroundSettings(c *ui.Context, k *colors) {
	ui.Row(c).Gap(6).AlignItems(ui.Center).Children(func() {
		ui.Text(c, "Background").FontSize(12.5).TextColor(k.text).Grow(1)
		if textButton(c, k, "Choose Image…").Clicked() {
			a.chooseBackground()
		}
	})
	pat, isPattern := patternOf(prefs.Background)
	ui.Row(c).Gap(6).Wrap().Role(ui.RoleRadioGroup).Label("Background").Children(func() {
		if bgThumb(c, k, "none", "None", nil, prefs.Background == "").Clicked() {
			a.setBackground("")
		}
		for _, p := range patterns {
			if bgThumb(c, k, p.id, p.name, a.patternThumb(p), isPattern && pat.id == p.id).Clicked() {
				a.setBackground(patternPrefix + p.id)
			}
		}
		if own := prefs.CustomBackground; own != "" {
			if bgThumb(c, k, "own", filepath.Base(own), a.customThumb(own), prefs.Background == own).Clicked() {
				a.setBackground(own)
			}
		}
	})
	switch {
	case prefs.Background == "":
		ui.Text(c, "A pattern of Lex's, or an image of yours, behind the title bar and the panes, which turn translucent over it.").
			FontSize(11.5).TextColor(k.textFaint)
		return
	case isPattern:
		ui.Text(c, pat.name+", drawn by Lex").FontSize(11.5).TextColor(k.textFaint)
	default:
		bg := a.background()
		ui.Row(c).Gap(6).AlignItems(ui.Center).Children(func() {
			ui.Text(c, filepath.Base(prefs.Background)).FontSize(11.5).TextColor(k.text).SingleLine().Ellipsis("…").
				MinWidth(0).Shrink(1).Tooltip(prefs.Background)
			switch {
			case bg == nil:
				ui.Text(c, "Loading…").FontSize(11.5).TextColor(k.textFaint).Shrink(0)
			case bg.err != "":
				ui.Text(c, bg.err).FontSize(11.5).TextColor(k.danger).Shrink(0)
			default:
				ui.Text(c, fmt.Sprintf("%d × %d", bg.w, bg.h)).FontSize(11.5).TextColor(k.textFaint).Shrink(0)
			}
		})
		fits := []string{"fill", "fit", "tile", "center"}
		at := 0
		for i, f := range fits {
			if prefs.BackgroundFit == f {
				at = i
			}
		}
		if i, ok := segmented(c, k, "Fit", []string{"Fill", "Fit", "Tile", "Center"}, at); ok {
			a.changeLook(func(s *settings) { s.BackgroundFit = fits[i] })
		}
	}
	op := prefs.paneOpacity()
	if v, ok := sliderRow(c, k, "Pane opacity", fmt.Sprintf("%.0f %%", op*100), "How much the panes hide the image", op, 0.3, 1); ok {
		a.changeLook(func(s *settings) { s.PaneOpacity = v })
	}
	label, hint := "Dim image", "Darkens the image, so that text stays readable"
	if !c.Theme().Dark {
		label, hint = "Fade image", "Lightens the image toward the theme"
	}
	dim := prefs.backgroundDim()
	if v, ok := sliderRow(c, k, label, fmt.Sprintf("%.0f %%", dim*100), hint, dim, 0, 0.8); ok {
		a.changeLook(func(s *settings) { s.BackgroundDim = v })
	}
	blur := float32(prefs.backgroundBlur())
	if v, ok := sliderRow(c, k, "Blur", fmt.Sprintf("%.0f px", blur), "Softens a busy image", blur, 0, 24); ok {
		a.changeLook(func(s *settings) { s.BackgroundBlur = float32(int(v + 0.5)) })
	}
}

// textSettings draws the fonts, the text's size, the rows' height, the
// cursor and the ligatures.
func (a *App) textSettings(c *ui.Context, k *colors) {
	a.fontPicker(c, k, "term", "Terminal font")
	a.fontPicker(c, k, "ui", "Interface font")

	// Text size: smaller, the size, bigger, and back to the default.
	settingRow(c, k, "Text size", func() {
		if iconButton(c, k, "list-remove", "Smaller Text", 26, 16).Tooltip(tip("Smaller Text", &cmdSmaller)).Clicked() {
			a.setFontSize(prefs.FontSize - 1)
		}
		ui.Text(c, fmt.Sprintf("%g", prefs.FontSize)).FontSize(13).FontWeight(600).TextColor(k.text).
			Width(38).TextAlign(ui.Center)
		if iconButton(c, k, "list-add", "Bigger Text", 26, 16).Tooltip(tip("Bigger Text", &cmdBigger)).Clicked() {
			a.setFontSize(prefs.FontSize + 1)
		}
		reset := ui.Text(c, "Reset").FontSize(12).Padding(3, 7).Radius(4).Margin(0, 0, 0, 4).Role(ui.RoleButton)
		if prefs.FontSize == defaultFontSize {
			reset.TextColor(k.textFaint.Alpha(0.5))
		} else {
			reset.TextColor(k.textMuted).Cursor(ui.CursorPointer)
			if reset.Hovered() {
				reset.Background(k.hover)
			}
			if reset.Clicked() {
				a.setFontSize(defaultFontSize)
			}
		}
	})

	// Line spacing: the option nearest the rows' height.
	at := 0
	for i, s := range lineSpacings {
		if abs32(s.height-prefs.lineHeight()) < abs32(lineSpacings[at].height-prefs.lineHeight()) {
			at = i
		}
	}
	labels := make([]string, len(lineSpacings))
	for i, s := range lineSpacings {
		labels[i] = s.label
	}
	if i, ok := segmented(c, k, "Line spacing", labels, at); ok {
		a.changeSettings(func(s *settings) { s.LineHeight = lineSpacings[i].height })
	}

	// The cursor: its shape, and whether it blinks.
	shapes := []string{"block", "bar", "underline"}
	at = int(prefs.cursorStyle())
	if i, ok := segmented(c, k, "Cursor", []string{"Block", "Bar", "Underline"}, at); ok {
		a.changeSettings(func(s *settings) { s.Cursor = shapes[i] })
	}
	if toggleRow(c, k, "Blinking cursor", "", !prefs.NoBlink) {
		a.changeSettings(func(s *settings) { s.NoBlink = !s.NoBlink })
	}
	if toggleRow(c, k, "Ligatures", "The font's coding ligatures, as → ≠ ≥, where it has them", !prefs.NoLigatures) {
		a.changeSettings(func(s *settings) { s.NoLigatures = !s.NoLigatures })
	}
}

// fontPicker draws the font of the terminals ("term") or of the rest of
// the window ("ui"): a button with its name in its face, which opens the
// list of the system's fonts under it.
func (a *App) fontPicker(c *ui.Context, k *colors, which, label string) {
	current, shown, tag := prefs.Font, prefs.Font, "Installed"
	if which == "ui" {
		current, shown = prefs.UIFont, prefs.UIFont
	}
	if shown == "" || shown == builtinFont && which == "term" {
		shown, tag = builtinFont, "Built in"
		if which == "ui" {
			shown, tag = "System", "Default"
		}
	}
	open := a.fontOpen == which
	ui.Column(c.Key("font-" + which)).Gap(6).Children(func() {
		ui.Text(c, label).FontSize(12.5).TextColor(k.text)
		b := ui.Row(c).Height(32).Padding(0, 10).Gap(8).AlignItems(ui.Center).Radius(6).Cursor(ui.CursorPointer).
			Role(ui.RoleComboBox).Label(label).Checked(open)
		switch {
		case open:
			b.Border(1, k.selEdge)
		case b.Hovered():
			b.Border(1, k.cardBorderFocused).Background(k.hover)
		default:
			b.Border(1, k.sep)
		}
		b.Children(func() {
			name := ui.Text(c, shown).FontSize(13).TextColor(k.text).Grow(1).MinWidth(0).SingleLine().Ellipsis("…")
			if which == "term" || current != "" {
				name.Font(shown)
			}
			ui.Text(c, tag).FontSize(11).TextColor(k.textFaint).Shrink(0)
			chevron(c, k, open)
		})
		if b.Clicked() {
			if open {
				a.fontOpen = ""
			} else {
				a.fontOpen, a.fontQuery = which, ""
				a.askFonts()
			}
		}
		if open {
			a.fontList(c, k, which, current)
		}
	})
}

// fontEntry is a font the list offers: its name, the value the settings
// keep for it, and what to say of it.
type fontEntry struct {
	name, value, tag string
	on               bool
}

// fontList lists the system's fonts, which a search narrows: the
// monospaced ones for the terminals, unless asked for all, and all of them,
// and the system's own, for the window.
func (a *App) fontList(c *ui.Context, k *colors, which, current string) {
	ui.Column(c).Radius(6).Border(1, k.sep).Clip().Children(func() {
		ui.TextInput(c, &a.fontQuery).Placeholder("Search fonts").Height(28).FontSize(12.5).Margin(6, 6, 2, 6).AutoFocus().
			Label("Search fonts")
		if a.fonts == nil {
			ui.Text(c, "Looking for the system's fonts…").FontSize(12).TextColor(k.textFaint).Padding(8, 10)
			return
		}
		q := strings.ToLower(strings.TrimSpace(a.fontQuery))
		var list []fontEntry
		if which == "ui" && strings.Contains("system", q) {
			list = append(list, fontEntry{"System", "", "Default", current == ""})
		}
		for _, f := range a.fonts {
			if which == "term" && !f.Monospaced && !a.fontAll || q != "" && !strings.Contains(strings.ToLower(f.Name), q) {
				continue
			}
			e := fontEntry{name: f.Name, value: f.Name}
			switch {
			case f.Name == builtinFont:
				e.tag = "Built in"
			case which == "term" && !f.Monospaced:
				e.tag = "Proportional"
			}
			if which == "term" && f.Name == builtinFont {
				e.value = ""
			}
			e.on = e.value == current
			list = append(list, e)
		}
		count := fmt.Sprintf("%d fonts", len(list))
		if which == "term" && !a.fontAll {
			count = fmt.Sprintf("%d monospaced fonts", len(list))
		}
		ui.Text(c, count).FontSize(11).TextColor(k.textFaint).Padding(4, 10, 2, 10)
		// So many rows, each in its own face, would cost more than they
		// help: a search finds the others.
		const most = 300
		ui.Scroll(c).MaxHeight(208).Padding(2, 4, 4, 4).Gap(1).Role(ui.RoleList).Label("Fonts").Children(func() {
			for _, e := range list[:min(len(list), most)] {
				row := ui.Row(c.Key(e.name)).Height(28).Padding(0, 8).Gap(8).AlignItems(ui.Center).Radius(4).Shrink(0).
					Cursor(ui.CursorPointer).Role(ui.RoleListItem).Label(e.name).Selected(e.on)
				if e.on {
					row.Background(k.selFill).Border(1, k.selEdge)
				} else if row.Hovered() {
					row.Background(k.hover)
				}
				row.Children(func() {
					name := ui.Text(c, e.name).FontSize(13).TextColor(k.text).Grow(1).MinWidth(0).SingleLine().Ellipsis("…")
					if e.value != "" || which == "term" {
						name.Font(e.name)
					}
					if e.tag != "" {
						ui.Text(c, e.tag).FontSize(10.5).TextColor(k.textFaint).Shrink(0)
					}
				})
				if row.Clicked() {
					a.pickFont(which, e.value)
				}
			}
			if len(list) == 0 {
				ui.Text(c, "No font matches “"+a.fontQuery+"”").FontSize(12).TextColor(k.textFaint).Padding(8)
			}
		})
		if which == "term" {
			ui.Box(c).Height(1).Background(k.sep)
			ui.Box(c).Padding(8, 10).Children(func() {
				if toggleRow(c, k, "Show proportional fonts", "", a.fontAll) {
					a.fontAll = !a.fontAll
				}
			})
		}
	})
}

// askFonts asks the system for its fonts, once, off the main thread.
func (a *App) askFonts() {
	if a.fontsAsked {
		return
	}
	a.fontsAsked = true
	go func() {
		fonts := ui.FontFamilies()
		a.laterService(func() { a.fonts = fonts })
	}()
}

// pickFont makes a font the terminals' or the window's, and closes the
// list.
func (a *App) pickFont(which, value string) {
	a.fontOpen = ""
	if which == "ui" {
		a.changeLook(func(s *settings) { s.UIFont = value })
		return
	}
	a.changeSettings(func(s *settings) { s.Font = value })
}

// chevron draws the arrow of a list that opens under its button.
func chevron(c *ui.Context, k *colors, open bool) {
	ui.Box(c).Size(10, 10).Shrink(0).Draw(func(p *ui.Painter, r ui.Rect) {
		y0, y1 := r.Y+3, r.Y+7
		if open {
			y0, y1 = y1, y0
		}
		p.StrokePolyline([][2]float32{{r.X + 1, y0}, {r.X + 5, y1}, {r.X + 9, y0}}, 1.3, k.textMuted)
	})
}

// bgThumb draws a background the settings offer: its little picture, or
// its name when it has none, in the accent's rim while it is the one shown.
func bgThumb(c *ui.Context, k *colors, id, name string, pic *ui.Bitmap, on bool) ui.Element {
	b := ui.Box(c.Key("bg-"+id)).Size(52, 36).Radius(6).Clip().Center().Cursor(ui.CursorPointer).
		Role(ui.RoleRadio).Label(name).Checked(on).Tooltip(name).Background(k.card)
	rim := float32(1)
	switch {
	case on:
		rim = 2
		b.Border(2, k.accent)
	case b.Hovered():
		b.Border(1, k.cardBorderFocused)
	default:
		b.Border(1, k.sep)
	}
	if pic != nil {
		// Inside the rim, which it would hide.
		b.Draw(func(p *ui.Painter, r ui.Rect) {
			p.Image(pic, ui.Rect{X: r.X + rim, Y: r.Y + rim, W: r.W - 2*rim, H: r.H - 2*rim}, ui.Cover)
		})
	} else {
		b.Children(func() {
			ui.Text(c, map[bool]string{true: "None", false: "…"}[id == "none"]).FontSize(11).TextColor(k.textFaint)
		})
	}
	return b
}

// textButton is a small button of a word, in a hairline rim.
func textButton(c *ui.Context, k *colors, label string) ui.Element {
	b := ui.Box(c.Key(label)).Height(26).Padding(0, 10).Center().Radius(buttonR).Border(1, k.sep).
		Cursor(ui.CursorPointer).Role(ui.RoleButton).Label(label)
	if b.Pressed() {
		b.Background(k.pressed)
	} else if b.Hovered() {
		b.Background(k.hover)
	}
	b.Transition(ui.ElementTransition{Colors: true, Duration: 120 * time.Millisecond})
	b.Children(func() {
		ui.Text(c, label).FontSize(12).TextColor(k.text).SingleLine()
	})
	return b
}

// sliderRow draws a setting of a value between lo and hi: its label and
// the value shown on a line, the slider, and a hint under it. It returns
// the value the slider moved to.
func sliderRow(c *ui.Context, k *colors, label, shown, hint string, value, lo, hi float32) (float32, bool) {
	v := float64(value)
	ui.Column(c.Key("slider-" + label)).Gap(4).Children(func() {
		ui.Row(c).Gap(8).AlignItems(ui.Center).Children(func() {
			ui.Text(c, label).FontSize(12.5).TextColor(k.text).Grow(1)
			ui.Text(c, shown).FontSize(12).FontWeight(600).TextColor(k.textMuted)
		})
		ui.Slider(c, &v, float64(lo), float64(hi)).WidthPercent(100).Label(label)
		if hint != "" {
			ui.Text(c, hint).FontSize(11).TextColor(k.textFaint)
		}
	})
	return float32(v), float32(v) != value
}

// settingRow lays out a setting: its label on the left, its control on
// the right.
func settingRow(c *ui.Context, k *colors, label string, control func()) {
	ui.Row(c).Gap(8).AlignItems(ui.Center).Children(func() {
		ui.Text(c, label).FontSize(12.5).TextColor(k.text).Grow(1).MinWidth(0)
		ui.Row(c).Gap(2).AlignItems(ui.Center).Shrink(0).Children(control)
	})
}

// segmented draws a label over a row of options, one of which is chosen
// (at), in patina's selection: it returns the option clicked.
func segmented(c *ui.Context, k *colors, label string, options []string, at int) (chosen int, ok bool) {
	chosen = -1
	ui.Column(c.Key("seg-" + label)).Gap(6).Children(func() {
		ui.Text(c, label).FontSize(12.5).TextColor(k.text)
		chosen, ok = optionRow(c, k, label, options, at, 30, 12.5, true)
	})
	return chosen, ok
}

// tabRow draws the panel's sections, as a small row of options.
func tabRow(c *ui.Context, k *colors, label string, options []string, at int) (int, bool) {
	return optionRow(c, k, label, options, at, 26, 12, false)
}

// optionRow draws a row of options, one of which is chosen (at), in
// patina's selection: it returns the option clicked. The options share
// the row's width evenly when even, else each is as wide as its word.
func optionRow(c *ui.Context, k *colors, label string, options []string, at int, height, size float32, even bool) (chosen int, ok bool) {
	chosen = -1
	group := ui.Row(c).Height(height).Padding(2).Gap(2).Radius(6).Border(1, k.sep).AlignItems(ui.Stretch).
		Role(ui.RoleRadioGroup).Label(label)
	group.Children(func() {
		for i, o := range options {
			b := ui.Box(c.Key(i)).Center().Padding(0, 8).Radius(4).Cursor(ui.CursorPointer).
				Role(ui.RoleRadio).Label(o).Checked(i == at)
			if even {
				b.Grow(1).Basis(0)
			} else {
				b.Shrink(0)
			}
			col := k.textMuted
			if i == at {
				b.Background(k.selFill).Border(1, k.selEdge)
				col = k.text
			} else if b.Hovered() {
				b.Background(k.hover)
				col = k.text
			}
			b.Transition(ui.ElementTransition{Colors: true, Duration: 120 * time.Millisecond})
			b.Children(func() {
				t := ui.Text(c, o).FontSize(size).TextColor(col).SingleLine()
				if i == at {
					t.FontWeight(600)
				}
			})
			if b.Clicked() && i != at {
				chosen = i
			}
		}
	})
	return chosen, chosen >= 0
}

// toggleRow draws a setting that is on or off, with a hint under its
// label, and a switch in the accent while on: it reports a click.
func toggleRow(c *ui.Context, k *colors, label, hint string, on bool) (clicked bool) {
	row := ui.Row(c.Key("toggle-" + label)).Gap(10).AlignItems(ui.Center).Cursor(ui.CursorPointer).Role(ui.RoleSwitch).Label(label).Checked(on)
	row.Children(func() {
		ui.Column(c).Grow(1).MinWidth(0).Gap(2).Children(func() {
			ui.Text(c, label).FontSize(12.5).TextColor(k.text)
			if hint != "" {
				ui.Text(c, hint).FontSize(11.5).TextColor(k.textFaint)
			}
		})
		track := ui.Box(c).Size(34, 20).Radius(10).Shrink(0)
		knob, x := k.textFaint, float32(3)
		if on {
			track.Background(k.accent)
			knob, x = k.onAccent, 34-14-3
		} else {
			track.Background(k.switchOff).Border(1, k.sep)
		}
		if row.Hovered() && !on {
			knob = k.text
		}
		track.Draw(func(p *ui.Painter, r ui.Rect) {
			p.Fill(ui.Rect{X: r.X + x, Y: r.Y + 3, W: 14, H: 14}, knob, 7)
		})
	})
	return row.Clicked()
}

func abs32(f float32) float32 {
	if f < 0 {
		return -f
	}
	return f
}

// tildePath writes a path in the home directory from "~".
func tildePath(path string) string {
	home, err := os.UserHomeDir()
	if err != nil || home == "" {
		return path
	}
	if rest, ok := strings.CutPrefix(path, home+string(filepath.Separator)); ok {
		return "~" + string(filepath.Separator) + rest
	}
	return path
}
