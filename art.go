package main

import (
	"image"
	"math"
	"time"

	"github.com/egoist/mygo/ui"
)

// Lex's abstract backgrounds: painted in the colors of the theme the
// window shows, so that they change with it, dark or light. They move
// while the settings move the background (settings.MovingBackground), and
// show a still frame else. They are painted in memory, as the scenes are,
// and shown as a picture: painted again only when they change (a step of
// fps a second, the theme, the window's size or the blur). The GPU would
// draw their many long lines as masks in the atlas of the text, which so
// large and changing shapes would crowd.

// An art is one of Lex's abstract backgrounds: base paints what stays
// still, once for a theme and a size, and frame what moves, at t seconds,
// over a copy of it. res is the share of the window's pixels it is painted
// at, the most it is painted at, maxW.
type art struct {
	base  func(f raster, pal artPalette)
	frame func(f raster, t float64, pal artPalette)
	fps   float64
	res   float64
	maxW  int
}

// artStill is the time of the frame shown while nothing moves.
const artStill = 40.0

// artTime is how far into its motion an art is, d into the background's.
func artTime(d time.Duration) float64 { return artStill + d.Seconds() }

// artPalette are the colors of a theme an art paints with.
type artPalette struct {
	bg, accent, second, third rgb
	dark                      bool
}

// paletteOf returns the colors of theme t for the arts.
func paletteOf(t *theme) artPalette {
	return artPalette{bg: hexRGB(t.bg), accent: hexRGB(t.accent), second: hexRGB(t.second), third: hexRGB(t.pal[4]), dark: t.dark}
}

// raster is a picture being painted, of any size, as pixels is a scene's.
type raster struct {
	img  *image.RGBA
	w, h int
}

func newRaster(w, h int) raster {
	return raster{img: image.NewRGBA(image.Rect(0, 0, w, h)), w: w, h: h}
}

func (f raster) fill(c rgb) {
	for i := 0; i < len(f.img.Pix); i += 4 {
		f.img.Pix[i], f.img.Pix[i+1], f.img.Pix[i+2], f.img.Pix[i+3] = c[0], c[1], c[2], 255
	}
}

// blend lays c over the pixel at (x, y), as opaque as a, dithered, so that
// faint layers of light show no bands.
func (f raster) blend(x, y int, c rgb, a float64) {
	if x < 0 || y < 0 || x >= f.w || y >= f.h || a <= 0 {
		return
	}
	a = min(a, 1)
	d := bayer4(x, y) - 0.5
	o := (y*f.w + x) * 4
	for i := range 3 {
		v := float64(f.img.Pix[o+i])
		f.img.Pix[o+i] = uint8(min(max(v+(float64(c[i])-v)*a+d, 0), 255))
	}
}

// glow lays a soft light of c round (cx, cy), reaching r, at most a.
func (f raster) glow(cx, cy, r float64, c rgb, a float64) {
	for y := max(int(cy-r), 0); y < min(int(cy+r)+1, f.h); y++ {
		for x := max(int(cx-r), 0); x < min(int(cx+r)+1, f.w); x++ {
			d := math.Hypot(float64(x)-cx, float64(y)-cy) / r
			if d < 1 {
				f.blend(x, y, c, a*smooth(1-d))
			}
		}
	}
}

// line draws an antialiased line a pixel wide from (x0, y0) to (x1, y1),
// as opaque as a, as Xiaolin Wu's algorithm does.
func (f raster) line(x0, y0, x1, y1 float64, c rgb, a float64) {
	steep := math.Abs(y1-y0) > math.Abs(x1-x0)
	if steep {
		x0, y0, x1, y1 = y0, x0, y1, x1
	}
	if x0 > x1 {
		x0, y0, x1, y1 = x1, y1, x0, y0
	}
	grad := 1.0
	if dx := x1 - x0; dx > 0 {
		grad = (y1 - y0) / dx
	}
	plot := func(x, y int, k float64) {
		if steep {
			f.blend(y, x, c, a*k)
		} else {
			f.blend(x, y, c, a*k)
		}
	}
	y := y0 + grad*(math.Round(x0)-x0)
	for x := int(math.Round(x0)); x <= int(math.Round(x1)); x++ {
		fy := math.Floor(y)
		plot(x, int(fy), 1-(y-fy))
		plot(x, int(fy)+1, y-fy)
		y += grad
	}
}

