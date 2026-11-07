package main

import (
	"slices"
	"time"

	"github.com/egoist/mygo/ui"
)

// The window's animations, which the settings turn on (settings.Animate):
// panes move to their places as a tab splits, closes and zooms, coming in
// and going out a little smaller and faded, and the tab chosen slides in
// as the one before slides out, the way of the tab bar. Off, every change
// shows at once. MyGo keeps them still while the system asks for less
// motion (Preferences.ReduceMotion), fading what comes and goes in place.

// paneEase is how panes move: quickly at first, a hint past their place,
// and back.
var paneEase = ui.CubicBezier(0.2, 1.12, 0.3, 1)

// paneMotion moves a pane to its cell, and brings a new one in, or a
// closed one out, from a little smaller. Its colors change as they do
// still (paneCard).
var paneMotion = ui.ElementTransition{
	Position:     true,
	Size:         true,
	Colors:       true,
	Duration:     280 * time.Millisecond,
	Ease:         paneEase,
	ExitDuration: 160 * time.Millisecond,
	ExitEase:     ui.CubicBezier(0.4, 0, 1, 1),
	Enter:        &ui.Motion{Opacity: 0, Scale: 0.94},
	Exit:         &ui.Motion{Opacity: 0, Scale: 0.94},
}

// dividerMotion moves a divider with the panes on either side of it.
var dividerMotion = ui.ElementTransition{
	Position:     true,
	Size:         true,
	Duration:     280 * time.Millisecond,
	Ease:         paneEase,
	ExitDuration: -1,
	Enter:        &ui.Motion{Opacity: 0},
}

// motion returns t, or, while a divider is dragged, t showing every
// change at once: the panes follow the pointer, rather than trail it.
func (a *App) motion(t ui.ElementTransition) ui.ElementTransition {
	if a.dragged >= a.frame-1 {
		t.Duration, t.Ease, t.Animation = -1, nil, nil
	}
	return t
}

// tabSlide is how far, in DIPs, the panes of a tab coming in or going out
// slide.
const tabSlide = 36

// tabMotion brings the panes of the tab chosen in from dir, 1 from the
// right and -1 from the left, faded, as those of the tab before leave
// the other way (tabExit), quicker, so that the two hardly show at once.
func tabMotion(dir float32) ui.ElementTransition {
	return ui.ElementTransition{
		Duration:     240 * time.Millisecond,
		ExitDuration: 110 * time.Millisecond,
		ExitEase:     ui.CubicBezier(0.4, 0, 1, 1),
		Enter:        &ui.Motion{X: tabSlide * dir, Opacity: 0},
		Exit:         &tabExit,
	}
}

// tabExit is where the panes of the tab before go: the way of the tab
// bar, half as far as those coming in come.
var tabExit = ui.Motion{Opacity: 0}

// tabExitFor returns where the panes of the tab before go, as the tab
// chosen comes from dir.
func tabExitFor(dir float32) ui.Motion {
	return ui.Motion{X: -tabSlide / 2 * dir, Opacity: 0}
}

// tabItemMotion is how a tab of the tab bar moves: to its place as tabs
// open, close and reorder, and, animated, opening from nothing and
// closing to nothing.
func tabItemMotion() ui.ElementTransition {
	t := ui.ElementTransition{Colors: true, Position: true, Duration: 160 * time.Millisecond}
	if prefs.Animate {
		t.Size = true
		t.Enter = &ui.Motion{Collapse: true, Opacity: 0}
		t.Exit = &ui.Motion{Collapse: true, Opacity: 0}
		t.ExitDuration = 140 * time.Millisecond
	}
	return t
}

// followTab notes the tab the frame shows, and the way it comes from when
// it is another than the last frame's: from the right when it is right of
// that one, or takes the place of one closed, and from the left else.
func (a *App) followTab() {
	t := a.tab()
	if t == a.shownTab {
		a.shownAt = a.active
		return
	}
	switch {
	case a.active > a.shownAt:
		a.tabDir = 1
	case a.active < a.shownAt:
		a.tabDir = -1
	case slices.Contains(a.tabs, a.shownTab):
		// The tab shown moved: the one now in its place came from where
		// that one went.
		a.tabDir = -1
		if slices.Index(a.tabs, a.shownTab) < a.active {
			a.tabDir = 1
		}
	default:
		a.tabDir = 1
	}
	a.shownTab, a.shownAt = t, a.active
}
