package main

import (
	"fmt"
	"math"

	"github.com/egoist/mygo/ui"
)

// The scenes an animation pane plays (animations.go): abstract places,
// lit, and alive in small ways: sand drifts, traffic streams, water
// falls, a helix turns, lights cross the sky, a scope traces its
// figures, packets run through a network. Each is composed on the
// design's pixels and goes on past them in a larger pane: more sky, more
// dunes, more skyline, more towers. Each is painted from the time and the
// pane's size alone.

// pick returns which of n a number from 0 to 1 picks.
func pick(h float64, n int) int { return min(int(h*float64(n)), n-1) }

// flock flies n birds over the sky for dur seconds of every period, from
// start cells in, gap cells apart, in rows two and three: each a v as its
// wings beat down, a ^ as they come up.
func (g *grid) flock(t, period, dur float64, n int, start, gap float64, c rgb) {
	fs := wrap(t, period)
	if fs >= dur {
		return
	}
	span := g.wd() / (designCols * 2)
	for k := range n {
		fk := float64(k)
		ch := '^'
		if int(math.Floor(t*3+fk))&1 == 1 {
			ch = 'v'
		}
		g.char(g.x0/2+start+fs*6*span+fk*gap, 2+float64(k%2), ch, c)
	}
}

// Dune Sea: dunes in four ranges to a setting sun, ringed in haze, the
// air over them shimmering, sand blowing off their crests, the wind
// ripples on them creeping, and a crescent moon over them.
var duneSeaAnim = &animation{id: "dune-sea", name: "Dune Sea", fps: 15, tile: ui.Hex("#c4566a"), draw: drawDuneSea}

// duneShape is the profile of dunes along u, a dune a unit: a long slope
// up the wind, a short steep one down it.
func duneShape(u float64) float64 {
	u = wrap(u, 1)
	if u < 0.68 {
		return smooth(u / 0.68)
	}
	return 1 - smooth((u-0.68)/0.32)
}

func drawDuneSea(g *grid, t float64, pal animPalette) {
	ink := pal.inks(duneSeaRoles)
	const hz = 36.0
	var (
		night, dusk = ink("#120e2c"), ink("#2b1b4a")
		crest, sand = ink("#ffe2b8"), ink("#ffe6c0")
		horizon     = ink("#f6a879")
	)
	g.sky(hz+4, night, dusk, ink("#6a2c5a"), ink("#c4566a"), ink("#f08f63"), ink("#ffd08a"))
	g.stars(t, 0.07, 5, func(float64) float64 { return 18 }, ink("#3a2f62"), ink("#f2e9ff"))
	g.glow(104, 9, 9, ink("#9a90c8"), 0.3)
	g.disc(104, 9, 3.6, ink("#f6efd6"), 1)
	g.disc(105.6, 8.2, 3.3, mixRGB(dusk, night, 0.5), 1)

	const sx, sy = 40.0, hz - 2
	for k := range 4 {
		fk := float64(k)
		g.ring(sx, sy, 15+fk*5+math.Sin(t*0.8+fk)*0.7, ink("#ffd9a0"), 0.09-fk*0.018)
	}
	g.glow(sx, sy, 26, ink("#ffbf7a"), 0.45)
	sunTop, sunBottom := ink("#fff0c4"), ink("#ff9a5c")
	for y := math.Floor(sy - 11); y <= sy+11; y++ {
		for x := math.Floor(sx - 11); x <= sx+11; x++ {
			if d := math.Hypot(x+0.5-sx, y+0.5-sy); d <= 11 {
				g.mix(x, y, mixRGB(sunTop, sunBottom, (y-sy+11)/22), clamp(11.5-d, 0, 1))
			}
		}
	}
	g.flock(t, 23, 9, 3, 4, 1.3, dusk)

	layers := []struct {
		base, amp, l, off float64
		lit, shade        rgb
		haze              float64
	}{
		{39, 3, 34, 0.1, ink("#eaa27d"), ink("#c47a62"), 0.45},
		{44, 5, 46, 0.55, ink("#e0905f"), ink("#a85f45"), 0.28},
		{51, 7, 60, 0.3, ink("#d27b45"), ink("#8c4a33"), 0.12},
		{60, 9, 84, 0.8, ink("#c1662f"), ink("#6e3826"), 0},
	}
	for k, l := range layers {
		fk := float64(k)
		ridge := func(x float64) float64 {
			return l.base - l.amp*duneShape(x/l.l+l.off) - 1.2*math.Sin(x*0.07+fk*1.7)
		}
		for x := g.x0; x < g.x1; x++ {
			top, facing := ridge(x), ridge(x-1)-ridge(x+1)
			for y := max(g.y0, math.Floor(top)); y < g.y1; y++ {
				d := y - top
				c := mixRGB(l.shade, l.lit, clamp(0.5+facing*0.9, 0, 1))
				c = mixRGB(c, l.shade, d/22)
				switch rip := math.Sin((d*0.62 + x*(0.11+fk*0.03) - t*(0.25+0.2*fk)) * 2.1); {
				case rip > 0.55:
					c = mixRGB(c, l.shade, 0.22+0.1*fk)
				case rip < -0.85:
					c = mixRGB(c, crest, 0.12)
				}
				g.set(x, y, mixRGB(c, horizon, l.haze))
			}
			g.mix(x, math.Floor(top), crest, 0.55-l.haze*0.6)
		}
		if k == 0 {
			// The air shimmers over the far dunes.
			row := make([]rgb, int(g.wd()))
			for y := hz - 6; y < hz+2; y++ {
				sh := math.Round(math.Sin(y*1.7+t*4) * 0.8)
				if sh == 0 {
					continue
				}
				for i := range row {
					row[i] = g.get(g.x0+float64(i)-sh, y)
				}
				for i, c := range row {
					g.set(g.x0+float64(i), y, c)
				}
			}
		}
		if k >= 2 {
			// Sand blows off the near crests.
			for i := range int(26 * g.wd() / 128) {
				x0 := g.x0 + g.wd()*hash(i, k, 7)
				s := wrap(t*0.45+hash(i, k, 8), 1)
				g.splat(x0+s*16, ridge(x0)-s*3+s*s*5, sand, (1-s)*0.55)
			}
		}
	}
	gust := ink("#c1662f")
	for i := range 3 {
		if s := wrap(t*0.4+float64(i)*1.9, 6); s < 2 {
			x := g.x0 + wrap(g.wd()*hash(i, int(math.Floor(t/15)), 9)+s*18, g.wd())
			g.char(x/2, 12+float64(i%2), '~', mixRGB(gust, crest, 1-s/2))
		}
	}
}

