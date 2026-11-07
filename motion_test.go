package main

import (
	"strings"
	"testing"
	"time"

	"lex/internal/rex"
)

// TestPaneMotion splits a pane with the animations on: the pane narrows
// over frames, rather than at once, while its terminal keeps its columns,
// and takes those of its new width once it is there. Off, the split shows
// at once.
func TestPaneMotion(t *testing.T) {
	a, tt := newTestApp(t)
	prefs.Animate = true
	t.Cleanup(func() { prefs.Animate = false })
	p := a.tab().Focus
	waitFor(t, tt, "the shell", func() bool { return p.term != nil && strings.Contains(p.term.Text(), testShell.prompt) })
	tt.Frame()
	full := p.bounds.W
	cols, _ := p.term.Size()

	a.split(false)
	var widths []float32
	var seen []int
	for range 12 {
		tt.Advance(40 * time.Millisecond)
		tt.Frame()
		c, _ := p.term.Size()
		widths, seen = append(widths, p.bounds.W), append(seen, c)
	}
	half := widths[len(widths)-1]
	for i, w := range widths {
		if abs(w-half) > 1 && seen[i] != cols {
			t.Errorf("the terminal took %d columns while its pane moved, %v wide", seen[i], w)
		}
	}
	if half > full*0.6 {
		t.Fatalf("the split pane is %v wide of %v", half, full)
	}
	between := false
	for _, w := range widths {
		if w > half+20 && w < full-20 {
			between = true
		}
	}
	if !between {
		t.Errorf("the pane went from %v to %v wide in no frame between: %v", full, half, widths)
	}
	if c, _ := p.term.Size(); c >= cols {
		t.Errorf("the terminal kept %d columns after its pane narrowed", c)
	}

	// Without animations, the zoom shows in the next frame.
	prefs.Animate = false
	a.toggleZoom()
	tt.Frame()
	tt.Frame()
	if z := a.tab().Zoom; z == nil || z.bounds.W < full-1 {
		t.Errorf("the zoomed pane is not full width at once")
	}
}

// TestFollowTab checks the way a tab comes from: from the right when it
// is right of the tab before, or takes the place of one closed, and from
// the left else.
func TestFollowTab(t *testing.T) {
	a := &App{}
	t1, t2, t3 := &Tab{ID: 1}, &Tab{ID: 2}, &Tab{ID: 3}
	a.tabs = []*Tab{t1, t2, t3}
	a.followTab()
	for _, step := range []struct {
		do   func()
		want float32
	}{
		{func() { a.active = 2 }, 1},
		{func() { a.active = 0 }, -1},
		// The first closes: the second takes its place, from the right.
		{func() { a.tabs = a.tabs[1:] }, 1},
		// The last closes, shown: the one left of it comes in.
		{func() { a.active = 1 }, 1},
		{func() { a.tabs = a.tabs[:1]; a.active = 0 }, -1},
	} {
		step.do()
		a.followTab()
		if a.tabDir != step.want {
			t.Errorf("tab %d came from %v, not %v", a.tab().ID, a.tabDir, step.want)
		}
	}
}

// TestMotionSettings turns the animations and the Oh My Posh prompt on
// from the settings, both off until asked for.
func TestMotionSettings(t *testing.T) {
	saved := prefs
	t.Cleanup(func() { prefs = saved })
	prefs = settings{FontSize: defaultFontSize}
	a, tt := newTestApp(t)
	tt.SetSize(1000, 1600) // the whole panel in view
	tt.Frame()
	if prefs.Animate || prefs.OhMyPosh || prefs.prompt() != "" {
		t.Fatalf("on by default: %+v", prefs)
	}
	click := clicker(t, tt)
	click("Settings")
	click("Animate panes and tabs")
	if !prefs.Animate {
		t.Errorf("after the switch: animate %v", prefs.Animate)
	}
	click("Text")
	click("Oh My Posh prompt")
	if !prefs.OhMyPosh || prefs.prompt() != rex.PromptOhMyPosh {
		t.Errorf("after the switch: Oh My Posh %v", prefs.OhMyPosh)
	}
	waitFor(t, tt, "the search for oh-my-posh", func() bool {
		return a.ompAsked && (a.ompPath != "" || tt.HasText("Lex does not find oh-my-posh. Install it (ohmyposh.dev): until a shell finds it on its PATH, the shell keeps its own prompt."))
	})
}
