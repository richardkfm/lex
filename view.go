package main

import (
	"runtime"
	"strings"
	"time"

	"github.com/egoist/mygo/plugins/terminal"
	"github.com/egoist/mygo/ui"
)

const (
	gap       = 8  // between panes, and around them
	titleH    = 44 // the title bar
	cardR     = 6  // the panes' corners, as Breeze rounds them
	headerH   = 33 // the panes' headers
	activeFor = 1500 * time.Millisecond
)

func (a *App) view(c *ui.Context) {
	a.services = c.Services()
	a.runPosted()
	k := colorsOf(c)
	uiTheme(c, k)
	a.focusedWin = a.win == nil || a.win.IsFocused()
	c.Root().Background(k.bg)
	root := ui.Column(c).Fill()
	// The background image shows through the title bar and the gaps, and
	// faintly through the panes, dimmed toward black, or in a light theme
	// toward its own color.
	bg := a.background()
	a.bgShown = bg != nil && bg.bmp != nil
	if a.bgShown {
		dim := ui.RGBA(0, 0, 0, prefs.backgroundDim())
		if !c.Theme().Dark {
			dim = k.bg.Alpha(prefs.backgroundDim())
		}
		fit := prefs.backgroundFit()
		root.Draw(func(p *ui.Painter, r ui.Rect) {
			drawBackdrop(p, r, bg, fit)
			p.Fill(r, dim, 0)
		})
	} else {
		root.Background(k.bg)
	}
	root.Children(func() {
		a.titleBar(c, k)
		ui.Column(c).Grow(1).MinHeight(0).Padding(0, gap, gap, gap).Children(func() {
			if t := a.tab(); t != nil {
				a.tabContent(c, k, t)
			}
		})
	})
	if a.err != "" {
		a.errorBanner(c, k)
	}
	a.shortcuts(c)
	a.palette(c, k)
	if a.saveDue && time.Since(a.lastSave) > time.Second {
		a.save()
	}
	// The window's title, which the Window menu and Mission Control show,
	// is the active tab's.
	if t := a.tab(); t != nil && a.win != nil {
		name, detail := t.label()
		if title := strings.TrimSpace(name + " " + detail); title != a.title {
			a.title = title
			a.win.SetTitle(title)
		}
	}
}

// overImage returns a face of color col, translucent as the panes are
// over the background image when the window shows one.
func (a *App) overImage(col ui.Color) ui.Color {
	if a.bgShown {
		return col.Alpha(prefs.paneOpacity())
	}
	return col
}

// uiTheme gives MyGo's own widgets, as the settings' sliders and the
// inputs, the theme's accent and text, and the window's text the font the
// settings chose.
func uiTheme(c *ui.Context, k *colors) {
	t := *c.Theme()
	t.Accent, t.AccentHover, t.AccentPressed, t.AccentText = k.accent, k.accent, k.accent.Alpha(0.85), k.onAccent
	t.InactiveAccent = k.accent
	t.Text, t.TextMuted = k.text, k.textMuted
	t.Selection = k.accent.Alpha(0.3)
	if prefs.UIFont != "" {
		t.Font = prefs.UIFont
	}
	c.SetTheme(&t)
}

// ownControls tells that the title bar draws the window's controls: on
// Windows, whose own would look like none of the rest of it, and on Linux,
// where GTK's take more room than their buttons need. GTK 3 puts its
// buttons in a header bar that also reserves room for a title of at least
// five characters, which stays empty, so Lex's buttons would sit about 90
// pixels to the left of them.
var ownControls = runtime.GOOS == "windows" || runtime.GOOS == "linux"

// titleBar draws the title bar under the window's controls: the host,
// the tabs, and the buttons of the command palette, a new tab and the
// settings.
func (a *App) titleBar(c *ui.Context, k *colors) {
	bar := c.TitleBar()
	left := bar.Left
	if left == 0 {
		left = 12 // in full screen, or beside no controls
	} else {
		left += 14
	}
	row := ui.Row(c).Height(titleH).Padding(0, max(bar.Right, 10), 0, left).Gap(14).AlignItems(ui.Center).DragWindow()
	// Over a background image, the title bar is backed by the window's
	// color as translucent as the panes, so that its text and buttons
	// stay readable on a picture of the other lightness.
	if a.bgShown {
		row.Background(a.overImage(k.bg))
	}
	row.Children(func() {
		a.hostChip(c, k)
		a.tabStrip(c, k)
		// The title bar between the tabs and the buttons, which drags the
		// window, and a double click on which zooms it.
		ui.Spacer(c).MinWidth(titleFree - 14)
		ui.Row(c).Gap(2).AlignItems(ui.Center).Shrink(0).Children(func() {
			if iconButton(c, k, "quickopen", "Command Palette", 30, 16).Tooltip(tip("Command Palette", &cmdPalette)).Clicked() {
				a.openPalette()
			}
			if iconButton(c, k, "list-add", "New Tab", 30, 16).Tooltip(tip("New Tab", &cmdNewTab)).Clicked() {
				a.newTab(a.currentDir())
			}
			a.settingsButton(c, k)
		})
		a.windowControls(c, k)
	})
}