// Neon Grid: a city of towers at night, their windows lit and some
// flickering, neon along their edges, a hologram turning over them, light
// trails streaming along three roads, and rain on the street, which
// mirrors it all.
var neonGridAnim = &animation{id: "neon-grid", name: "Neon Grid", fps: 20, tile: ui.Hex("#c2187a"), draw: drawNeonGrid}

func drawNeonGrid(g *grid, t float64, pal animPalette) {
	ink := pal.inks(neonGridRoles)
	const street = 52.0
	var (
		pink, cyan, amber = ink("#ff3fa8"), ink("#3fd9ff"), ink("#ffb347")
		red, dusk         = ink("#ff3045"), ink("#241a40")
		lit               = [4]rgb{cyan, ink("#ff4fd8"), amber, ink("#c8c0ff")}
	)
	g.sky(street, ink("#04040f"), ink("#0a0920"), ink("#1a0f33"), ink("#2e1240"))
	g.stars(t, 0.03, 15, func(float64) float64 { return 12 }, ink("#231d45"), ink("#9f96d9"))
	for y := max(30, g.y0); y < street; y++ {
		for x := g.x0; x < g.x1; x++ {
			g.mix(x, y, mixRGB(pink, cyan, noise(x/30, t*0.1, 3)), 0.12*(y-30)/22)
		}
	}
	towers := func(seed int, minW, maxW, lo, hi float64, body rgb, bright, pitch float64) {
		g.spans(-3, func(i int) (float64, float64) {
			return minW + math.Floor((maxW-minW)*hash(i, 1, seed)), 1 + math.Floor(3*hash(i, 8, seed))
		}, func(i int, x, w float64) {
			top := math.Floor(lo + (hi-lo)*hash(i, 2, seed))
			b := mixRGB(body, dusk, hash(i, 3, seed)*0.4)
			for y := top; y < street; y++ {
				for xx := x; xx < x+w; xx++ {
					g.set(xx, y, b)
				}
			}
			for y := top + 2; y < street-1; y += pitch {
				for xx := x + 1; xx < x+w-1; xx += 2 {
					ix, iy := int(xx), int(y)
					if hash(ix, iy, seed+1) > 0.32 {
						continue
					}
					if hash(ix, iy, seed+2) < 0.1 && hash(ix, iy+int(math.Floor(t/4)), seed+3) >= 0.5 {
						continue // out for now
					}
					g.set(xx, y, mixRGB(b, lit[pick(hash(ix, iy, seed+4), 4)], bright*(0.5+0.5*hash(ix, iy, seed+5))))
				}
			}
			if hash(i, 4, seed) < 0.35 && hash(i, int(math.Floor(t*12)), seed+6) > 0.05 {
				c := lit[pick(hash(i, 5, seed), 2)]
				for y := top; y < street; y++ {
					g.mix(x, y, c, bright)
				}
				for xx := x; xx < x+w; xx++ {
					g.mix(xx, top, c, bright*0.8)
				}
			}
			if top < lo+6 && wrap(t+float64(i)*0.7, 2) < 0.6 {
				g.set(x+math.Floor(w/2), top-1, red)
			}
		})
	}
	towers(31, 5, 11, 12, 30, ink("#161230"), 0.35, 3)

	// The hologram: a wire octahedron, turning, ringed, over the towers.
	const hx, hy, hr, tilt = 92.0, 13.0, 7.5, 0.45
	a := t * 0.7
	var pts [6][3]float64
	for i, v := range [6][3]float64{{1, 0, 0}, {-1, 0, 0}, {0, 1, 0}, {0, -1, 0}, {0, 0, 1}, {0, 0, -1}} {
		x1 := v[0]*math.Cos(a) + v[2]*math.Sin(a)
		z1 := -v[0]*math.Sin(a) + v[2]*math.Cos(a)
		y2 := v[1]*math.Cos(tilt) - z1*math.Sin(tilt)
		z2 := v[1]*math.Sin(tilt) + z1*math.Cos(tilt)
		pts[i] = [3]float64{hx + x1*hr, hy + y2*hr*0.9, z2}
	}
	flick := 1.0
	if hash(int(math.Floor(t*12)), 0, 33) < 0.07 {
		flick = 0.35
	}
	front, back := ink("#9ff6ff"), ink("#ff7ad9")
	for _, e := range [12][2]int{{0, 2}, {0, 3}, {0, 4}, {0, 5}, {1, 2}, {1, 3}, {1, 4}, {1, 5}, {2, 4}, {4, 3}, {3, 5}, {5, 2}} {
		p, q := pts[e[0]], pts[e[1]]
		c := back
		if p[2]+q[2] > 0 {
			c = front
		}
		g.aline(p[0], p[1], q[0], q[1], c, 0.55*flick)
	}
	g.glow(hx, hy, 14, cyan, 0.12*flick)
	for k := range 24 {
		an := float64(k)/24*6.28 + t*0.4
		a := 0.25
		if math.Sin(an) > 0 {
			a = 0.6
		}
		g.splat(hx+12*math.Cos(an), hy+9+2.4*math.Sin(an), back, a)
	}
	towers(41, 7, 15, 22, 40, ink("#0b0918"), 0.75, 2)

	roads := []struct {
		y, dir     float64
		head, tail rgb
		v          float64
	}{
		{42, 1, ink("#fff4d6"), amber, 24},
		{46, -1, ink("#ff8a8a"), red, 18},
		{49, 1, ink("#c8faff"), cyan, 32},
	}
	asphalt := ink("#2a2340")
	for k, r := range roads {
		fk := float64(k)
		ry := func(x float64) float64 { return r.y + 1.8*math.Sin(x*0.03+fk*2) }
		for x := g.x0; x < g.x1; x++ {
			g.mix(x, ry(x), asphalt, 0.8)
		}
		for c := range int(math.Round(9 * g.wd() / 128)) {
			p := g.x0 + wrap(g.wd()*hash(c, k, 34)+r.dir*r.v*t, g.wd()+30) - 15
			for j := range 14 {
				fj := float64(j)
				x := p - r.dir*fj
				g.splat(x, ry(x)-0.5, mixRGB(r.head, r.tail, fj/6), (1-fj/14)*0.8)
			}
		}
	}

	wet := ink("#07060f")
	for y := max(street, g.y0); y < g.y1; y++ {
		d := y - street
		for x := g.x0; x < g.x1; x++ {
			rip := math.Round(math.Sin(y*1.9+t*4+x*0.05) * (0.6 + d*0.08))
			g.set(x, y, mixRGB(g.get(x+rip, street-1-d*1.6), wet, 0.45+d*0.035))
		}
	}
	drop := ink("#9fb4e8")
	for k := range max(1, int(math.Round(5*g.wd()/128))) {
		ph := t*1.2 + float64(k)*0.37
		s := wrap(ph, 1.8)
		x := g.x0 + g.wd()*hash(k, int(math.Floor(ph/1.8)), 35)
		y := street + 3 + (g.y1-street-4)*hash(k, 9, 35)
		for an := 0.0; an < 6.28; an += 0.3 {
			g.splat(x+s*4*math.Cos(an), y+s*1.1*math.Sin(an), drop, 0.4*(1-s/1.8))
		}
	}
	rain := ink("#8fa3d6")
	for k := range int(140 * g.area()) {
		v := 28 + 20*hash(k, 1, 36) // slow enough to read as falling, not flicker
		x := g.x0 + wrap(g.wd()*hash(k, 2, 36)-t*v*0.25, g.wd())
		y := g.y0 + wrap(g.hd()*hash(k, 3, 36)+t*v, g.hd())
		for j := range 3 {
			fj := float64(j)
			g.mix(x+fj*0.25, y-fj, rain, 0.32-fj*0.08)
		}
	}
}

