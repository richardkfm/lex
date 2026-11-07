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

// TestMovingBackground plays Campfire while the settings move the
// background, and keeps it still otherwise.
func TestMovingBackground(t *testing.T) {
	saved := prefs
	t.Cleanup(func() { prefs, moveOnCPU = saved, false })
	prefs = settings{FontSize: defaultFontSize, Background: "lex:campfire"}
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
	// Stopped, it stays where it was.
	prefs.MovingBackground = false
	tt.Frame()
	d := a.drift
	tt.Advance(time.Second)
	if a.drift != d {
		t.Errorf("the drift went on from %v to %v, stopped", d, a.drift)
	}
}
