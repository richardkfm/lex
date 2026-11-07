package main

import (
	"image"
	"math"
	"time"

	"github.com/egoist/mygo/ui"
	xdraw "golang.org/x/image/draw"
)

// Lex's scenes: backgrounds of pixel art, sceneW×sceneH pixels shown as
// squares, which play while the settings move the background
// (settings.MovingBackground) and show a still frame else. A scene draws
// its still parts once (base) and its moving parts over a copy of them at
// each of its steps (frame), from the time alone: any frame draws at
// once, and a scene that stops goes on from where it was.

const (
	sceneW, sceneH = 320, 180
	// sceneFPS is how many frames a second a scene steps at.
	sceneFPS = 20
	// sceneStill is the time of the frame shown while nothing moves.
	sceneStill = 3.2
)

// A scene is one of Lex's animated backgrounds.
type scene struct {
	base  func(f pixels)
	frame func(f pixels, t float64)
}

// render returns the scene's frame at t seconds.
func (s *scene) render(t float64) *image.RGBA {
	base := image.NewRGBA(image.Rect(0, 0, sceneW, sceneH))
	s.base(pixels{base})
	img := image.NewRGBA(base.Rect)
	s.draw(img, base, t)
	return img
}

// draw paints the frame at t into img, over a copy of base.
func (s *scene) draw(img, base *image.RGBA, t float64) {
	copy(img.Pix, base.Pix)
	s.frame(pixels{img}, t)
}

// blurredScene returns a scene's frame blurred by blur DIPs of a window
// as wide as a full HD screen: drawn twice as large first, as so few
// pixels would blur in steps.
func blurredScene(img *image.RGBA, blur int) *image.RGBA {
	b := img.Bounds()
	big := image.NewRGBA(image.Rect(0, 0, b.Dx()*2, b.Dy()*2))
	xdraw.NearestNeighbor.Scale(big, big.Bounds(), img, b, xdraw.Src, nil)
	return blurred(big, float64(blur)*float64(big.Bounds().Dx())/1920)
}

// sceneView plays a scene: the bitmap the window shows, updated at each
// step, in two buffers by turns, as a bitmap takes its picture without a
// copy.
type sceneView struct {
	id   string
	blur int
	base *image.RGBA
	bufs [2]*image.RGBA
	cur  int
	step int64
	bmp  *ui.Bitmap
}

// sceneBitmap returns the bitmap of the scene p at d into its play.
func (a *App) sceneBitmap(p pattern, d time.Duration) *ui.Bitmap {
	blur := prefs.backgroundBlur()
	v := a.scene
	if v == nil || v.id != p.id || v.blur != blur {
		v = &sceneView{id: p.id, blur: blur, step: -1}
		v.base = image.NewRGBA(image.Rect(0, 0, sceneW, sceneH))
		p.scene.base(pixels{v.base})
		for i := range v.bufs {
			v.bufs[i] = image.NewRGBA(v.base.Rect)
		}
		a.scene = v
	}
	// The play starts at the still frame, so that the scene comes alive
	// from it.
	t := sceneStill + d.Seconds()
	step := int64(t * sceneFPS)
	if step == v.step {
		return v.bmp
	}
	v.step = step
	v.cur ^= 1
	img := v.bufs[v.cur]
	p.scene.draw(img, v.base, float64(step)/sceneFPS)
	shown := img
	if blur > 0 {
		shown = blurredScene(img, blur)
	}
	if v.bmp == nil {
		v.bmp = ui.NewBitmap(shown)
		if blur == 0 {
			v.bmp.Pixelated()
		}
	} else {
		v.bmp.Update(shown)
	}
	return v.bmp
}

// pixels is a frame of a scene being painted.
type pixels struct{ img *image.RGBA }

// rgb is a color of a scene's palette.
type rgb [3]uint8

// hexRGB returns the color "#rrggbb".
func hexRGB(s string) rgb {
	c := rgbOf(s)
	return rgb{uint8(c[0]*255 + 0.5), uint8(c[1]*255 + 0.5), uint8(c[2]*255 + 0.5)}
}

func (f pixels) in(x, y int) bool { return x >= 0 && y >= 0 && x < sceneW && y < sceneH }