// Garden City: white towers at dusk, green spiraling up them, gardens on
// their tops and wind sails turning over those, water falling from them,
// bridges arching between them with planters glowing, a maglev streaking
// by, and a canal mirroring it all, lanterns drifting on it.
var gardenCityAnim = &animation{id: "garden-city", name: "Garden City", fps: 15, tile: ui.Hex("#2e9b57"), draw: drawGardenCity}

// gardenTower is a tower of the garden city: its middle, width and top,
// and a seed of its own.
type gardenTower struct {
	cx, w, top float64
	s          int
}

// halfWidth is how far the tower goes either side of its middle at y,
// down to the canal: a little wider at its foot.
func (tw gardenTower) halfWidth(y, canal float64) float64 {
	return tw.w / 2 * (0.82 + 0.18*(y-tw.top)/(canal-tw.top))
}

func drawGardenCity(g *grid, t float64, pal animPalette) {
	ink := pal.inks(gardenCityRoles)
	const canal, sx, sy = 52.0, 86.0, 42.0
	var (
		gold, teal         = ink("#ffcc66"), ink("#5ff0e0")
		haze, green        = ink("#e3a258"), ink("#6cc46f")
		ivy, white         = ink("#2e8b57"), rgb{255, 255, 255}
		stone, stoneShadow = ink("#f4ecd8"), ink("#5f7f88")
		leafDark, leafLit  = ink("#1f6a45"), ink("#7fd47a")
		sunlit             = ink("#ffe08a")
	)
	g.sky(canal, ink("#0a2231"), ink("#103d4a"), ink("#1d6668"), ink("#5b9c86"), haze, ink("#ffd58a"))
	g.stars(t, 0.06, 131, func(float64) float64 { return 14 }, ink("#1a4a58"), ink("#d8fff6"))

	// The sun, low behind the towers, its rays through the haze.
	g.glow(sx, sy, 46, ink("#ffbf6a"), 0.5)
	ray := ink("#ffe2a0")
	for y := g.y0; y < sy; y++ {
		for x := g.x0; x < g.x1; x++ {
			d := math.Hypot(x-sx, y-sy)
			if d >= 95 {
				continue
			}
			an := math.Atan2(y-sy, x-sx)
			if r := noise(an*7+t*0.04, 0.5, 132)*0.7 + noise(an*19-t*0.03, 1.5, 133)*0.3; r > 0.55 {
				g.mix(x, y, ray, (r-0.55)*0.55*(1-d/95))
			}
		}
	}
	g.disc(sx, sy, 6.5, ink("#fff3cc"), 1)

	// The far city, hazy, green on its roofs.
	far, farLit := ink("#2a6466"), ink("#5b9c86")
	g.spans(-2, func(i int) (float64, float64) {
		return 3 + math.Floor(4*hash(i, 1, 134)), 1 + math.Floor(3*hash(i, 4, 134))
	}, func(i int, x, w float64) {
		top := 26 + math.Floor(12*hash(i, 2, 134))
		c := mixRGB(far, farLit, 0.35+0.25*hash(i, 3, 134))
		for y := top; y < canal; y++ {
			cy := mixRGB(c, haze, clamp((y-30)/30, 0, 0.35))
			for xx := x; xx < x+w; xx++ {
				g.set(xx, y, cy)
			}
		}
		for xx := x; xx < x+w; xx++ {
			g.set(xx, top, mixRGB(c, green, 0.5))
		}
		for y := top + 2; y < canal; y += 3 {
			for xx := x + 1; xx < x+w-1; xx += 2 {
				if hash(int(xx), int(y), 135) < 0.25 {
					g.mix(xx, y, gold, 0.55)
				}
			}
		}
	})

	// The maglev's track, on pillars, between the towers' feet.
	track := func(x float64) float64 { return 45 + 2.2*math.Sin(x*0.045+0.6) }
	rail, pillar := ink("#cfe6e0"), ink("#9fbcb6")
	for x := g.x0; x < g.x1; x++ {
		g.set(x, track(x)+1, rail)
		if wrap(x, 13) == 6 {
			for y := math.Floor(track(x)) + 2; y < canal; y++ {
				g.set(x, y, pillar)
			}
		}
	}

	// The towers, and more past the design's edges for a wider pane.
	towers := []gardenTower{{18, 10, 12, 1}, {43, 7, 19, 2}, {66, 12, 11, 3}, {104, 9, 14, 4}}
	first := 0
	for k := 1; ; k++ {
		cx := math.Floor(18 - 30*float64(k) + 6*(hash(k, 1, 149)-0.5))
		if cx < g.x0-12 {
			break
		}
		tw := gardenTower{cx, 7 + math.Floor(5*hash(k, 2, 149)), 9 + math.Floor(11*hash(k, 3, 149)), 4 + 2*k}
		towers = append([]gardenTower{tw}, towers...)
		first++
	}
	for k := 1; ; k++ {
		cx := math.Floor(104 + 30*float64(k) + 6*(hash(-k, 1, 149)-0.5))
		if cx > g.x1+12 {
			break
		}
		towers = append(towers, gardenTower{cx, 7 + math.Floor(5*hash(-k, 2, 149)), 9 + math.Floor(11*hash(-k, 3, 149)), 5 + 2*k})
	}
	crownDark, crownLit, sail, beacon := ink("#174d34"), ink("#ffd98a"), ink("#d8e6e2"), ink("#ff5a5a")
	fallFoam, fall, spray := ink("#e6fbff"), ink("#7fd6e6"), ink("#c8f6ff")
	for _, tw := range towers {
		s := float64(tw.s)
		lit := 1.0 // the side the sun lights
		if tw.cx > sx {
			lit = -1
		}
		var falls [][2]float64
		for y := tw.top; y < canal; y++ {
			h := tw.halfWidth(y, canal)
			for x := math.Floor(tw.cx - h); x <= tw.cx+h; x++ {
				u := clamp((x+0.5-tw.cx)/h, -1, 1)
				v := u * lit
				c := mixRGB(stoneShadow, stone, (v+1)/2)
				if v > 0.75 {
					c = mixRGB(c, gold, 0.5)
				}
				ix, iy := int(x), int(y)
				if band := math.Sin(y*0.42 - math.Asin(u)*1.8 + s*2 + t*0.05); band > 0.45 {
					leaf := noise(x*0.9, y*0.7, 136+tw.s)
					c = mixRGB(leafDark, leafLit, clamp(leaf*0.7+(v+1)*0.25, 0, 1))
					if v > 0.6 {
						c = mixRGB(c, sunlit, 0.35)
					}
				} else if int(y-tw.top)%3 == 1 && int(math.Floor(x-tw.cx+20))%2 == 0 && hash(ix, iy, 137+tw.s) < 0.42 {
					w := gold
					if hash(ix, iy, 138) >= 0.75 {
						w = teal
					}
					c = mixRGB(c, w, 0.85-0.3*hash(ix, iy+int(math.Floor(t/6)), 139))
				}
				g.set(x, y, c)
			}
			iy := int(y)
			if math.Sin(y*0.42+math.Pi/2*1.8+s*2+t*0.05) > 0.45 && hash(iy, 1, 140+tw.s) < 0.5 {
				g.set(tw.cx-h-1, y+1, ivy)
			}
			if math.Sin(y*0.42-math.Pi/2*1.8+s*2+t*0.05) > 0.45 && hash(iy, 2, 140+tw.s) < 0.5 {
				g.set(tw.cx+h+1, y+1, green)
				if hash(iy, 3, 140+tw.s) < 0.18 {
					falls = append(falls, [2]float64{tw.cx + h + 1, y + 2})
				}
			}
		}
		for _, f := range falls[:min(len(falls), 2)] {
			for y := f[1]; y < canal; y++ {
				c := fall
				if wrap(y-t*14+f[0], 4) < 1.2 {
					c = fallFoam
				}
				g.set(f[0], y, c)
			}
			ch := '·'
			if int(math.Floor(t*5+f[0]))&1 == 1 {
				ch = '°'
			}
			g.char(f[0]/2, (canal-1)/4, ch, spray)
		}
		// A garden crowning it, rim-lit on the sun's side.
		rx := tw.w/2 + 2.4
		for y := tw.top - 4; y <= tw.top+2; y++ {
			for x := math.Floor(tw.cx - rx - 1); x <= tw.cx+rx+1; x++ {
				v := (x + 0.5 - tw.cx) / rx * lit
				e := math.Hypot((x+0.5-tw.cx)/rx, (y+0.5-tw.top+0.5)/3.2) - 0.4*noise(x*0.9, y*0.9+t*0.1, 146+tw.s)
				if e >= 0.8 {
					continue
				}
				c := mixRGB(crownDark, green, clamp(noise(x*1.3, y*1.3, 147)*0.6+(v+1)*0.25, 0, 1))
				if e > 0.58 && v > -0.3 && y < tw.top+1 {
					c = mixRGB(c, crownLit, 0.55)
				}
				g.set(x, y, c)
			}
		}
		for k := range 3 {
			fk := float64(k)
			x := tw.cx + (hash(k, 1, 148+tw.s)-0.5)*tw.w
			g.set(x, tw.top-1, mixRGB(ivy, gold, 0.5+0.5*math.Sin(t*1.5+fk*2+s)))
		}
		// The wind sail: two helical blades turning on a mast.
		for k := range 7 {
			fk := float64(k)
			y, ph := tw.top-4-fk, t*3+fk*0.55+s
			g.set(tw.cx, y, sail)
			near, far := 0.95, 0.4
			if math.Cos(ph) <= 0 {
				near, far = far, near
			}
			g.splat(tw.cx+0.5+2.6*math.Sin(ph), y+0.5, white, near)
			g.splat(tw.cx+0.5-2.6*math.Sin(ph), y+0.5, white, far)
		}
		if wrap(t+s, 2) < 0.5 {
			g.set(tw.cx, tw.top-11, beacon)
		}
	}

	// Bridges, arching from tower to tower, planters glowing along them:
	// those of the design, and one to each tower past it.
	type bridge struct {
		i, j   int
		ya, yb float64
	}
	bridges := []bridge{{0, 1, 24, 26}, {1, 2, 30, 29}, {2, 3, 20, 24}, {0, 2, 40, 40}}
	for i := range bridges {
		bridges[i].i += first
		bridges[i].j += first
	}
	for i := range towers {
		if i < first || i > first+3 {
			j := i + 1
			if i > first+3 {
				j = i - 1
			}
			ya := 21 + 18*hash(towers[i].s, 5, 150)
			bridges = append(bridges, bridge{min(i, j), max(i, j), ya, ya + 6*(hash(towers[i].s, 6, 150)-0.5)})
		}
	}
	deck, deckShadow := ink("#efe6d2"), ink("#9fb5b0")
	for k, b := range bridges {
		A, B := towers[b.i], towers[b.j]
		x0, x1 := A.cx+A.halfWidth(b.ya, canal)+1, B.cx-B.halfWidth(b.yb, canal)-1
		for x := math.Floor(x0); x <= x1; x++ {
			u := (x - x0) / max(x1-x0, 1)
			y := b.ya + (b.yb-b.ya)*u - 2.2*math.Sin(math.Pi*u)
			g.set(x, y, deck)
			g.set(x, y+1, deckShadow)
			if ix := int(x); ix%3 == 0 {
				p := 0.6 + 0.4*math.Sin(t*2+x*0.7+float64(k))
				c := teal
				if ix%6 != 0 {
					c = gold
				}
				g.set(x, y-1, mixRGB(ivy, c, p))
				g.glow(x+0.5, y-0.5, 2.2, c, 0.25*p)
			}
			if hash(int(x), k, 141) < 0.3 {
				g.set(x, y+2, ivy)
			}
		}
	}

	// The maglev, streaking through with its trail of light.
	head := g.x0 + wrap(t*26, g.wd()+90) - 30
	nose, cabin, trailEnd := ink("#f2fffd"), ink("#bfeee6"), ink("#1d6668")
	for j := range 34 {
		fj := float64(j)
		x := head - fj
		if x < g.x0 || x >= g.x1 {
			continue
		}
		y := track(x)
		if j < 12 {
			g.splat(x, y-0.4, nose, 0.95)
			g.set(x, y-1.2, cabin)
		} else {
			g.splat(x, y-0.4, mixRGB(teal, trailEnd, (fj-12)/22), 0.7*(1-(fj-12)/22))
		}
	}
	g.glow(head+1, track(head)-0.5, 4, ink("#e8fffb"), 0.4)

	// The canal: everything mirrored, rippling, the sun glittering on it.
	water, glint, ripple := ink("#0c3440"), ink("#fff0c0"), ink("#9fe8e0")
	for y := max(canal, g.y0); y < g.y1; y++ {
		d := y - canal
		f, sy2 := min(d/12, 1), canal-1-d*1.6
		rip := math.Sin(y*0.9+t*1.8)*(0.4+d*0.08) + math.Sin(y*2.3-t*2.6)*0.3
		hw := 2.5 + min(d, 16)*0.9
		for x := g.x0; x < g.x1; x++ {
			xr := x + rip + 0.5*math.Sin(x*0.15+y*0.5+t)
			x0 := math.Floor(xr)
			c := mixRGB(mixRGB(g.get(x0, sy2), g.get(x0+1, sy2), xr-x0), water, 0.32+f*0.4)
			if dx := math.Abs(x - sx); dx < hw && math.Sin(x*1.3+y*2.7+t*4)*math.Sin(x*0.5-t*1.7+y*1.1) > 0.3 {
				c = mixRGB(c, glint, 0.8*(1-dx/hw))
			}
			if math.Sin(x*0.21+y*3.1-t*1.2+2*math.Sin(x*0.05)) > 0.92 {
				c = mixRGB(c, ripple, 0.2)
			}
			g.set(x, y, c)
		}
	}
	quay := ink("#efe2c4")
	for x := g.x0; x < g.x1; x++ {
		g.mix(x, canal, quay, 0.45)
	}
	for k := range max(1, int(math.Round(5*g.wd()/128))) {
		x := g.x0 + wrap(g.wd()*hash(k, 1, 142)+t*(0.8+hash(k, 2, 142)), g.wd())
		y := canal + 3 + (g.y1-canal-4)*hash(k, 3, 142)
		g.splat(x, y, gold, 1)
		g.glow(x, y, 3, gold, 0.3)
		g.mix(x, y+2, gold, 0.25)
	}

	// Leaves framing the bottom corners, rim-lit by the sun.
	shadeDark, shadeLit, rim := ink("#0b211a"), ink("#163b2a"), mixRGB(ink("#1f4a33"), ink("#ffcf7a"), 0.5)
	leaves := func(cx, cy, r, sway float64) {
		for y := math.Floor(cy - r); y < min(g.y1, cy+r+1); y++ {
			for x := math.Floor(cx - r*1.4); x <= cx+r*1.4; x++ {
				e := math.Hypot((x-cx-sway)/(r*1.4), (y-cy)/r) - 0.25*noise(x/2, y/2, 143)
				if e >= 1 {
					continue
				}
				if e > 0.82 && y < cy {
					g.set(x, y, rim)
				} else {
					g.set(x, y, mixRGB(shadeDark, shadeLit, noise(x/1.5, y/1.5, 144)))
				}
			}
		}
	}
	leaves(g.x0+4, g.y1-3, 9, math.Sin(t*0.9)*0.6)
	leaves(g.x0-2, g.y1-12, 6, math.Sin(t*0.9+1)*0.6)
	leaves(g.x1-4, g.y1-4, 10, math.Sin(t*0.8+2)*0.6)

	// Spores of light drifting up, and birds now and then.
	sporeGold, sporeTeal := ink("#ffe08a"), ink("#9ff8ea")
	for k := range int(16 * g.area()) {
		fk := float64(k)
		if math.Sin(t*2+fk) <= -0.3 {
			continue
		}
		x := g.x0 + wrap(g.wd()*hash(k, 1, 145)+math.Sin(t*0.6+fk)*4, g.wd())
		y := g.y0 + wrap(g.hd()*hash(k, 2, 145)-t*(1+1.5*hash(k, 3, 145)), g.hd())
		c := sporeGold
		if k%3 == 0 {
			c = sporeTeal
		}
		col, row := cellOf(x, y)
		g.char(col, row, '·', c)
	}
	g.flock(t, 19, 8, 4, 6, 1.4, ink("#0f3340"))
}