// windowControls draws the buttons that minimize, maximize and close the
// window, when it has none of its own: they are the title bar's other
// buttons, and close's face turns red under the pointer.
func (a *App) windowControls(c *ui.Context, k *colors) {
	win := a.win
	if !ownControls || win == nil || win.IsFullScreen() {
		return
	}
	maximized := win.IsMaximized()
	ui.Row(c).Gap(2).AlignItems(ui.Center).Shrink(0).Children(func() {
		ui.Box(c).Size(1, 16).Margin(0, 6).Background(k.sep)
		if iconButton(c, k, "window-minimize", "Minimize", 30, 16).Tooltip("Minimize").Clicked() {
			win.Minimize()
		}
		glyph, label := "window-maximize", "Maximize"
		if maximized {
			glyph, label = "window-restore", "Restore"
		}
		if iconButton(c, k, glyph, label, 30, 16).Tooltip(label).Clicked() {
			if maximized {
				win.Unmaximize()
			} else {
				win.Maximize()
			}
		}
		b := ui.Box(c).Size(30, 30).Center().Radius(buttonR).Cursor(ui.CursorPointer).Role(ui.RoleButton).Label("Close").Tooltip("Close")
		col := k.iconMuted
		if b.Pressed() {
			b.Background(k.danger.Alpha(0.8))
			col = k.bg
		} else if b.Hovered() {
			b.Background(k.danger)
			col = k.bg
		}
		b.Transition(ui.ElementTransition{Colors: true, Duration: 120 * time.Millisecond})
		b.Children(func() {
			ui.Icon(c, icon("window-close")).Size(16, 16).TextColor(col)
		})
		if b.Clicked() {
			win.Close()
		}
	})
}

// buttonR rounds the buttons' faces.
const buttonR = 5

// tip is the tooltip of a button that runs a command: its label and its
// keys.
func tip(label string, cmd *command) string {
	if cmd.Keys == "" {
		return label
	}
	return label + "  " + cmd.Keys
}

// iconButton is a borderless button of an icon, with a face on hover;
// label names it for screen readers and its tooltip.
func iconButton(c *ui.Context, k *colors, name, label string, size, iconSize float32) ui.Element {
	b := ui.Box(c).Size(size, size).Center().Radius(min(buttonR, size/4)).Cursor(ui.CursorPointer).Role(ui.RoleButton).Label(label)
	if b.Pressed() {
		b.Background(k.pressed)
	} else if b.Hovered() {
		b.Background(k.hover)
	}
	b.Transition(ui.ElementTransition{Colors: true, Duration: 120 * time.Millisecond})
	col := k.iconMuted
	if b.Hovered() {
		col = k.text
	}
	b.Children(func() {
		ui.Icon(c, icon(name)).Size(iconSize, iconSize).TextColor(col)
	})
	return b
}

// tabContent lays out the panes of a tab, or the one zoomed.
func (a *App) tabContent(c *ui.Context, k *colors, t *Tab) {
	if t.Zoom != nil {
		a.paneCard(c, k, t, t.Zoom).Grow(1)
		return
	}
	a.node(c, k, t, t.Root).Grow(1)
}

// node lays out a node of the tree of splits.
func (a *App) node(c *ui.Context, k *colors, t *Tab, n *Node) ui.Element {
	if n.Pane != nil {
		return a.paneCard(c, k, t, n.Pane)
	}
	var box ui.Element
	if n.Vertical {
		box = ui.Column(c.Key(n.ID))
	} else {
		box = ui.Row(c.Key(n.ID))
	}
	box.MinWidth(0).MinHeight(0).AlignItems(ui.Stretch)
	bounds := box.Bounds()
	// The tree may change as its panes build, as a split: build the
	// children it has now.
	first, second, ratio := n.A, n.B, n.Ratio
	box.Children(func() {
		a.node(c, k, t, first).Grow(ratio).Basis(0).MinWidth(0).MinHeight(0)
		div := ui.Box(c.Key("divider")).Role(ui.RoleSplitter).Label("Divider")
		if n.Vertical {
			div.Height(gap).Cursor(ui.CursorResizeRow)
		} else {
			div.Width(gap).Cursor(ui.CursorResizeColumn)
		}
		if dx, dy, ok := div.Dragged(); ok {
			total := bounds.W - gap
			d := dx
			if n.Vertical {
				total, d = bounds.H-gap, dy
			}
			if total > 0 {
				n.Ratio = min(max(n.Ratio+d/total, 0.08), 0.92)
				a.changed()
			}
		}
		if div.DoubleClicked() {
			n.Ratio = 0.5
			a.changed()
		}
		// The divider under the pointer is a line along the gap, in the
		// second color, and in the accent while it moves.
		if div.Hovered() || div.Dragging() || div.Pressed() {
			col := k.hover.Alpha(0.6)
			if div.Dragging() || div.Pressed() {
				col = k.accent.Alpha(0.7)
			}
			div.Draw(func(p *ui.Painter, r ui.Rect) {
				if n.Vertical {
					p.Fill(ui.Rect{X: r.X, Y: r.Y + r.H/2 - 1, W: r.W, H: 2}, col, 1)
				} else {
					p.Fill(ui.Rect{X: r.X + r.W/2 - 1, Y: r.Y, W: 2, H: r.H}, col, 1)
				}
			})
		}
		a.node(c, k, t, second).Grow(1 - ratio).Basis(0).MinWidth(0).MinHeight(0)
	})
	return box
}