// artView paints an art and keeps the bitmap it shows as: the still parts
// painted once, the moving ones over a copy of them at each step, in two
// buffers by turns, as a bitmap takes its picture without a copy. With
// async, it paints the steps after the first off the main thread, which
// keeps drawing the window meanwhile, and shows each once it is painted.
type artView struct {
	async bool
	key   artKey
	base  raster
	bufs  [2]raster
	cur   int
	step  int64
	bmp   *ui.Bitmap
	// gen counts the keys, so that a step painted for another is dropped;
	// busy tells that one is being painted, which done brings.
	gen  int
	busy bool
	done chan artFrame
}

// artFrame is a step of an art painted off the main thread.
type artFrame struct {
	gen, buf int
	img      *image.RGBA
}

// artKey is what the still parts of an art are painted for.
type artKey struct {
	id, theme string
	w, h      int
	blur      int
}

// bitmap returns the bitmap of art a, of the pattern id, for a window
// w×h pixels, at t seconds, in theme th, blurred by blur DIPs.
func (v *artView) bitmap(a *art, id string, w, h int, th *theme, t float64, blur int) *ui.Bitmap {
	k := min(a.res, float64(a.maxW)/float64(max(w, 1)))
	w, h = max(int(float64(w)*k), 16), max(int(float64(h)*k), 9)
	pal := paletteOf(th)
	key := artKey{id: id, theme: th.id, w: w, h: h, blur: blur}
	if v.key != key {
		v.key, v.step, v.bmp, v.busy = key, -1, nil, false
		v.gen++
		v.base = newRaster(w, h)
		a.base(v.base, pal)
		v.bufs = [2]raster{newRaster(w, h), newRaster(w, h)}
	}
	if v.busy {
		select {
		case f := <-v.done:
			v.busy = false
			if f.gen == v.gen {
				v.show(f.buf, f.img)
			}
		default:
		}
	}
	step := int64(t * a.fps)
	if step == v.step && v.bmp != nil || v.busy {
		return v.bmp
	}
	v.step = step
	next, base, gen := v.cur^1, v.base, v.gen
	paint := func() *image.RGBA {
		f := v.bufs[next]
		copy(f.img.Pix, base.img.Pix)
		a.frame(f, float64(step)/a.fps, pal)
		if blur > 0 {
			// A DIP of the window is k of the art's pixels.
			return blurred(f.img, float64(blur)*k)
		}
		return f.img
	}
	if !v.async || v.bmp == nil {
		v.show(next, paint())
		return v.bmp
	}
	if v.done == nil {
		v.done = make(chan artFrame, 2)
	}
	v.busy = true
	go func() { v.done <- artFrame{gen: gen, buf: next, img: paint()} }()
	return v.bmp
}

// show shows the picture painted in buffer buf.
func (v *artView) show(buf int, img *image.RGBA) {
	v.cur = buf
	if v.bmp == nil {
		v.bmp = ui.NewBitmap(img)
	} else {
		v.bmp.Update(img)
	}
}