// Helix: a double helix turning, its base pairs lettered as they face
// out, sparks running up its strands, a ring circling it, in the colors
// of the window's theme.
var helixAnim = &animation{id: "helix", name: "Helix", fps: 24, tile: ui.Hex("#6a4cc2"), draw: drawHelix}

func drawHelix(g *grid, t float64, pal animPalette) {
	bg, acc, sec, txt := pal.bg, pal.accent, pal.second, pal.text
	bases := [4]rgb{pal.ansi[2], pal.ansi[3], pal.ansi[5], pal.ansi[6]}
	partner := [4]int{1, 0, 3, 2}
	spark := rgb{255, 255, 255}
	if !pal.dark {
		spark = txt
	}
	const r, period = 11.0, 26.0
	cx, cy := math.Floor((g.x0+g.x1)/2), (g.y0+g.y1)/2
	top, hh := g.y0+6, g.hd()-12
	g.fill(bg)
	// A lattice of dots, fading to the sides, and a glow behind the helix.
	for y := g.y0 + wrap(2-g.y0, 6); y < g.y1; y += 6 {
		for x := g.x0 + wrap(3-g.x0, 6); x < g.x1; x += 6 {
			g.mix(x, y, txt, 0.07*(1-math.Abs(x-cx)/70))
		}
	}
	for y := g.y0; y < g.y1; y++ {
		v := 1.0
		if y < top {
			v = max(0, 1-(top-y)/6)
		} else if y > top+hh {
			v = max(0, 1-(y-top-hh)/6)
		}
		for x := math.Floor(cx - 26); x <= cx+26; x++ {
			d := math.Abs(x-cx) / 26
			g.mix(x, y, sec, 0.12*(1-d)*(1-d)*v*v)
		}
	}
	orbit := func(front bool) {
		for k := range 44 {
			an := float64(k)/44*6.283 + t*0.45
			if (math.Sin(an) > 0) != front {
				continue
			}
			c, a := mixRGB(bg, acc, 0.35), 0.5
			if front {
				c, a = mixRGB(bg, acc, 0.9), 0.9
			}
			g.splat(cx+30*math.Cos(an), cy+2+7*math.Sin(an)-6*math.Cos(an)*0.3, c, a)
		}
	}
	orbit(false)
	motes := []rune("·∙˙")
	for i := range int(math.Round(14 * g.hd() / 64)) {
		a := t*(0.15+0.2*hash(i, 1, 71)) + 6.3*hash(i, 2, 71)
		x := cx + (28+26*hash(i, 3, 71))*math.Cos(a)
		y := g.y0 + wrap(g.hd()*hash(i, 4, 71)-t*(1+hash(i, 5, 71)*2), g.hd())
		k := 0.32
		if math.Sin(a) < 0 {
			k = 0.18
		}
		col, row := cellOf(x, y)
		g.char(col, row, motes[i%3], mixRGB(bg, txt, k))
	}

	angle := func(y float64) float64 { return (y-top)/period*2*math.Pi + t*0.9 }
	shade := func(c rgb, z float64) rgb { return mixRGB(bg, c, 0.2+0.8*(z+1)/2) }
	strand := func(sg float64, c rgb, back bool) {
		for y := top; y < top+hh; y += 0.5 {
			th := angle(y)
			off, z := r*math.Sin(th)*sg, math.Cos(th)*sg
			if (z < 0) != back {
				continue
			}
			col, w := shade(c, z), 1+(z+1)*0.6
			for d := -w / 2; d <= w/2; d += 0.5 {
				g.splat(cx+off+d, y, col, 0.5)
			}
			if z > 0.6 {
				g.splat(cx+off-0.6, y, mixRGB(col, rgb{255, 255, 255}, 0.5), 0.25)
			}
		}
	}
	strand(1, acc, true)
	strand(-1, sec, true)
	pairs := [4][2]rune{{'A', 'T'}, {'T', 'A'}, {'C', 'G'}, {'G', 'C'}}
	for y, k := top+2, 0; y < top+hh; y, k = y+4, k+1 {
		th := angle(y)
		off, z := r*math.Sin(th), math.Cos(th)
		ao := math.Abs(off)
		if ao < 2 {
			continue
		}
		kind := pick(hash(k, 3, 72), 4)
		ca, cb := bases[kind], bases[partner[kind]]
		left, right := ca, cb
		if off > 0 {
			left, right = cb, ca
		}
		a := 0.35 + 0.45*(1-math.Abs(z))
		for x := -ao + 1; x < ao-1; x += 0.5 {
			g.splat(cx+x, y, mixRGB(left, right, (x+ao)/(2*ao)), a*0.5)
		}
		if math.Abs(math.Sin(th)) > 0.75 && k%2 == 0 {
			p, q := pairs[kind][0], pairs[kind][1]
			if off < 0 {
				p, q = q, p
			}
			c1, r1 := cellOf(cx-ao-3, y)
			c2, _ := cellOf(cx+ao+3, y)
			g.char(c1, r1, p, shade(ca, 0.6))
			g.char(c2, r1, q, shade(cb, 0.6))
		}
	}
	strand(1, acc, false)
	strand(-1, sec, false)
	n := max(1, int(math.Round(6*hh/52)))
	for k := range n {
		y := top + wrap(t*9+float64(k)*hh/float64(n), hh)
		sg := -1.0
		if k%2 == 1 {
			sg = 1
		}
		a := 0.25
		if math.Cos(angle(y))*sg > 0 {
			a = 0.7
		}
		for j := range 5 {
			fj := float64(j)
			yy := y - fj*0.8
			if yy < top {
				break
			}
			g.splat(cx+r*math.Sin(angle(yy))*sg, yy, spark, a*(1-fj/5))
		}
	}
	orbit(true)
}