// paneCard draws a pane: a flat card with a header over its terminal, in
// a hairline rim, brighter round the focused pane, whose glyph in the
// header takes the accent.
func (a *App) paneCard(c *ui.Context, k *colors, t *Tab, p *Pane) ui.Element {
	focused := t.Focus == p
	card := ui.Column(c.Key(p.ID)).Radius(cardR).Clip().MinWidth(0).MinHeight(0)
	border := k.cardBorder
	if focused {
		border = k.cardBorderFocused
	}
	// A pane of a kind to keep in mind takes its tint in its rim, and as
	// the settings ask, in its header and over its face.
	tn := paneTint(k, p.info)
	face := k.card
	if tn.on {
		border = tn.col.Alpha(0.5)
		if focused {
			border = tn.col.Alpha(0.85)
		}
		if tn.wash {
			face = mixColor(face, tn.col, map[bool]float32{true: 0.06, false: 0.05}[c.Theme().Dark])
		}
	}
	card.Background(a.overImage(face)).Border(1, border)
	card.Transition(ui.ElementTransition{Colors: true, Duration: 160 * time.Millisecond})
	hovered := card.Hovered()
	card.Children(func() {
		a.paneHeader(c, k, t, p, focused, hovered, tn)
		body := ui.Box(c).Grow(1).MinHeight(0).Padding(0, 5, 6, 5)
		body.Children(func() {
			if p.term == nil {
				return
			}
			tv := terminal.View(c, p.term).Fill()
			if a.focusReq == p && a.tab() == t {
				tv.Focus()
				a.focusReq = nil
			}
			if tv.Focused() && t.Focus != p {
				t.setFocus(p)
				a.changed()
			}
			if tv.Focused() {
				p.attention = false
			}
		})
	})
	p.bounds = card.Bounds()
	if card.Pressed() && t.Focus != p {
		t.setFocus(p)
		a.focusReq = p
	}
	return card
}

// paneHeader draws a pane's program, title and directory, its activity,
// and its buttons.
func (a *App) paneHeader(c *ui.Context, k *colors, t *Tab, p *Pane, focused, hovered bool, tn tint) {
	name, detail := p.label()
	prog := programOf(p.info.Program)
	if p.info.Idle || p.info.Program == "" {
		prog = programOf(p.info.Shell)
	}
	h := ui.Row(c).Height(headerH).Padding(0, 8, 0, 12).Gap(7).AlignItems(ui.Center).MinWidth(0)
	if tn.header {
		h.Background(tn.col.Alpha(map[bool]float32{true: 0.12, false: 0.10}[c.Theme().Dark]))
		line := tn.col.Alpha(0.25)
		h.Draw(func(p *ui.Painter, r ui.Rect) { p.Fill(ui.Rect{X: r.X, Y: r.Y + r.H - 1, W: r.W, H: 1}, line, 0) })
	}
	if h.DoubleClicked() {
		t.setFocus(p)
		a.toggleZoom()
	}
	if h.Clicked() {
		t.setFocus(p)
		a.focusReq = p
	}
	h.Children(func() {
		glyph := k.iconMuted
		if focused {
			glyph = k.accent
		}
		ui.Icon(c, icon(prog.Glyph)).Size(15, 15).TextColor(glyph)
		ui.Row(c).Grow(1).MinWidth(0).Gap(5).AlignItems(ui.Center).ClipX().Children(func() {
			ui.Text(c, name).FontSize(12.5).FontWeight(600).TextColor(k.text).SingleLine().Ellipsis("…").Shrink(0).MaxWidthPercent(80)
			if detail != "" {
				ui.Text(c, detail).FontSize(12.5).FontWeight(500).TextColor(k.text.Alpha(0.86)).SingleLine().Ellipsis("…").Shrink(1).MinWidth(0)
			}
			a.activityBadge(c, k, p)
		})
		if tn.header {
			tintChip(c, k, tn)
		}
		show := focused || hovered
		ui.Row(c).Gap(1).AlignItems(ui.Center).Opacity(map[bool]float32{true: 1, false: 0}[show]).Children(func() {
			if !show {
				return
			}
			if iconButton(c, k, "view-split-left-right", "Split Right", 26, 16).Tooltip(tip("Split Right", &cmdSplitRight)).Clicked() {
				t.setFocus(p)
				a.split(false)
			}
			if iconButton(c, k, "view-split-top-bottom", "Split Down", 26, 16).Tooltip(tip("Split Down", &cmdSplitDown)).Clicked() {
				t.setFocus(p)
				a.split(true)
			}
			zoom, label := "view-fullscreen", "Zoom"
			if t.Zoom == p {
				zoom, label = "view-restore", "Unzoom"
			}
			if iconButton(c, k, zoom, "Zoom", 26, 16).Tooltip(tip(label, &cmdZoom)).Clicked() {
				t.setFocus(p)
				a.toggleZoom()
			}
			if iconButton(c, k, "window-close", "Close Pane", 26, 16).Tooltip(tip("Close Pane", &cmdClosePane)).Clicked() {
				a.later(c, func() { a.closePane(p) })
			}
		})
	})
}

