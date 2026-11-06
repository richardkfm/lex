package main

import (
	"image"
	"math"
	"strings"
)

// The backgrounds Lex draws itself, which the settings offer beside a
// picture of the user's: abstract, dark enough for every theme's panes,
// and drawn at the size they are shown, so that they stay sharp.

// patternPrefix marks a background that is one of Lex's patterns, as
// "lex:nebula", in settings.Background, where a picture's path is else.
const patternPrefix = "lex:"

// A pattern is one of Lex's backgrounds: draw paints it w×h pixels, at
// scale pixels for each of a 1920 pixels wide screen's.
type pattern struct {
	id, name string
	draw     func(f *canvas, scale float64)
}

var patterns = []pattern{
	{id: "nebula", name: "Nebula", draw: drawNebula},
	{id: "dunes", name: "Dunes", draw: drawDunes},
	{id: "dots", name: "Dots", draw: drawDots},
	{id: "lines", name: "Lines", draw: drawLines},
}

// patternOf returns the pattern a background names, if it names one.
func patternOf(background string) (pattern, bool) {
	id, ok := strings.CutPrefix(background, patternPrefix)
	if !ok {
		return pattern{}, false
	}
	for _, p := range patterns {
		if p.id == id {
			return p, true
		}
	}
	return pattern{}, false
}

// patternW and patternH are the size a pattern is drawn at for the
// window, which it covers: larger than full HD, for the larger screens.
const patternW, patternH = 2560, 1440

// render paints the pattern w×h pixels.
func (p pattern) render(w, h int, scale float64) *image.RGBA {
	f := newCanvas(w, h)
	p.draw(f, scale)
	return f.rgba()
}

// drawNebula paints glows of violet, teal and plum on near black.
func drawNebula(f *canvas, _ float64) {
	f.fill(rgbOf("#06050a"))
	// The glows, bottom first: each fades out at a share of the distance
	// from its center to the farthest corner, as CSS's radial gradients.
	for _, g := range []struct {
		x, y, reach float64
		col         string
	}{
		{0.62, 0.12, 0.30, "#7a2a52"},
		{0.78, 0.72, 0.48, "#0e5a66"},
		{0.18, 0.28, 0.42, "#4a2380"},
	} {
		cx, cy := g.x*float64(f.w), g.y*float64(f.h)
		far := math.Hypot(max(cx, float64(f.w)-cx), max(cy, float64(f.h)-cy))
		r := g.reach * far
		col := rgbOf(g.col)
		f.each(func(x, y float64) ([3]float64, float64) {
			t := 1 - math.Hypot(x-cx, y-cy)/r
			if t <= 0 {
				return col, 0
			}
			return col, smooth(t)
		})
	}
}

// drawDunes paints a sunset, apricot down to deep plum.
func drawDunes(f *canvas, _ float64) {
	stops := []struct {
		at  float64
		col [3]float64
	}{{0, rgbOf("#f3c58a")}, {0.38, rgbOf("#d9824f")}, {0.66, rgbOf("#8a3b3a")}, {1, rgbOf("#2a1420")}}
	// Along CSS's 172 degrees: nearly straight down.
	a := 172 * math.Pi / 180
	dx, dy := math.Sin(a), -math.Cos(a)
	w, h := float64(f.w), float64(f.h)
	span := math.Abs(w*dx) + math.Abs(h*dy)
	f.each(func(x, y float64) ([3]float64, float64) {
		t := ((x-w/2)*dx+(y-h/2)*dy)/span + 0.5
		for i := 1; i < len(stops); i++ {
			if t <= stops[i].at || i == len(stops)-1 {
				s := min(max((t-stops[i-1].at)/(stops[i].at-stops[i-1].at), 0), 1)
				return lerp3(stops[i-1].col, stops[i].col, s), 1
			}
		}
		return stops[len(stops)-1].col, 1
	})
}