// Aurora: northern lights in three curtains, rippling, over snowy ranges
// and a still lake that mirrors them, and a shooting star now and then.
var auroraAnim = &animation{id: "aurora", name: "Aurora", fps: 15, tile: ui.Hex("#1f8f6a"), draw: drawAurora}

func drawAurora(g *grid, t float64, pal animPalette) {
	ink := pal.inks(auroraRoles)
	const lake = 50.0
	g.sky(lake, ink("#02040c"), ink("#061028"), ink("#0b1a3a"), ink("#12284c"))
	g.stars(t, 0.08, 91, func(float64) float64 { return 40 }, ink("#24345e"), ink("#e8f0ff"))
	if ms := wrap(t, 12); ms < 0.7 {
		n := int(math.Floor(t / 12))
		x0, y0 := g.x0+20+(g.wd()-48)*hash(n, 1, 92), 3+6*hash(n, 2, 92)
		for j := range 10 {
			fj := float64(j)
			g.splat(x0-ms*40+fj*2.2, y0+ms*14-fj*0.8, rgb{255, 255, 255}, 1-fj/10)
		}
	}
	violet, green, mint := ink("#8a5cff"), ink("#3cff9d"), ink("#b8ffe0")
	for c := range 3 {
		fc := float64(c)
		strength := 0.75
		if c == 1 {
			strength = 1
		}
		for x := g.x0; x < g.x1; x++ {
			base := 8 + fc*6 + 5*math.Sin(x*0.035+t*0.12+fc*1.3) + 3*(noise(x/18+t*0.08, fc, 93)-0.5)*2
			in := clamp((0.5+0.5*math.Sin(x*0.22+t*1.1+fc*2+noise(x*0.08, t*0.25, 94+c)*5))*(0.55+0.45*noise(x/40, t*0.1, 95)), 0, 1)
			n := 12 + 8*noise(x/10, t*0.2, 96+c)
			for j := 0.0; j < n; j++ {
				k := j / n
				col, fade := mixRGB(green, mint, (k-0.25)/0.75), 1.0
				if k < 0.25 {
					col = mixRGB(violet, green, k/0.25)
				}
				if k < 0.2 {
					fade = k / 0.2
				}
				g.mix(x, base+j, col, in*fade*(1-k*0.55)*0.42*strength)
			}
		}
	}
	peaks := func(x float64, s int, l float64) float64 {
		return math.Pow(1-math.Abs(2*fbm(x/l, float64(s), 97+s)-1), 1.6)
	}
	ranges := []struct {
		off          float64
		s            int
		base, amp, l float64
		body, snow   rgb
	}{
		{0, 1, 44, 22, 30, ink("#1a2442"), ink("#9aaed2")},
		{40, 2, 47, 16, 22, ink("#121a32"), ink("#b4c4e2")},
		{90, 3, 50, 9, 16, ink("#0a0f1e"), ink("#c8d4ec")},
	}
	for _, rg := range ranges {
		ridge := func(x float64) float64 { return rg.base - rg.amp*peaks(x+rg.off, rg.s, rg.l) }
		snowLine := rg.base - rg.amp*0.55
		for x := g.x0; x < g.x1; x++ {
			top, facing := ridge(x), ridge(x-1)-ridge(x+1)
			for y := max(g.y0, math.Floor(top)); y < lake; y++ {
				c := mixRGB(rg.body, rgb{}, (y-top)/40)
				if high := (snowLine-y)/(rg.amp*0.45) + (noise(x/2.5, y/2, 100)-0.5)*0.8; high > 0 {
					c = mixRGB(c, rg.snow, clamp(high*1.5, 0, 1)*clamp(0.55+facing*0.5, 0.15, 1))
				}
				g.set(x, y, c)
			}
		}
	}
	water, glint := ink("#0a1426"), ink("#9fb4d8")
	for y := max(lake, g.y0); y < g.y1; y++ {
		for x := g.x0; x < g.x1; x++ {
			c := mixRGB(g.get(x+math.Round(math.Sin(y*0.9+t*0.6)*0.6), lake-1-(y-lake)*1.8), water, 0.5+(y-lake)*0.02)
			if math.Abs(math.Sin(x*0.2+y*0.9)*math.Sin(x*0.05-y*0.3)) > 0.97 {
				c = mixRGB(c, glint, 0.25)
			}
			g.set(x, y, c)
		}
	}
}