func (f pixels) set(x, y int, c rgb) {
	if !f.in(x, y) {
		return
	}
	o := f.img.PixOffset(x, y)
	f.img.Pix[o], f.img.Pix[o+1], f.img.Pix[o+2], f.img.Pix[o+3] = c[0], c[1], c[2], 255
}

func (f pixels) at(x, y int) rgb {
	o := f.img.PixOffset(x, y)
	return rgb{f.img.Pix[o], f.img.Pix[o+1], f.img.Pix[o+2]}
}

// mix lays c over the pixel at (x, y), as opaque as a.
func (f pixels) mix(x, y int, c rgb, a float64) {
	if !f.in(x, y) || a <= 0 {
		return
	}
	a = min(a, 1)
	o := f.img.PixOffset(x, y)
	for i := range 3 {
		v := float64(f.img.Pix[o+i])
		f.img.Pix[o+i] = uint8(v + (float64(c[i])-v)*a + 0.5)
	}
}

// dither sets the pixel at (x, y) to c where a passes the threshold of
// its place in a 4×4 Bayer matrix: a share a of the pixels of an area
// take it, evenly spread, as pixel art shades.
func (f pixels) dither(x, y int, c rgb, a float64) {
	if a > bayer4(x, y) {
		f.set(x, y, c)
	}
}

// bayer4 is the threshold of (x, y) in a 4×4 Bayer matrix, from 0 to 1.
func bayer4(x, y int) float64 {
	m := [16]float64{0, 8, 2, 10, 12, 4, 14, 6, 3, 11, 1, 9, 15, 7, 13, 5}
	return (m[(y&3)*4+x&3] + 0.5) / 16
}

// gradient paints rows top to bottom (excluded) in bands of the colors
// of stops, dithered from one to the next.
func (f pixels) gradient(top, bottom int, stops ...rgb) {
	n := len(stops) - 1
	for y := top; y < bottom; y++ {
		t := float64(y-top) / float64(max(bottom-top-1, 1)) * float64(n)
		i := min(int(t), n-1)
		frac := t - float64(i)
		for x := range sceneW {
			f.set(x, y, stops[i])
			f.dither(x, y, stops[i+1], frac)
		}
	}
}

// disc paints a disc of radius r at (cx, cy).
func (f pixels) disc(cx, cy, r float64, c rgb) {
	for y := int(cy - r - 1); y <= int(cy+r+1); y++ {
		for x := int(cx - r - 1); x <= int(cx+r+1); x++ {
			if math.Hypot(float64(x)+0.5-cx, float64(y)+0.5-cy) <= r {
				f.set(x, y, c)
			}
		}
	}
}

// glow lights the pixels round (cx, cy), out to rx across and ry down, in
// dithered rings of the colors of levels, the brightest in the middle;
// where tells which pixels it lights.
func (f pixels) glow(cx, cy, rx, ry, strength float64, levels []rgb, where func(x, y int) bool) {
	for y := int(cy - ry); y <= int(cy+ry); y++ {
		for x := int(cx - rx); x <= int(cx+rx); x++ {
			if !f.in(x, y) || where != nil && !where(x, y) {
				continue
			}
			d := math.Hypot((float64(x)-cx)/rx, (float64(y)-cy)/ry)
			if d >= 1 {
				continue
			}
			v := (1 - d) * (1 - d) * strength * float64(len(levels))
			i := int(v)
			if i > 0 {
				f.set(x, y, levels[min(i, len(levels))-1])
			}
			if i < len(levels) {
				f.dither(x, y, levels[i], v-float64(i))
			}
		}
	}
}

// pine paints a pine of height h standing at (x, base): tiers that widen
// downward, each starting narrower than the one above ended, on a short
// trunk, with snow on the tiers when snow is set.
func (f pixels) pine(x, base, h int, body rgb, snow *rgb) {
	top := base - h
	tiers := max(3, h/11)
	tierH := float64(h-3) / float64(tiers)
	for y := top; y < base-2; y++ {
		d := float64(y - top)
		at := math.Mod(d, tierH) / tierH
		k := math.Floor(d / tierH)
		hw := 0.6 + (k*0.55+0.35+0.75*at)*tierH*0.42
		w := int(hw)
		for dx := -w; dx <= w; dx++ {
			f.set(x+dx, y, body)
		}
		if snow != nil && at < 1/tierH*1.5 && k > 0 {
			// The top of a tier, where it sticks out of the one above.
			for dx := -w; dx <= w; dx++ {
				if abs(float32(dx)) > float32(w)*0.45 || y == top {
					f.set(x+dx, y, *snow)
				}
			}
		}
	}
	if snow != nil {
		f.set(x, top, *snow)
	}
	for y := base - 3; y < base; y++ {
		f.set(x, y, body)
		f.set(x+1, y, body)
	}
}