// The Ribbons: bundles of fine strands, as silk, waving slowly across the
// window and twisting, in the second color, the accent and the theme's
// blue, over the window's color with a faint light of each.
var ribbons = &art{fps: 20, res: 0.5, maxW: 960,
	base: func(f raster, pal artPalette) {
		f.fill(pal.bg)
		w, h := float64(f.w), float64(f.h)
		a := 0.14
		if !pal.dark {
			a = 0.10
		}
		f.glow(w*0.12, h*0.08, w*0.6, pal.second, a)
		f.glow(w*0.92, h*0.95, w*0.5, pal.accent, a*0.7)
	},
	frame: func(f raster, t float64, pal artPalette) {
		w, h := float64(f.w), float64(f.h)
		u := w / 1920
		for i, rb := range [3]struct {
			col                   rgb
			y, amp, f1, f2, f3    float64
			speed, spread, offset float64
			strands               int
			strength              float64
		}{
			{pal.second, 0.34, 0.12, 0.7, 1.6, 0.9, 0.11, 0.09, 0.0, 26, 1},
			{pal.accent, 0.56, 0.10, 0.9, 1.9, 0.6, 0.08, 0.07, 2.1, 22, 0.9},
			{pal.third, 0.74, 0.08, 0.5, 1.3, 1.1, 0.06, 0.06, 4.3, 18, 0.7},
		} {
			phase := t * rb.speed * 2 * math.Pi / 6
			// The glow under the ribbon, then its strands, column by
			// column: each strand crosses a column once.
			sigma := 34 * u
			glowA := 0.10 * rb.strength
			if !pal.dark {
				glowA = 0.08 * rb.strength
			}
			// The glow's weights, by how far a row is from the ribbon's
			// middle, in whole pixels.
			reach := int(2.5 * sigma)
			weights := make([]float64, 2*reach+1)
			for d := -reach; d <= reach; d++ {
				k := float64(d) / sigma
				weights[d+reach] = glowA * math.Exp(-k*k)
			}
			for x := 0; x < f.w; x++ {
				X := (float64(x) + 0.5) / w
				cy := h * (rb.y + rb.amp*(0.62*math.Sin(X*2*math.Pi*rb.f1+phase+rb.offset)+
					0.38*math.Sin(X*2*math.Pi*rb.f2-phase*0.7+rb.offset*1.7)))
				tw := math.Sin(X*2*math.Pi*rb.f3 + phase*1.15 + rb.offset)
				wave := h * 0.02 * math.Sin(X*9+phase+float64(i))
				mid := int(math.Round(cy))
				for d, wt := range weights {
					f.blend(x, mid+d-reach, rb.col, wt)
				}
				for s := range rb.strands {
					o := float64(s)/float64(rb.strands-1) - 0.5
					y := cy + o*rb.spread*h*tw + o*o*wave
					mid := 1 - math.Abs(o)*2
					a := (0.10 + 0.40*mid*mid) * rb.strength
					if !pal.dark {
						a *= 1.1
					}
					fy := math.Floor(y)
					f.blend(x, int(fy), rb.col, a*(1-(y-fy)))
					f.blend(x, int(fy)+1, rb.col, a*(y-fy))
				}
			}
		}
	},
}