// drawDots paints a fine grid of faint dots on dark slate.
func drawDots(f *canvas, scale float64) {
	slate(f, "#10141c", "#07090d", 160)
	step, in, out := 18*scale, 1.2*scale, 1.8*scale
	white := [3]float64{1, 1, 1}
	f.each(func(x, y float64) ([3]float64, float64) {
		cx := math.Mod(x, step) - step/2
		cy := math.Mod(y, step) - step/2
		d := math.Hypot(cx, cy)
		switch {
		case d <= in:
			return white, 0.28
		case d < out:
			return white, 0.28 * (out - d) / (out - in)
		}
		return white, 0
	})
}

// drawLines paints thin diagonal stripes on dark slate.
func drawLines(f *canvas, scale float64) {
	f.fill(rgbOf("#0d1016"))
	period, width := 14*scale, max(2*scale, 1)
	white := [3]float64{1, 1, 1}
	f.each(func(x, y float64) ([3]float64, float64) {
		p := math.Mod((x+y)*math.Sqrt2/2, period)
		// Covered for width pixels, with a pixel's edge either way.
		cover := min(max(min(p+0.5, width+0.5-p), 0), 1)
		return white, 0.09 * cover
	})
}

// slate fills the canvas with a gradient from one color to another at a
// CSS angle.
func slate(f *canvas, from, to string, deg float64) {
	a, b := rgbOf(from), rgbOf(to)
	r := deg * math.Pi / 180
	dx, dy := math.Sin(r), -math.Cos(r)
	w, h := float64(f.w), float64(f.h)
	span := math.Abs(w*dx) + math.Abs(h*dy)
	f.each(func(x, y float64) ([3]float64, float64) {
		t := min(max(((x-w/2)*dx+(y-h/2)*dy)/span+0.5, 0), 1)
		return lerp3(a, b, t), 1
	})
}

// canvas is a picture being painted, in colors from 0 to 1.
type canvas struct {
	w, h int
	pix  [][3]float64
}

func newCanvas(w, h int) *canvas {
	return &canvas{w: w, h: h, pix: make([][3]float64, w*h)}
}

func (f *canvas) fill(c [3]float64) {
	for i := range f.pix {
		f.pix[i] = c
	}
}

// each lays the color fn gives each pixel, at its center, over it, as
// opaque as the alpha fn gives.
func (f *canvas) each(fn func(x, y float64) ([3]float64, float64)) {
	for y := range f.h {
		for x := range f.w {
			c, a := fn(float64(x)+0.5, float64(y)+0.5)
			if a <= 0 {
				continue
			}
			i := y*f.w + x
			f.pix[i] = lerp3(f.pix[i], c, min(a, 1))
		}
	}
}

// rgba returns the picture, dithered so that its gradients show no
// bands.
func (f *canvas) rgba() *image.RGBA {
	img := image.NewRGBA(image.Rect(0, 0, f.w, f.h))
	// A 4×4 Bayer matrix: thresholds from -0.5 to 0.5 of a step.
	bayer := [16]float64{0, 8, 2, 10, 12, 4, 14, 6, 3, 11, 1, 9, 15, 7, 13, 5}
	for y := range f.h {
		for x := range f.w {
			d := (bayer[(y%4)*4+x%4]+0.5)/16 - 0.5
			c := f.pix[y*f.w+x]
			o := img.PixOffset(x, y)
			for ch := range 3 {
				img.Pix[o+ch] = uint8(min(max(c[ch]*255+d+0.5, 0), 255))
			}
			img.Pix[o+3] = 255
		}
	}
	return img
}

func rgbOf(hex string) [3]float64 {
	var v [3]float64
	for i := range 3 {
		var b byte
		for _, ch := range hex[1+2*i : 3+2*i] {
			b <<= 4
			switch {
			case ch >= '0' && ch <= '9':
				b |= byte(ch - '0')
			case ch >= 'a' && ch <= 'f':
				b |= byte(ch-'a') + 10
			}
		}
		v[i] = float64(b) / 255
	}
	return v
}

func lerp3(a, b [3]float64, t float64) [3]float64 {
	return [3]float64{a[0] + (b[0]-a[0])*t, a[1] + (b[1]-a[1])*t, a[2] + (b[2]-a[2])*t}
}

// smooth eases a glow's fall from 0 to 1.
func smooth(t float64) float64 { return t * t * (3 - 2*t) }
