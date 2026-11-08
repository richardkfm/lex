package main

import (
	"bytes"
	"math"
	"strings"
	"testing"
	"time"

	"github.com/egoist/mygo/ui"
)

// TestAnimations paints each animation at the sizes panes have, smaller
// than the design, its size, and larger, wide and tall: every pixel
// painted, the same frame for the same time, and another a little later.
func TestAnimations(t *testing.T) {
	for _, th := range []*theme{themeByID(defaultDarkTheme), themeByID("aurora-light")} {
		for _, themed := range []bool{false, true} {
			pal := paletteOf(th)
			pal.themed = themed
			for _, an := range animations {
				for _, size := range [][2]int{{8, 4}, {40, 12}, {designCols, designRows}, {72, 23}, {200, 60}, {300, 18}, {30, 70}} {
					g := newGrid(size[0], size[1])
					g.paint(an, animStill, pal)
					for i := 3; i < len(g.img.Pix); i += 4 {
						if g.img.Pix[i] != 255 {
							t.Fatalf("%s at %v leaves pixel %d unpainted", an.id, size, i/4)
						}
					}
					still := append([]byte(nil), g.img.Pix...)
					g.paint(an, animStill, pal)
					if !bytes.Equal(g.img.Pix, still) {
						t.Errorf("%s at %v paints its frame two ways", an.id, size)
					}
				}
				g := newGrid(72, 23)
				g.paint(an, animStill, pal)
				still := append([]byte(nil), g.img.Pix...)
				g.paint(an, animStill+2, pal)
				if bytes.Equal(g.img.Pix, still) {
					t.Errorf("%s is the same two seconds later", an.id)
				}
			}
		}
	}
}

// TestSceneColors paints the places in the colors of the themes: each
// theme's its own, their own when not themed whatever the theme, and as
// light or dark as their own in a light theme too, a color keeping its
// lightness. Helix takes the theme either way.
func TestSceneColors(t *testing.T) {
	frame := func(an *animation, id string, themed bool) []byte {
		pal := paletteOf(themeByID(id))
		pal.themed = themed
		g := newGrid(designCols, designRows)
		g.paint(an, animStill, pal)
		return append([]byte(nil), g.img.Pix...)
	}
	lightness := func(pix []byte) float64 {
		sum := 0.0
		for i := 0; i < len(pix); i += 4 {
			l, _, _ := oklch(rgb{pix[i], pix[i+1], pix[i+2]})
			sum += l
		}
		return sum / float64(len(pix)/4)
	}
	for _, an := range animations {
		natural := frame(an, "aurora", false)
		if an == helixAnim {
			if !bytes.Equal(frame(an, "aurora", true), natural) || bytes.Equal(frame(an, "ember", false), natural) {
				t.Error("helix does not take the theme the same either way")
			}
			continue
		}
		if !bytes.Equal(frame(an, "ember", false), natural) {
			t.Errorf("%s in its own colors changes with the theme", an.id)
		}
		aurora, ember := frame(an, "aurora", true), frame(an, "ember", true)
		if bytes.Equal(aurora, natural) || bytes.Equal(aurora, ember) {
			t.Errorf("%s takes no theme's colors", an.id)
		}
		if l, nl := lightness(frame(an, "paper", true)), lightness(natural); math.Abs(l-nl) > 0.03 {
			t.Errorf("%s is %.2f light in Paper, %.2f in its own colors", an.id, l, nl)
		}
	}
	for _, c := range []string{"#120e2c", "#2e8b57", "#ffcc66", "#3fd9ff", "#f4ecd8", "#8a5cff"} {
		for _, to := range []string{"#077141", "#ff9cc6", "#2f5d9e", "#eadccb"} {
			l, _, _ := oklch(hexRGB(c))
			lt, _, _ := oklch(tone(hexRGB(c), hexRGB(to)))
			if math.Abs(l-lt) > 0.015 {
				t.Errorf("%s in the hue of %s is %.3f light, not %.3f", c, to, lt, l)
			}
		}
	}
}

// BenchmarkAnimations paints a frame of each animation in a large pane.
func BenchmarkAnimations(b *testing.B) {
	pal := paletteOf(themeByID(defaultDarkTheme))
	for _, an := range animations {
		b.Run(an.id, func(b *testing.B) {
			g := newGrid(127, 31)
			for i := range b.N {
				g.paint(an, float64(i)/an.fps, pal)
			}
		})
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
	tt.Type("animation: aurora")
	tt.Key(0, ui.KeyEnter)
	tt.Frame()
	tt.Frame()
	tab := a.tab()
	p := tab.Focus
	if len(tab.panes()) != 2 || p == shell || p.anim != "aurora" || p.SID != "" || p.term != nil {
		t.Fatalf("the palette made %d panes, the focused one %+v", len(tab.panes()), p)
	}
	if name, _ := p.label(); name != "Aurora" {
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
		t.Errorf("after Aurora comes %q", p.anim)
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
