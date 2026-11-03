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
	ui.PopoverBase(c, gear, &a.settingsOpen, func(panel ui.Element) {
		panel.AttachTo(gear, ui.AnchorBottomRight, ui.AnchorTopRight).Margin(8, 0, 0, 0).
			Width(320).Padding(14).Radius(8).Background(k.panel).Border(1, k.panelBorder).
			Shadow(0, 12, 32, 0, k.shadow).Gap(14).Role(ui.RoleDialog).Label("Settings")
		a.settingsPanel(c, k)
	})
	// The focus goes back to the pane the panel took it from.
	if was && !a.settingsOpen {
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

func (a *App) settingsPanel(c *ui.Context, k *colors) {
	ui.Text(c, "Settings").FontSize(14).FontWeight(700).TextColor(k.text)

	// Appearance.
	modes := []string{"dark", "light", "system"}
	at := 0
	for i, m := range modes {
		if appearance() == m {
			at = i
		}
	}
	if i, ok := segmented(c, k, "Appearance", []string{"Dark", "Light", "System"}, at); ok {
		a.setAppearance(modes[i])
	}

	// Text size: smaller, the size, bigger, and back to the default.
	settingRow(c, k, "Text size", func() {
		if iconButton(c, k, "list-remove", "Smaller Text", 26, 14).Tooltip(tip("Smaller Text", &cmdSmaller)).Clicked() {
			a.setFontSize(prefs.FontSize - 1)
		}
		ui.Text(c, fmt.Sprintf("%g", prefs.FontSize)).FontSize(13).FontWeight(600).TextColor(k.text).
			Width(38).TextAlign(ui.Center)
		if iconButton(c, k, "list-add", "Bigger Text", 26, 14).Tooltip(tip("Bigger Text", &cmdBigger)).Clicked() {
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
	at = 0
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

	ui.Box(c).Height(1).Background(k.sep)

	if toggleRow(c, k, "Ligatures", "JetBrains Mono's coding ligatures, as → ≠ ≥", !prefs.NoLigatures) {
		a.changeSettings(func(s *settings) { s.NoLigatures = !s.NoLigatures })
	}
	if toggleRow(c, k, "Done notifications", "When a long command finishes in the background", !prefs.NoNotify) {
		a.changeSettings(func(s *settings) { s.NoNotify = !s.NoNotify })
	}

	ui.Text(c, "Saved in "+tildePath(settingsPath())).FontSize(11).TextColor(k.textFaint).
		SingleLine().Ellipsis("…").MinWidth(0)
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
	ui.Column(c).Gap(6).Children(func() {
		ui.Text(c, label).FontSize(12.5).TextColor(k.text)
		group := ui.Row(c).Height(30).Padding(2).Gap(2).Radius(6).Border(1, k.sep).AlignItems(ui.Stretch).
			Role(ui.RoleRadioGroup).Label(label)
		group.Children(func() {
			for i, o := range options {
				b := ui.Box(c.Key(i)).Grow(1).Basis(0).Center().Radius(4).Cursor(ui.CursorPointer).
					Role(ui.RoleRadio).Label(o).Checked(i == at)
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
					t := ui.Text(c, o).FontSize(12.5).TextColor(col).SingleLine()
					if i == at {
						t.FontWeight(600)
					}
				})
				if b.Clicked() && i != at {
					chosen = i
				}
			}
		})
	})
	return chosen, chosen >= 0
}

// toggleRow draws a setting that is on or off, with a hint under its
// label, and a switch in the accent while on: it reports a click.
func toggleRow(c *ui.Context, k *colors, label, hint string, on bool) (clicked bool) {
	row := ui.Row(c).Gap(10).AlignItems(ui.Center).Cursor(ui.CursorPointer).Role(ui.RoleSwitch).Label(label).Checked(on)
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
			knob, x = k.bg, 34-14-3
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
