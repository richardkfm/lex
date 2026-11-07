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

// The Campfire: a fire burning under a night sky of a few stars, between
// hills and pines, its light flickering on the ground, embers rising.

var campfire = func() *scene {
	sky := []rgb{hexRGB("#06051a"), hexRGB("#0d0a27"), hexRGB("#181033"), hexRGB("#291539"), hexRGB("#3a1a3b")}
	far, near, ground, deep := hexRGB("#251a3d"), hexRGB("#160e26"), hexRGB("#130b15"), hexRGB("#0b060c")
	treeBody := hexRGB("#0a0610")
	ridge := func(x int) int {
		return 94 + int(16*noise(float64(x)/40, 0, 3)+6*noise(float64(x)/11, 0, 4))
	}
	hill := func(x int) int {
		return 118 + int(8*noise(float64(x)/30, 5, 6)+3*noise(float64(x)/8, 5, 7))
	}
	groundTop := func(x int) int { return 134 + int(2.5*noise(float64(x)/22, 9, 8)) }
	skyStars := stars(100, 0.007, 20)
	// The fire's foot, left of the middle, where a split down the middle
	// of the window would leave it in a gap.
	const fx, fy = 116, 151
	fireLight := []rgb{hexRGB("#24110f"), hexRGB("#3a1912"), hexRGB("#5a2614"), hexRGB("#7e3717")}
	skyLight := []rgb{hexRGB("#3a1a3b"), hexRGB("#4c1f3a"), hexRGB("#622638")}
	flame := []rgb{hexRGB("#a0231a"), hexRGB("#ea4d1b"), hexRGB("#ff951f"), hexRGB("#ffd247"), hexRGB("#fff3c4")}
	ember := []rgb{hexRGB("#fff0a6"), hexRGB("#ffb238"), hexRGB("#ec5a1e"), hexRGB("#8c2a19")}
	logBark, logEnd, logCore := hexRGB("#4d301d"), hexRGB("#8a5d3b"), hexRGB("#c48b55")
	stone, stoneLit := hexRGB("#2f2a36"), hexRGB("#5e4b4b")
	return &scene{
		base: func(f pixels) {
			f.gradient(0, 136, sky...)
			f.drawStars(skyStars, hexRGB("#5d5590"), hexRGB("#d9d1ff"), 0, false)
			for x := range sceneW {
				for y := ridge(x); y < sceneH; y++ {
					f.set(x, y, far)
				}
				for y := hill(x); y < sceneH; y++ {
					f.set(x, y, near)
				}
				gt := groundTop(x)
				for y := gt; y < sceneH; y++ {
					f.set(x, y, ground)
					f.dither(x, y, deep, float64(y-gt)/float64(sceneH-gt)*1.2)
				}
			}
			for _, t := range []struct{ x, base, h int }{
				{8, 146, 58}, {25, 142, 44}, {42, 147, 64}, {58, 139, 28},
				{196, 137, 20}, {214, 141, 36}, {234, 147, 62}, {254, 143, 46},
				{274, 147, 66}, {294, 141, 40}, {312, 147, 56},
			} {
				f.pine(t.x, t.base, t.h, treeBody, nil)
			}
		},
		frame: func(f pixels, t float64) {
			flicker := 0.78 + 0.22*noise(t*6, 0.5, 9)
			f.drawStars(skyStars, hexRGB("#5d5590"), hexRGB("#d9d1ff"), t, true)
			// The fire's light: on the sky above it, and on the ground.
			f.glow(fx, fy-14, 46, 34, 0.9*flicker, skyLight, func(x, y int) bool { return y < hill(x) && y >= ridge(x)-6 })
			f.glow(fx, fy, 96, 26, flicker, fireLight, func(x, y int) bool { return y >= groundTop(x) })
			// The stones round the fire, lit on the side facing it.
			for i := range 9 {
				a := math.Pi * (0.05 + 0.9*float64(i)/8)
				sx, sy := fx+int(18*math.Cos(a+math.Pi)), fy+2+int(4*math.Sin(a))
				for dx := range 4 {
					f.set(sx+dx-1, sy, stone)
					f.set(sx+dx-1, sy+1, stone)
				}
				f.dither(sx, sy, stoneLit, flicker)
				f.dither(sx+1, sy, stoneLit, flicker*0.8)
			}
			// The flames: three tongues swaying, torn by noise rising
			// through them.
			for y := fy - 46; y < fy; y++ {
				h := float64(fy-y) / 44
				for x := fx - 20; x <= fx+20; x++ {
					heat := 0.0
					for i, tg := range [3]struct{ off, w, tall, sway float64 }{{0, 11, 1, 2.3}, {-6, 7, 0.68, 3.1}, {6, 7, 0.74, 2.7}} {
						if h >= tg.tall {
							continue
						}
						up := h / tg.tall
						c := float64(fx) + tg.off + math.Sin(t*tg.sway+h*4+float64(i))*2.4*up
						side := 1 - math.Abs(float64(x)+0.5-c)/(tg.w*(1-up*0.85))
						if side > 0 {
							heat = max(heat, side*(1-up*up*0.7))
						}
					}
					if heat <= 0 {
						continue
					}
					heat *= 0.45 + 0.85*noise(float64(x)*0.32, float64(y)*0.3+t*9, 11)
					heat *= flicker*0.25 + 0.85
					switch {
					case heat > 0.82:
						f.set(x, y, flame[4])
					case heat > 0.62:
						f.set(x, y, flame[3])
					case heat > 0.42:
						f.set(x, y, flame[2])
					case heat > 0.25:
						f.set(x, y, flame[1])
					case heat > 0.13:
						f.set(x, y, flame[0])
					}
				}
			}
			// The logs, crossed, in front of the flames' foot.
			for _, l := range [2][4]float64{{fx - 16, fy + 3, fx + 15, fy - 4}, {fx - 15, fy - 4, fx + 16, fy + 3}} {
				n := int(math.Hypot(l[2]-l[0], l[3]-l[1]))
				for i := 0; i <= n; i++ {
					x := l[0] + (l[2]-l[0])*float64(i)/float64(n)
					y := l[1] + (l[3]-l[1])*float64(i)/float64(n)
					for d := -1; d <= 1; d++ {
						f.set(int(x), int(y)+d, logBark)
					}
				}
				for _, e := range [2][2]float64{{l[0], l[1]}, {l[2], l[3]}} {
					f.set(int(e[0]), int(e[1])-1, logEnd)
					f.set(int(e[0]), int(e[1]), logCore)
					f.set(int(e[0]), int(e[1])+1, logEnd)
				}
			}
			for x := fx - 9; x <= fx+9; x++ {
				f.dither(x, fy-1, flame[1], 0.5*flicker)
			}
			// The embers, rising and fading.
			for i := range 28 {
				v := 9 + 14*hash(i, 1, 30)
				life := 45 + 45*hash(i, 2, 30)
				s := math.Mod(t*v/life+hash(i, 3, 30), 1)
				x := float64(fx) + (hash(i, 4, 30)-0.5)*14 + math.Sin(t*1.9+float64(i))*3*s + (hash(i, 5, 30)-0.5)*26*s
				y := float64(fy-6) - s*life
				f.set(int(x), int(y), ember[min(int(s*4.4), 3)])
			}
		},
	}
}()

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