// tintChip names the kind of a tinted pane in its header: a dot of its
// color and a word, in a pill of it.
func tintChip(c *ui.Context, k *colors, tn tint) {
	ui.Row(c).Height(20).Padding(0, 8).Gap(5).AlignItems(ui.Center).Radius(10).Shrink(0).
		Background(tn.col.Alpha(0.14)).Border(1, tn.col.Alpha(0.4)).Label(tn.chip).Children(func() {
		ui.Box(c).Size(6, 6).Radius(3).Background(tn.col)
		ui.Text(c, tn.chip).FontSize(11).FontWeight(600).TextColor(k.text).SingleLine()
	})
}

// status is what a pane is doing.
type status int

const (
	statusIdle      status = iota // a shell at its prompt
	statusRunning                 // a program printing
	statusQuiet                   // a program running quietly
	statusAttention               // done or ringing out of sight
)

func (a *App) statusOf(c *ui.Context, p *Pane) status {
	if p.attention {
		return statusAttention
	}
	if p.info.Idle || p.info.Program == "" || programOf(p.info.Program).Shell {
		return statusIdle
	}
	last := time.Unix(0, p.lastData.Load())
	if since := c.Now().Sub(last); since < activeFor {
		c.After(activeFor - since)
		return statusRunning
	}
	return statusQuiet
}

// activityBadge shows what a pane does: a pulse while its program
// prints, a dot when it asks for attention.
func (a *App) activityBadge(c *ui.Context, k *colors, p *Pane) {
	switch a.statusOf(c, p) {
	case statusRunning:
		activityDot(c, k.busy, 6)
	case statusAttention:
		ui.Box(c).Size(7, 7).Radius(4).Background(k.attention).Margin(0, 0, 0, 2).Tooltip("Needs attention")
	}
}

// activityDot is the dot of a program printing: a solid dot in a soft
// halo. It does not animate, as drawing frames all along would cost more
// than it tells.
func activityDot(c *ui.Context, col ui.Color, size float32) ui.Element {
	e := ui.Box(c).Size(size+6, size+6).Margin(0, 0, 0, 1).Tooltip("Printing")
	e.Draw(func(p *ui.Painter, r ui.Rect) {
		cx, cy := r.X+r.W/2, r.Y+r.H/2
		halo := size/2 + 2.5
		p.Fill(ui.Rect{X: cx - halo, Y: cy - halo, W: 2 * halo, H: 2 * halo}, col.Alpha(0.22), halo)
		p.Fill(ui.Rect{X: cx - size/2, Y: cy - size/2, W: size, H: size}, col, size/2)
	})
	return e
}

// errorBanner shows what went wrong with the server.
func (a *App) errorBanner(c *ui.Context, k *colors) {
	ui.Overlay(c, func() {
		ui.Row(c).Absolute().Bottom(18).Left(0).Right(0).Justify(ui.Center).PassThrough().Children(func() {
			ui.Row(c).Gap(10).Padding(8, 10, 8, 14).Radius(cardR).Background(k.panel).Border(1, k.panelBorder).
				Shadow(0, 8, 24, 0, k.shadow).AlignItems(ui.Center).Children(func() {
				ui.Box(c).Size(8, 8).Radius(4).Background(k.danger)
				ui.Text(c, a.err).FontSize(12.5).TextColor(k.text).MaxWidth(520)
				if iconButton(c, k, "window-close", "Dismiss", 22, 14).Clicked() {
					a.err = ""
				}
			})
		})
	})
}
