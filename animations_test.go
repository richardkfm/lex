package main

import (
	"bytes"
	"strings"
	"testing"
	"time"

	"github.com/egoist/mygo/ui"
)

// TestAnimations paints each animation at the sizes panes have: every
// pixel painted, the same frame for the same time, and another a little
// later.
func TestAnimations(t *testing.T) {
	th := themeByID(defaultDarkTheme)
	for _, an := range animations {
		for _, size := range [][2]int{{8, 4}, {40, 12}, {72, 23}, {200, 60}} {
			g := newGrid(size[0], size[1])
			an.draw(g, animStill, paletteOf(th))
			for i := 3; i < len(g.img.Pix); i += 4 {
				if g.img.Pix[i] != 255 {
					t.Fatalf("%s at %v leaves pixel %d unpainted", an.id, size, i/4)
				}
			}
			still := append([]byte(nil), g.img.Pix...)
			g.reset()
			an.draw(g, animStill, paletteOf(th))
			if !bytes.Equal(g.img.Pix, still) {
				t.Errorf("%s at %v paints its frame two ways", an.id, size)
			}
		}
		g := newGrid(72, 23)
		an.draw(g, animStill, paletteOf(th))
		still := append([]byte(nil), g.img.Pix...)
		g.reset()
		an.draw(g, animStill+2, paletteOf(th))
		if bytes.Equal(g.img.Pix, still) {
			t.Errorf("%s is the same two seconds later", an.id)
		}
	}
}

// TestAnimationPane opens an animation from the palette beside the shell:
// a pane of no session, which takes the focus and keeps it, plays, switches
// to the next animation from its header, and comes back with the layout.
func TestAnimationPane(t *testing.T) {
	a, tt := newTestApp(t)
	shell := a.tab().Focus
	waitFor(t, tt, "the shell", func() bool { return shell.term != nil && strings.Contains(shell.term.Text(), testShell.prompt) })
	a.openPalette()
	tt.Frame()
	tt.Type("animation: sahara")
	tt.Key(0, ui.KeyEnter)
	tt.Frame()
	tt.Frame()
	tab := a.tab()
	p := tab.Focus
	if len(tab.panes()) != 2 || p == shell || p.anim != "sahara" || p.SID != "" || p.term != nil {
		t.Fatalf("the palette made %d panes, the focused one %+v", len(tab.panes()), p)
	}
	if name, _ := p.label(); name != "Sahara" {
		t.Errorf("the pane is named %q", name)
	}
	tt.Advance(time.Second)
	tt.Frame()
	if tab.Focus != p {
		t.Fatal("the shell's terminal took the focus back")
	}
	if p.player == nil || p.player.bmp == nil {
		t.Fatal("the animation did not play")
	}
	step := p.player.step
	tt.Advance(time.Second)
	if p.player.step == step {
		t.Error("the animation stands still")
	}
	if err := tt.Click("Next Animation"); err != nil {
		t.Fatal(err)
	}
	if p.anim != animations[0].id {
		t.Errorf("after Sahara comes %q", p.anim)
	}
	a.saveNow()
	shell.term.Close()
	b := &App{client: a.client}
	if !b.restore() {
		t.Fatal("nothing restored")
	}
	var anims []string
	for _, q := range b.tabs[0].panes() {
		if q.anim != "" {
			anims = append(anims, q.anim)
		}
	}
	if len(anims) != 1 || anims[0] != animations[0].id {
		t.Errorf("restored animations %q", anims)
	}
}