// Oscilloscope: a scope in X–Y mode tracing Lissajous figures, two sines
// in a ratio that changes every few seconds, 1:1, 1:2, 3:2, 3:4, 5:4,
// 2:3, as their phase drifts, the trace brightest where it slows and
// leaving an afterglow; its graticule behind, and its channels'
// frequencies and phase read out under it. A larger pane is a larger
// screen, of the same ten divisions by eight.
var oscilloscopeAnim = &animation{id: "oscilloscope", name: "Oscilloscope", fps: 24, tile: ui.Hex("#0f9f8f"), draw: drawOscilloscope}

// scopeRatios are the ratios of the scope's two sines, in turn.
var scopeRatios = [...][2]float64{{1, 1}, {1, 2}, {3, 2}, {3, 4}, {5, 4}, {2, 3}}

func drawOscilloscope(g *grid, t float64, pal animPalette) {
	ink := pal.inks(oscilloscopeRoles)
	var (
		screen, grat, axis = ink("#060b10"), ink("#1a2b36"), ink("#2a4252")
		trace, label       = ink("#46f08c"), ink("#56708a")
		value, trig        = ink("#7dffb2"), ink("#ffd75e")
	)
	g.fill(screen)
	// The screen is all the pane but its last row, which reads out.
	left, right, last := g.x0/2, g.x1/2, g.y1/4-1
	sx0, sx1, sy0, sy1 := g.x0, g.x1, g.y0, g.y1
	readout := g.rows >= 3
	if readout {
		sy1 = last * 4
	}
	w, h := sx1-sx0, sy1-sy0
	for k := 0.0; k <= 10; k++ {
		x, c := sx0+math.Round(k*(w-1)/10), grat
		if k == 5 {
			c = axis
		}
		for y := sy0 + 1; y < sy1; y += 2 {
			g.set(x, y, c)
		}
	}
	for k := 0.0; k <= 8; k++ {
		y, c := sy0+math.Round(k*(h-1)/8), grat
		if k == 4 {
			c = axis
		}
		for x := sx0 + 1; x < sx1; x += 2 {
			g.set(x, y, c)
		}
	}
	mid, tick := sy0+math.Round((h-1)/2), max(2, math.Round(w/50))
	for x := sx0; x < sx1; x += tick {
		g.set(x, mid-1, axis)
		g.set(x, mid+1, axis)
	}

	// The figure: x a sine of a, y of b, the ratio easing to the next in
	// the last 1.6 seconds of its 7.
	const hold = 7.0
	n := math.Floor(t / hold)
	r0 := scopeRatios[int(n)%len(scopeRatios)]
	r1 := scopeRatios[int(n+1)%len(scopeRatios)]
	m := smooth(clamp((t-n*hold-5.4)/1.6, 0, 1))
	a, b := r0[0]+(r1[0]-r0[0])*m, r0[1]+(r1[1]-r0[1])*m
	phi := t * 0.45
	cx, cy, ax, ay := (sx0+sx1)/2, (sy0+sy1)/2, w*0.42, h*0.4
	at := func(s, ph float64) (float64, float64) {
		return cx + ax*math.Sin(a*s+ph), cy - ay*math.Sin(b*s)
	}
	length := 0.0
	px, py := at(0, phi)
	for i := 1; i <= 400; i++ {
		x, y := at(float64(i)/400*2*math.Pi, phi)
		length += math.Hypot(x-px, y-py)
		px, py = x, y
	}
	// curve traces the figure at phase ph, gain of the trace's light to
	// a pixel along it, density samples a pixel: more light where it
	// slows, as the beam lingers there. The afterglow is the figure at
	// the phases just past, fainter as they are older.
	curve := func(ph, gain, density float64) {
		samples := max(300, int(length*density))
		k := gain * length / float64(samples)
		for i := range samples {
			x, y := at(float64(i)/float64(samples)*2*math.Pi, ph)
			g.light(x, y, trace, k)
			g.light(x+1, y, trace, k*0.18)
			g.light(x-1, y, trace, k*0.18)
			g.light(x, y+1, trace, k*0.18)
			g.light(x, y-1, trace, k*0.18)
		}
	}
	for k := 10.0; k >= 1; k-- {
		curve(phi-k*0.035, 0.32*math.Pow(0.78, k), 3)
	}
	curve(phi, 1, 6)

	if !readout {
		return
	}
	col := left + 1
	for _, part := range []struct {
		s string
		c rgb
	}{
		{"CH1 ", label}, {fmt.Sprintf("%.0fHz", 220*a), value},
		{"  CH2 ", label}, {fmt.Sprintf("%.0fHz", 220*b), value},
		{"  φ ", label}, {fmt.Sprintf("%.2fπ", wrap(phi/math.Pi, 2)), value},
		{"  X-Y  1V/div", label},
	} {
		if col+float64(len([]rune(part.s))) > right {
			break
		}
		col = g.text(col, last, part.s, part.c)
	}
	if right-left > 50 && int(math.Floor(t*2))%4 != 0 {
		g.text(right-7, g.y0/4, "● TRIG", trig)
	}
}