// The Contours: the lines of a map of heights that rise and fall slowly,
// as dunes shifting, the low ones in the accent and the high ones in the
// second color, every fourth line stronger, over the window's color.
var contours = &art{fps: 12, res: 1, maxW: 1920,
	base: func(f raster, pal artPalette) {
		f.fill(pal.bg)
		f.glow(float64(f.w)*0.5, float64(f.h)*0.55, float64(f.w)*0.7, pal.accent, 0.05)
	},
	frame: func(f raster, t float64, pal artPalette) {
		w := float64(f.w)
		u := w / 1920
		// The heights, on a grid of cells about 18 pixels of a full HD
		// window wide: two octaves of noise drifting through time.
		cols := max(int(w/(18*u)), 24)
		cell := w / float64(cols)
		rows := int(float64(f.h)/cell) + 2
		scale := float64(cols) / 7
		z := t * 0.035
		field := make([]float64, (cols+1)*(rows+1))
		for j := 0; j <= rows; j++ {
			for i := 0; i <= cols; i++ {
				x, y := float64(i)/scale+t*0.012, float64(j)/scale
				field[j*(cols+1)+i] = 0.68*noise3(x, y, z, 90) + 0.32*noise3(x*2.3, y*2.3, z*1.6, 91)
			}
		}
		at := func(i, j int) float64 { return field[j*(cols+1)+i] }
		const levels = 15
		for l := range levels {
			level := 0.2 + 0.6*float64(l)/float64(levels-1)
			k := float64(l) / float64(levels-1)
			col := rgb{
				uint8(float64(pal.accent[0]) + (float64(pal.second[0])-float64(pal.accent[0]))*k),
				uint8(float64(pal.accent[1]) + (float64(pal.second[1])-float64(pal.accent[1]))*k),
				uint8(float64(pal.accent[2]) + (float64(pal.second[2])-float64(pal.accent[2]))*k),
			}
			alpha, strong := 0.22, l%4 == 1
			if strong {
				alpha = 0.5
			}
			if !pal.dark {
				alpha *= 1.3
			}
			edge := func(i, j, side int) (float64, float64) {
				// Where the level crosses side of cell (i, j): top, right,
				// bottom or left.
				x0, y0, x1, y1 := i, j, i+1, j
				switch side {
				case 1:
					x0, y0, x1, y1 = i+1, j, i+1, j+1
				case 2:
					x0, y0, x1, y1 = i, j+1, i+1, j+1
				case 3:
					x0, y0, x1, y1 = i, j, i, j+1
				}
				a, b := at(x0, y0), at(x1, y1)
				s := 0.5
				if a != b {
					s = (level - a) / (b - a)
				}
				return (float64(x0) + float64(x1-x0)*s) * cell, (float64(y0) + float64(y1-y0)*s) * cell
			}
			for j := range rows {
				for i := range cols {
					c := 0
					if at(i, j) > level {
						c |= 8
					}
					if at(i+1, j) > level {
						c |= 4
					}
					if at(i+1, j+1) > level {
						c |= 2
					}
					if at(i, j+1) > level {
						c |= 1
					}
					for _, seg := range marching[c] {
						x0, y0 := edge(i, j, seg[0])
						x1, y1 := edge(i, j, seg[1])
						f.line(x0, y0, x1, y1, col, alpha)
						if strong {
							// Twice as wide: again, half a pixel across.
							f.line(x0+0.5*u, y0+0.5*u, x1+0.5*u, y1+0.5*u, col, alpha*0.8)
						}
					}
				}
			}
		}
	},
}

// marching are the segments of a contour through a cell, by which of its
// corners are above the level (top left 8, top right 4, bottom right 2,
// bottom left 1): pairs of the sides they join, 0 top, 1 right, 2
// bottom, 3 left.
var marching = [16][][2]int{
	{}, {{3, 2}}, {{2, 1}}, {{3, 1}},
	{{0, 1}}, {{3, 0}, {2, 1}}, {{0, 2}}, {{3, 0}},
	{{3, 0}}, {{0, 2}}, {{3, 2}, {0, 1}}, {{0, 1}},
	{{3, 1}}, {{2, 1}}, {{3, 2}}, {},
}

// noise3 is smooth value noise in three dimensions, from 0 to 1.
func noise3(x, y, z float64, seed int) float64 {
	x0, y0, z0 := math.Floor(x), math.Floor(y), math.Floor(z)
	ix, iy, iz := int(x0), int(y0), int(z0)
	sx, sy, sz := smooth(x-x0), smooth(y-y0), smooth(z-z0)
	layer := func(z int) float64 {
		s := seed*7919 + z*104729
		a, b := hash(ix, iy, s), hash(ix+1, iy, s)
		c, d := hash(ix, iy+1, s), hash(ix+1, iy+1, s)
		top, bottom := a+(b-a)*sx, c+(d-c)*sx
		return top + (bottom-top)*sy
	}
	a, b := layer(iz), layer(iz+1)
	return a + (b-a)*sz
}