// hash returns a number from 0 to 1 of the integers given, the same each
// time.
func hash(x, y, seed int) float64 {
	h := uint32(x)*374761393 + uint32(y)*668265263 + uint32(seed)*2246822519
	h = (h ^ h>>13) * 1274126177
	h ^= h >> 16
	return float64(h) / math.MaxUint32
}

// noise is smooth value noise, from 0 to 1.
func noise(x, y float64, seed int) float64 {
	x0, y0 := math.Floor(x), math.Floor(y)
	ix, iy := int(x0), int(y0)
	sx, sy := smooth(x-x0), smooth(y-y0)
	a, b := hash(ix, iy, seed), hash(ix+1, iy, seed)
	c, d := hash(ix, iy+1, seed), hash(ix+1, iy+1, seed)
	top, bottom := a+(b-a)*sx, c+(d-c)*sx
	return top + (bottom-top)*sy
}

// star is a star of a scene's sky; those that twinkle the frames draw.
type star struct {
	x, y     int
	bright   float64
	twinkle  bool
	rate, at float64
}

// stars returns the stars of a sky above sky, of a seed's own.
func stars(sky int, density float64, seed int) []star {
	var out []star
	for y := range sky {
		for x := range sceneW {
			if hash(x, y, seed) < 1-density {
				continue
			}
			out = append(out, star{x: x, y: y, bright: 0.35 + 0.65*hash(x, y, seed+1),
				twinkle: hash(x, y, seed+2) > 0.55, rate: 1.5 + 2.5*hash(x, y, seed+3), at: 6.3 * hash(x, y, seed+4)})
		}
	}
	return out
}

// drawStars paints the stars at t: those that twinkle only in the frames.
func (f pixels) drawStars(list []star, dim, bright rgb, t float64, frame bool) {
	for _, s := range list {
		if s.twinkle != frame {
			continue
		}
		b := s.bright
		if s.twinkle {
			b *= 0.5 + 0.5*math.Sin(t*s.rate+s.at)
		}
		switch {
		case b > 0.7:
			f.set(s.x, s.y, bright)
		case b > 0.3:
			f.set(s.x, s.y, dim)
		}
	}
}

// fall is where a particle that falls, as a snowflake, is at t: it falls
// v pixels a second from y0 and sways amp to either side of x0 as wind
// carries it.
func fall(x0, y0, v, amp, wind, rate, at, t float64) (int, int) {
	y := math.Mod(y0+t*v, sceneH+6) - 3
	x := math.Mod(x0+t*wind+amp*math.Sin(t*rate+at)+sceneW*8, sceneW)
	return int(x), int(y)
}

// The Snowfall: a cabin among pines on a snowy night, its window lit and
// its chimney smoking, under the moon, snow falling before it.