// Traceroute: a network, hosts with their addresses and links between
// them, packets running along the links; a scan sweeping across it, each
// host it passes showing a port open, a handshake flashing along a link
// now and then, a hex dump scrolling by at its edge, and the trace's
// route under it. A larger pane shows more of the network.
var tracerouteAnim = &animation{id: "traceroute", name: "Traceroute", fps: 20, tile: ui.Hex("#3b5bdb"), draw: drawTraceroute}

// traceHost is a host of the traceroute's network: its place in the
// lattice the hosts are laid on, its cell, and its pixel.
type traceHost struct {
	i, j     int
	col, row float64
	x, y     float64
}

func drawTraceroute(g *grid, t float64, pal animPalette) {
	ink := pal.inks(tracerouteRoles)
	var (
		night, link, rule = ink("#05070d"), ink("#2b3566"), ink("#1c2240")
		dim, dump         = ink("#6b7399"), ink("#4a5170")
		packet, found     = ink("#5cf2ff"), ink("#ffd75e")
	)
	g.fill(night)
	left, right, top, bottom := g.x0/2, g.x1/2, g.y0/4, g.y1/4
	status := g.rows >= 3
	mapBottom := bottom
	if status {
		mapBottom--
	}
	// The hex dump takes the right fourteen columns of a pane wide
	// enough.
	dumpAt := right
	if right-left >= 60 {
		dumpAt = right - 14
	}

	// The hosts, on a lattice of twelve columns by five rows, a few
	// missing, each with room for its label right of it.
	const sx, sy = 12, 5
	var hosts []*traceHost
	at := map[[2]int]*traceHost{}
	for j := int(math.Floor((top-2)/sy)) - 1; float64(j*sy) < mapBottom; j++ {
		for i := int(math.Floor((left-4)/sx)) - 1; float64(i*sx) < dumpAt; i++ {
			if hash(i, j, 3) > 0.85 {
				continue
			}
			col := float64(2+i*sx) + math.Floor(hash(i, j, 1)*3)
			row := float64(1+j*sy) + math.Floor(hash(i, j, 2)*2)
			if col < left || col+7 >= dumpAt || row < top || row >= mapBottom {
				continue
			}
			h := &traceHost{i: i, j: j, col: col, row: row, x: col*2 + 1, y: row*4 + 2}
			hosts = append(hosts, h)
			at[[2]int{i, j}] = h
		}
	}
	// Links and packets pass under the hosts' labels, not over them.
	labels := map[[2]int]bool{}
	for _, h := range hosts {
		for c := h.col + 1; c < h.col+8; c++ {
			labels[[2]int{int(c), int(h.row)}] = true
		}
	}
	bare := func(x, y float64) bool {
		c, r := cellOf(x, y)
		return !labels[[2]int{int(c), int(r)}]
	}
	type traceLink struct {
		a, b *traceHost
		kind int
	}
	var links []traceLink
	for _, h := range hosts {
		for kind, d := range [3]struct {
			di, dj int
			p      float64
		}{{1, 0, 0.85}, {0, 1, 0.6}, {1, 1, 0.22}} {
			if o := at[[2]int{h.i + d.di, h.j + d.dj}]; o != nil && hash(h.i*7+d.di, h.j*7+d.dj, 4) < d.p {
				links = append(links, traceLink{h, o, kind})
			}
		}
	}
	for _, l := range links {
		n := math.Ceil(math.Hypot(l.b.x-l.a.x, l.b.y-l.a.y) * 2)
		for s := 0.0; s <= n; s++ {
			if x, y := l.a.x+(l.b.x-l.a.x)*s/n, l.a.y+(l.b.y-l.a.y)*s/n; int(s)/4%2 == 0 && bare(x, y) {
				g.mix(x, y, link, 0.8)
			}
		}
	}
	// Packets, one or two a link, each its own way and pace.
	for _, l := range links {
		e, f := l.a.i*3+l.kind, l.a.j
		np := 1
		if hash(e, f, 5) < 0.45 {
			np = 2
		}
		for q := range np {
			sp, back := 0.16+0.3*hash(e, f*2+q, 6), hash(e, f*2+q, 7) < 0.5
			for k := range 6 {
				p := wrap(hash(e, f*2+q, 8)+t*sp-float64(k)*0.012, 1)
				if back {
					p = 1 - p
				}
				x, y := l.a.x+(l.b.x-l.a.x)*p, l.a.y+(l.b.y-l.a.y)*p
				if !bare(x, y) {
					continue
				}
				if k == 0 {
					g.light(x, y, packet, 1)
				} else {
					g.mix(x, y, packet, 0.5*(1-float64(k)/6))
				}
			}
		}
	}
	// A handshake along a link, every 1.4 seconds.
	if hk := math.Floor(t / 1.4); len(links) > 0 && t-hk*1.4 < 0.6 {
		l := links[pick(hash(int(hk), 0, 77), len(links))]
		n := math.Ceil(math.Hypot(l.b.x-l.a.x, l.b.y-l.a.y) * 1.5)
		for s := 0.0; s <= n; s++ {
			if x, y := l.a.x+(l.b.x-l.a.x)*s/n, l.a.y+(l.b.y-l.a.y)*s/n; bare(x, y) {
				g.splat(x, y, packet, 0.7*(1-(t-hk*1.4)/0.6))
			}
		}
	}
	// The scan sweeps the map in 7 seconds of every 11, and the hosts it
	// passed show a port open till it starts again.
	const sweepFor, scanEvery = 7.0, 11.0
	local, cycle := wrap(t, scanEvery), int(math.Floor(t/scanEvery))
	sweep := g.x0 + min(1, local/sweepFor)*(dumpAt*2-g.x0)
	if local < sweepFor {
		for y := top * 4; y < mapBottom*4; y++ {
			g.light(sweep, y, packet, 0.4)
			for d := 1.0; d < 10; d++ {
				g.mix(sweep-d, y, packet, 0.1*(1-d/10))
			}
		}
	}
	ports := [...]string{":22", ":80", ":443", ":8080", ":5432"}
	for _, h := range hosts {
		lit := h.x < sweep
		if lit {
			g.glow(h.x, h.y, 4, found, 0.15)
			g.text(h.col, h.row, "◆", found)
			g.text(h.col+2, h.row, ports[pick(hash(h.i, h.j, cycle+40), len(ports))], mixRGB(found, night, 0.25))
			continue
		}
		g.text(h.col, h.row, "◇", dim)
		g.text(h.col+2, h.row, fmt.Sprintf("%02x.%02x", int(hash(h.i, h.j, 20)*255), int(hash(h.i, h.j, 21)*255)), dim)
	}
	if dumpAt < right {
		for y := g.y0; y < g.y1; y++ {
			g.set(dumpAt*2, y, rule)
		}
		for row := top; row < mapBottom; row++ {
			line := int(row) + int(math.Floor(t*2.5))
			c := dump
			if hash(line, 9, 91) < 0.12 {
				c = packet
			}
			g.text(dumpAt+2, row, fmt.Sprintf("%02x %02x %02x %02x", int(hash(line, 0, 90)*255), int(hash(line, 1, 90)*255),
				int(hash(line, 2, 90)*255), int(hash(line, 3, 90)*255)), c)
		}
	}
	if status && right-left > 4 {
		k := int(math.Floor(t / 2.2))
		q := func(b int) int { return 2 + int(hash(k, b, 60)*250) }
		s := []rune(fmt.Sprintf("trace 10.%d.%d.%d  hops %d  rtt %dms", q(1), q(2), q(3),
			4+int(hash(k, 4, 60)*9), 12+int(hash(k, 5, 60)*30)))
		s = s[:min(len(s), int(right-left)-4)]
		g.text(left+1, bottom-1, "›", packet)
		g.text(left+3, bottom-1, string(s), dim)
	}
}
