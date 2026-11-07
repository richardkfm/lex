package main

import (
	"bytes"
	"testing"
	"time"

	"github.com/egoist/mygo/ui"
)

// TestScenes draws each scene: its still frame at its own size, the same
// frame for the same time, and another a step later, as it plays.
func TestScenes(t *testing.T) {
	n := 0
	for _, p := range patterns {
		if p.scene == nil {
			continue
		}
		n++
		still := p.render(0, 0, 0)
		if b := still.Bounds(); b.Dx() != sceneW || b.Dy() != sceneH {
			t.Errorf("%s is %v", p.id, b)
		}
		again := p.scene.render(sceneStill)
		if !bytes.Equal(still.Pix, again.Pix) {
			t.Errorf("%s draws its still frame two ways", p.id)
		}
		later := p.scene.render(sceneStill + 1.0/sceneFPS)
		changed := 0
		for i := 0; i < len(later.Pix); i += 4 {
			if later.Pix[i] != still.Pix[i] || later.Pix[i+1] != still.Pix[i+1] || later.Pix[i+2] != still.Pix[i+2] {
				changed++
			}
		}
		if changed < 20 {
			t.Errorf("%s changes %d pixels in a step", p.id, changed)
		}
	}
	if n != 2 {
		t.Errorf("%d scenes", n)
	}
}

// TestDrift keeps the drifting background over the whole window: it pans
// no farther than its zoom leaves room for.
func TestDrift(t *testing.T) {
	r := ui.Rect{W: 1600, H: 900}
	if z, dx, dy := driftAt(0, r); z != 1 || dx != 0 || dy != 0 {
		t.Errorf("the drift starts at zoom %v, moved by %v, %v", z, dx, dy)
	}
	moved := false
	for d := time.Duration(0); d < 10*time.Minute; d += 700 * time.Millisecond {
		z, dx, dy := driftAt(d, r)
		if abs(dx) > r.W*(z-1)/2 || abs(dy) > r.H*(z-1)/2 {
			t.Fatalf("at %v the background, %v as large, moved by %v, %v: an edge shows", d, z, dx, dy)
		}
		moved = moved || abs(dx) > 20 && abs(dy) > 10
	}
	if !moved {
		t.Error("the background hardly moves")
	}
}

// TestMovingBackground plays the Rainy City while the settings move the
// background, and keeps it still otherwise.
func TestMovingBackground(t *testing.T) {
	saved := prefs
	t.Cleanup(func() { prefs, moveOnCPU = saved, false })
	prefs = settings{FontSize: defaultFontSize, Background: "lex:rain"}
	moveOnCPU = true
	a, tt := newTestApp(t)
	waitFor(t, tt, "the background", func() bool { return a.bg != nil && a.bg.bmp != nil })
	tt.Advance(time.Second)
	if a.drift != 0 || a.scene != nil {
		t.Fatalf("the background moved by itself: %v", a.drift)
	}
	prefs.MovingBackground = true
	tt.Frame()
	tt.Advance(time.Second)
	if a.drift < 900*time.Millisecond || a.scene == nil || a.scene.step < int64((sceneStill+0.9)*sceneFPS) {
		t.Fatalf("after a second of moving, the drift is %v and the scene %+v", a.drift, a.scene)
	}
	// An art moves too, its steps painted off the main thread.
	prefs.Background = "lex:ribbons"
	tt.Frame()
	start := int64(artTime(a.drift) * ribbons.fps)
	for i := 0; i < 50 && (a.art == nil || a.art.step < start+5); i++ {
		time.Sleep(10 * time.Millisecond)
		tt.Advance(100 * time.Millisecond)
	}
	if a.art == nil || a.art.step < start+5 {
		t.Errorf("Ribbons did not move: %+v", a.art)
	}
	// Stopped, it stays where it was.
	prefs.MovingBackground = false
	tt.Frame()
	d := a.drift
	tt.Advance(time.Second)
	if a.drift != d {
		t.Errorf("the drift went on from %v to %v, stopped", d, a.drift)
	}
}

// TestArts paints the abstract backgrounds in a dark and a light theme:
// in the theme's colors, the same for the same time, another a step
// later, and painted again only when the step, the theme or the size
// changes.
func TestArts(t *testing.T) {
	dark, light := themeByID("aurora"), themeByID("aurora-light")
	n := 0
	for _, p := range patterns {
		if p.art == nil {
			continue
		}
		n++
		var v artView
		a := v.bitmap(p.art, p.id, 640, 360, dark, artStill, 0)
		still := append([]byte(nil), v.bufs[v.cur].img.Pix...)
		if v.bitmap(p.art, p.id, 640, 360, dark, artStill+0.1/p.art.fps, 0) != a || v.step != int64(artStill*p.art.fps) {
			t.Errorf("%s painted again within a step", p.id)
		}
		v.bitmap(p.art, p.id, 640, 360, dark, artStill+2, 0)
		if bytes.Equal(v.bufs[v.cur].img.Pix, still) {
			t.Errorf("%s is the same two seconds later", p.id)
		}
		// The light theme's: lighter on average than the dark one's.
		var w artView
		w.bitmap(p.art, p.id, 640, 360, light, artStill, 0)
		if lum(w.bufs[w.cur].img.Pix) < lum(still)+100 {
			t.Errorf("%s in Aurora Light is as dark as in Aurora", p.id)
		}
		var again artView
		again.bitmap(p.art, p.id, 640, 360, dark, artStill, 0)
		if !bytes.Equal(again.bufs[again.cur].img.Pix, still) {
			t.Errorf("%s paints its still frame two ways", p.id)
		}
	}
	if n != 2 {
		t.Errorf("%d arts", n)
	}
}

// lum is the mean of the red, green and blue of pixels, from 0 to 255.
func lum(pix []byte) float64 {
	sum := 0
	for i := 0; i < len(pix); i += 4 {
		sum += int(pix[i]) + int(pix[i+1]) + int(pix[i+2])
	}
	return float64(sum) / float64(len(pix)/4*3)
}