var snowfall = func() *scene {
	sky := []rgb{hexRGB("#060c1e"), hexRGB("#0b1630"), hexRGB("#122243"), hexRGB("#1c3158"), hexRGB("#29416a")}
	mountain, peakSnow := hexRGB("#1b2b4b"), hexRGB("#7f96b8")
	farPine, nearPine := hexRGB("#132139"), hexRGB("#0b1424")
	snowLit, snowMid, snowShade := hexRGB("#d3deec"), hexRGB("#adbdd3"), hexRGB("#8698b4")
	pineSnow, farSnow := hexRGB("#c9d6e7"), hexRGB("#5f7699")
	moon, crater := hexRGB("#e9eef7"), hexRGB("#c4cee0")
	halo := []rgb{hexRGB("#2f4772"), hexRGB("#3d5683")}
	logA, logB, dark := hexRGB("#4a3327"), hexRGB("#3b281f"), hexRGB("#1f150f")
	roof, brick := hexRGB("#3a2b2d"), hexRGB("#55443f")
	lamp, lampHot := hexRGB("#ffc35c"), hexRGB("#ffe29a")
	spill := hexRGB("#d8b98a")
	smoke := hexRGB("#7f8ca3")
	ridge := func(x int) int {
		return 90 + int(18*noise(float64(x)/46, 0, 40)+6*noise(float64(x)/13, 0, 41))
	}
	groundTop := func(x int) int { return 134 + int(3*noise(float64(x)/26, 3, 42)) }
	skyStars := stars(96, 0.005, 50)
	type flake struct{ x, y, v, amp, rate, at float64 }
	var flakes [3][]flake
	for layer, n := range [3]int{80, 60, 30} {
		for i := range n {
			h := func(k int) float64 { return hash(i, k, 60+layer*7) }
			v := [3]float64{7, 14, 26}[layer] + [3]float64{4, 6, 9}[layer]*h(1)
			flakes[layer] = append(flakes[layer], flake{x: sceneW * h(2), y: sceneH * h(3), v: v,
				amp: [3]float64{1.5, 3, 5}[layer] * (0.5 + h(4)), rate: 0.6 + 1.2*h(5), at: 6.3 * h(6)})
		}
	}
	flakeColor := [3]rgb{hexRGB("#7f92b0"), hexRGB("#c3d0e3"), hexRGB("#ffffff")}
	const (
		cabinX, cabinW = 108, 46 // its walls, from the left
		eaves          = 127     // the foot of the roof
		floor          = 147     // the foot of the walls
	)
	return &scene{
		base: func(f pixels) {
			f.gradient(0, 138, sky...)
			f.drawStars(skyStars, hexRGB("#566c96"), hexRGB("#dfe8f7"), 0, false)
			f.glow(254, 36, 26, 26, 0.8, halo, nil)
			f.disc(254, 36, 10, moon)
			f.disc(250, 33, 2.2, crater)
			f.disc(258, 40, 1.6, crater)
			f.disc(256, 31, 1.1, crater)
			for x := range sceneW {
				top := ridge(x)
				for y := top; y < sceneH; y++ {
					f.set(x, y, mountain)
					// Snow on the peaks, thinning down their sides.
					if top < 100 {
						f.dither(x, y, peakSnow, (1-float64(y-top)/7)*float64(100-top)/8)
					}
				}
			}
			for i := range 46 {
				x := i*7 + int(5*hash(i, 1, 43))
				h := 9 + int(9*hash(i, 2, 43))
				fs := farSnow
				f.pine(x, 131+int(3*hash(i, 3, 43)), h, farPine, &fs)
			}
			for x := range sceneW {
				gt := groundTop(x)
				for y := gt; y < sceneH; y++ {
					f.set(x, y, snowMid)
					t := float64(y-gt) / float64(sceneH-gt)
					f.dither(x, y, snowLit, 0.9-t*1.6+0.5*noise(float64(x)/18, float64(y)/6, 44)-0.25)
					f.dither(x, y, snowShade, t*1.3-0.55)
				}
			}
			// The cabin: walls of logs, a door, a window and a chimney,
			// under a roof heavy with snow.
			for y := eaves; y < floor; y++ {
				for x := cabinX; x < cabinX+cabinW; x++ {
					c := logA
					if (y-eaves)%4 == 3 {
						c = logB
					}
					f.set(x, y, c)
				}
			}
			for y := eaves + 7; y < floor; y++ {
				for x := cabinX + 7; x < cabinX+15; x++ {
					f.set(x, y, dark)
				}
			}
			for y := 110; y < 122; y++ {
				for x := cabinX + 34; x < cabinX+39; x++ {
					f.set(x, y, brick)
				}
			}
			for x := cabinX + 33; x < cabinX+40; x++ {
				f.set(x, 109, pineSnow)
				f.set(x, 110, pineSnow)
			}
			const peak, apex = cabinX + cabinW/2, 107
			for y := apex; y <= eaves; y++ {
				hw := float64(y-apex)/float64(eaves-apex)*(cabinW/2+6) + 1
				for x := peak - int(hw); x <= peak+int(hw); x++ {
					f.set(x, y, roof)
					// Snow lies along the roof's slopes, three pixels deep,
					// and on the eaves.
					in := hw - math.Abs(float64(x-peak))
					if in < 3.5 || y >= eaves-1 {
						f.set(x, y, pineSnow)
					}
				}
			}
			for x := cabinX - 2; x < cabinX+cabinW+2; x++ {
				f.set(x, floor, snowLit)
				f.set(x, floor-1, snowLit)
			}
			// The window's light on the snow before it.
			for y := floor + 1; y < floor+12; y++ {
				spread := float64(y - floor)
				for x := cabinX + 23 - int(spread*0.8); x < cabinX+37+int(spread*0.8); x++ {
					f.dither(x, y, spill, 0.7-spread/14)
				}
			}
			for _, t := range []struct{ x, base, h int }{
				{12, 150, 62}, {30, 146, 48}, {50, 152, 70}, {76, 143, 32},
				{196, 142, 30}, {268, 150, 58}, {288, 146, 44}, {307, 152, 66},
			} {
				ps := pineSnow
				f.pine(t.x, t.base, t.h, nearPine, &ps)
			}
		},
		frame: func(f pixels, t float64) {
			f.drawStars(skyStars, hexRGB("#566c96"), hexRGB("#dfe8f7"), t, true)
			// The chimney's smoke, puffs rising and drifting with the wind.
			for i := range 7 {
				s := math.Mod(t*0.11+float64(i)/7, 1)
				x := float64(cabinX+36) + s*22 + math.Sin(t*0.7+float64(i)*1.7)*2*s
				y := 107 - s*46
				r := 1.2 + s*3.2
				for py := int(y - r); py <= int(y+r); py++ {
					for px := int(x - r); px <= int(x+r); px++ {
						if math.Hypot(float64(px)+0.5-x, float64(py)+0.5-y) <= r {
							f.dither(px, py, smoke, 0.55*(1-s))
						}
					}
				}
			}
			// The window, its lamp flickering a little.
			warm := 0.5 + 0.5*noise(t*3, 0.5, 45)
			for y := eaves + 6; y < eaves+14; y++ {
				for x := cabinX + 25; x < cabinX+36; x++ {
					f.set(x, y, lamp)
					f.dither(x, y, lampHot, warm*0.8-float64(y-eaves-6)/14)
				}
			}
			for x := cabinX + 24; x < cabinX+37; x++ {
				f.set(x, eaves+5, dark)
				f.set(x, eaves+14, dark)
				f.set(x, eaves+9, logB)
			}
			for y := eaves + 5; y <= eaves+14; y++ {
				f.set(cabinX+24, y, dark)
				f.set(cabinX+30, y, logB)
				f.set(cabinX+36, y, dark)
			}
			// The snow, falling in three layers, the nearest the
			// fastest and the largest.
			for layer, list := range flakes {
				for i, fl := range list {
					x, y := fall(fl.x, fl.y, fl.v, fl.amp, 3+float64(layer)*2, fl.rate, fl.at, t)
					f.set(x, y, flakeColor[layer])
					if layer == 2 && i%3 == 0 {
						f.set(x+1, y, flakeColor[layer])
						f.set(x, y+1, flakeColor[layer])
						f.set(x+1, y+1, flakeColor[layer])
					}
				}
			}
		},
	}
}()

// The Rainy city: a skyline at night in the rain, its windows lighting up
// and going dark, a sign flickering, an antenna's light blinking, the
// street wet with the lights' reflections, rippling as the rain falls.

var rainyCity = func() *scene {
	sky := []rgb{hexRGB("#080919"), hexRGB("#0f1030"), hexRGB("#1a1640"), hexRGB("#2b1d48"), hexRGB("#3e2550")}
	cloud := hexRGB("#141433")
	farBody, farLit := hexRGB("#1c1936"), hexRGB("#4a3d5e")
	nearBody, nearEdge, unlit := hexRGB("#0f0d1f"), hexRGB("#1b1833"), hexRGB("#1a1830")
	warm, cool, tv := hexRGB("#ffcf7a"), hexRGB("#8fd3ff"), hexRGB("#c39cff")
	walk, street := hexRGB("#1b1929"), hexRGB("#0b0a16")
	pole, lamp, cone := hexRGB("#2a2740"), hexRGB("#ffe6a8"), hexRGB("#4d3d3c")
	neon, neonDim, red := hexRGB("#ff4fa3"), hexRGB("#4a1f3c"), hexRGB("#ff3b3b")
	rain := [3]rgb{hexRGB("#353d62"), hexRGB("#6f7aa8"), hexRGB("#a9b3dc")}
	const curb = 146 // the top of the sidewalk; the street starts four pixels down
	const road = curb + 4

	type building struct{ x, w, top int }
	var far, near []building
	for x, i := -4, 0; x < sceneW; i++ {
		w := 10 + int(16*hash(i, 1, 70))
		far = append(far, building{x, w, 66 + int(40*hash(i, 2, 70))})
		x += w - 1
	}
	for x, i := -6, 0; x < sceneW; i++ {
		w := 22 + int(22*hash(i, 1, 71))
		top := 40 + int(58*hash(i, 2, 71))
		near = append(near, building{x, w, top})
		x += w + int(5*hash(i, 3, 71))
	}
	// The windows of the near buildings: some always lit or dark, others
	// switching now and then, each at its own pace.
	type window struct {
		x, y           int
		col            rgb
		lit, changes   bool
		period, offset float64
	}
	var windows []window
	for bi, b := range near {
		for y := b.top + 4; y < curb-5; y += 5 {
			for x := b.x + 3; x+2 <= b.x+b.w-3; x += 4 {
				h := func(k int) float64 { return hash(x*7+bi, y, 72+k) }
				col := warm
				switch v := h(1); {
				case v > 0.82:
					col = tv
				case v > 0.58:
					col = cool
				}
				windows = append(windows, window{x: x, y: y, col: col, lit: h(2) < 0.36,
					changes: h(3) < 0.2, period: 20 + 60*h(4), offset: 100 * h(5)})
			}
		}
	}
	tallest := near[0]
	for _, b := range near {
		if b.top < tallest.top && b.x > 20 && b.x+b.w < sceneW-20 {
			tallest = b
		}
	}
	sign := near[len(near)/2]
	lamps := []int{58, 236}
	type drop struct{ x, y, v float64 }
	var drops [3][]drop
	for layer, n := range [3]int{90, 60, 18} {
		for i := range n {
			drops[layer] = append(drops[layer], drop{x: sceneW * hash(i, 1, 80+layer), y: sceneH * hash(i, 2, 80+layer),
				v: [3]float64{130, 200, 290}[layer] * (0.85 + 0.3*hash(i, 3, 80+layer))})
		}
	}
	return &scene{
		base: func(f pixels) {
			f.gradient(0, road, sky...)
			// Low clouds, lit faintly by the city.
			for y := range 70 {
				for x := range sceneW {
					v := noise(float64(x)/38, float64(y)/9, 73)*0.7 + noise(float64(x)/13, float64(y)/5, 74)*0.3
					f.dither(x, y, cloud, (v-0.5)*2.2*(1-float64(y)/70))
				}
			}
			for _, b := range far {
				for y := b.top; y < curb; y++ {
					for x := b.x; x < b.x+b.w; x++ {
						f.set(x, y, farBody)
						if (x-b.x)%3 == 1 && (y-b.top)%4 == 2 && hash(x, y, 75) < 0.18 {
							f.set(x, y, farLit)
						}
					}
				}
			}
			for _, b := range near {
				for y := b.top; y < curb; y++ {
					for x := b.x; x < b.x+b.w; x++ {
						f.set(x, y, nearBody)
					}
					f.set(b.x, y, nearEdge)
				}
				for x := b.x; x < b.x+b.w; x++ {
					f.set(x, b.top, nearEdge)
				}
			}
			for _, w := range windows {
				for dx := range 2 {
					for dy := range 2 {
						f.set(w.x+dx, w.y+dy, unlit)
					}
				}
			}
			// An antenna on the tallest building, a water tower on another.
			ax := tallest.x + tallest.w/2
			for y := tallest.top - 12; y < tallest.top; y++ {
				f.set(ax, y, nearEdge)
			}
			tower := near[(len(near)/2+2)%len(near)]
			tx := tower.x + tower.w/3
			for y := tower.top - 8; y < tower.top-3; y++ {
				for x := tx; x < tx+7; x++ {
					f.set(x, y, nearEdge)
				}
			}
			for y := tower.top - 3; y < tower.top; y++ {
				f.set(tx+1, y, nearEdge)
				f.set(tx+5, y, nearEdge)
			}
			// The sidewalk, and the street lamps' poles.
			for y := curb; y < road; y++ {
				for x := range sceneW {
					f.set(x, y, walk)
				}
			}
			for _, lx := range lamps {
				for y := curb - 30; y < curb; y++ {
					f.set(lx, y, pole)
				}
				for x := lx - 1; x <= lx+3; x++ {
					f.set(x, curb-31, pole)
				}
			}
		},
		frame: func(f pixels, t float64) {
			for _, w := range windows {
				lit := w.lit
				if w.changes {
					lit = hash(w.x, w.y+int(math.Floor((t+w.offset)/w.period)), 76) < 0.4
				}
				if !lit {
					continue
				}
				for dx := range 2 {
					for dy := range 2 {
						f.set(w.x+dx, w.y+dy, w.col)
					}
				}
			}
			// The antenna's light, blinking, and the sign, flickering now
			// and then.
			if math.Mod(t, 1.6) < 0.5 {
				ax := tallest.x + tallest.w/2
				f.set(ax, tallest.top-13, red)
				f.dither(ax-1, tallest.top-13, red, 0.5)
				f.dither(ax+1, tallest.top-13, red, 0.5)
			}
			sx, sy := sign.x+sign.w-5, sign.top+10
			on := hash(int(t*sceneFPS), 0, 77) > 0.04 && math.Mod(t, 23) > 0.6
			for y := sy; y < sy+16; y++ {
				for x := sx; x < sx+3; x++ {
					c := neonDim
					if on && (y-sy)%4 != 3 {
						c = neon
					}
					f.set(x, y, c)
				}
				if on {
					f.dither(sx-1, y, neon, 0.35)
					f.dither(sx+3, y, neon, 0.35)
				}
			}
			// The lamps and their light on the sidewalk.
			for _, lx := range lamps {
				f.set(lx+2, curb-30, lamp)
				f.set(lx+3, curb-30, lamp)
				for y := curb - 29; y < road+2; y++ {
					spread := float64(y-(curb-29)) * 0.4
					for x := lx + 2 - int(spread); x <= lx+3+int(spread); x++ {
						f.dither(x, y, cone, 0.45-float64(y-(curb-29))/90)
					}
				}
			}
			// The street: dark and wet, the lights above it reflected,
			// rippling.
			for y := road; y < sceneH; y++ {
				d := y - road
				fade := 0.55 - float64(d)/float64(sceneH-road)*0.4
				ripple := int(math.Round(math.Sin(float64(y)*1.3+t*4) * 1.2))
				for x := range sceneW {
					f.set(x, y, street)
					sy := curb - 1 - d*2
					if sy < 0 {
						continue
					}
					src := f.at(min(max(x+ripple, 0), sceneW-1), sy)
					lum := (float64(src[0]) + float64(src[1]) + float64(src[2])) / 765
					if lum > 0.3 {
						f.mix(x, y, src, fade*lum)
					}
				}
			}
			// The rain, slanting, in three layers, and splashes on the
			// street.
			for layer, list := range drops {
				n := [3]int{2, 4, 6}[layer]
				for _, d := range list {
					y := math.Mod(d.y+t*d.v, sceneH+12) - 6
					x := math.Mod(d.x-t*d.v*0.18+sceneW*20, sceneW)
					for k := range n {
						f.set(int(x+float64(k)*0.18), int(y)-k, rain[layer])
					}
				}
			}
			step := int(t * sceneFPS)
			for i := range 26 {
				if hash(i, step, 81) < 0.5 {
					continue
				}
				x, y := int(sceneW*hash(i, step, 82)), road+1+int(float64(sceneH-road-2)*hash(i, step, 83))
				f.set(x, y, rain[2])
				f.dither(x-1, y, rain[1], 0.6)
				f.dither(x+1, y, rain[1], 0.6)
			}
		},
	}
}()
