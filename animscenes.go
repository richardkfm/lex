package main

import (
	"math"

	"github.com/egoist/mygo/ui"
)

// The animations an animation pane plays (animations.go): small scenes,
// each composed round what it shows, on the ground of the pane's bottom
// and centered in it, a larger pane showing more of the sky round them
// rather than more of them. Their figures are drawn pixel by pixel
// (sprite), and live a little: they walk, turn, blink, peek, sway and
// fly, and lights flicker. Each is painted from the time and the pane's
// size alone, w×h pixels, two to a cell, with characters over the cells.

// sprite paints art, rows of characters, its top left at (x, y): each
// character a pixel of the color pal gives it, those it has none for
// none; flip mirrors it left to right.
func (g *grid) sprite(x, y int, art []string, pal map[byte]rgb, flip bool) {
	for j, row := range art {
		for i := 0; i < len(row); i++ {
			c, ok := pal[row[i]]
			if !ok {
				continue
			}
			if flip {
				g.set(x+len(row)-1-i, y+j, c)
			} else {
				g.set(x+i, y+j, c)
			}
		}
	}
}

// twinkles puts stars over the cells whose pixels are above sky(col):
// dots, a few brighter, each twinkling at its pace.
func (g *grid) twinkles(t float64, density float64, seed int, sky func(x int) int, dim, bright rgb) {
	for row := range g.rows {
		for col := range g.cols {
			h := hash(col, row, seed)
			if h > density || row*2+1 >= sky(col) {
				continue
			}
			b := 0.55 + 0.45*math.Sin(t*(0.8+2*hash(col, row, seed+1))+6.3*hash(col, row, seed+2))
			switch {
			case h < density*0.08:
				g.char(col, row, '✦', mixRGB(dim, bright, b))
			case h < density*0.2 && b > 0.5:
				g.char(col, row, '+', mixRGB(dim, bright, b*0.8))
			default:
				g.char(col, row, '·', mixRGB(dim, bright, b*0.7))
			}
		}
	}
}

// Sahara: a caravan crossing the crest of a dune at night under a large
// moon, three camels with their riders, roped one to the next, their
// tracks behind them; a fennec fox peeks over the dune in front now and
// then, a star falls, sand blows off a crest.
var saharaAnim = &animation{id: "sahara", name: "Sahara", fps: 12, tile: ui.Hex("#c27a3a"), draw: drawSahara}

var (
	camelArt = []string{
		"............hhh.",
		"...........hBBBm",
		"...........BB...",
		".....hhh...B....",
		"....hBBBh..B....",
		"...hBSSSBhBB....",
		"..hBBSySBBBB....",
		".tBBBBBBBBBB....",
		".tBBBBBBBBB.....",
		"..BBBBBBBB......",
	}
	riderArt = []string{
		"..T..",
		".TTT.",
		"..f..",
		".RRR.",
		"RRRRR",
	}
	camelPal = map[byte]rgb{'B': hexRGB("#2a2131"), 'h': hexRGB("#7d6d8c"), 'm': hexRGB("#3e3242"), 't': hexRGB("#2a2131"),
		'S': hexRGB("#7a2f44"), 'y': hexRGB("#c9a347"), 'T': hexRGB("#4a5aa6"), 'f': hexRGB("#5a4440"), 'R': hexRGB("#313a78")}
	foxArt = []string{
		".E.....E.",
		".EE...EE.",
		".EiF.FiE.",
		"..FFFFF..",
		"..FwFwF..",
		"...FnF...",
		"..FFFFF..",
	}
	foxPal = map[byte]rgb{'E': hexRGB("#8f7660"), 'i': hexRGB("#b07a72"), 'F': hexRGB("#a68a6e"), 'w': hexRGB("#f2eadc"), 'n': hexRGB("#2a2026")}
)

func drawSahara(g *grid, t float64, _ animPalette) {
	W, H := g.w, g.h
	cx := W / 2
	fx := func(x int, a, f, p float64) float64 { return a * math.Sin(float64(x-cx)*f+p) }
	far := func(x int) int { return H - 25 + int(fx(x, 3, 0.07, 1)+fx(x, 1.5, 0.19, 0)) }
	mid := func(x int) int { return H - 15 + int(fx(x, 2.2, 0.045, 0.5)+fx(x, 1, 0.13, 2)) }
	near := func(x int) int { return H - 5 + int(fx(x, 1.8, 0.09, 3)+fx(x, 0.8, 0.23, 0)) }
	g.gradient(0, H, hexRGB("#060918"), hexRGB("#0c1130"), hexRGB("#1a1b40"), hexRGB("#2c2548"))
	g.twinkles(t, 0.07, 400, far, hexRGB("#3a3d6a"), hexRGB("#dcdcff"))
	// The moon, its light round it.
	mx, my := float64(cx+22), math.Max(float64(H-40), 8)
	g.glow(mx, my, 18, hexRGB("#3d3f72"), 0.55)
	g.glow(mx, my, 10, hexRGB("#8d89b8"), 0.35)
	g.disc(mx, my, 5.6, hexRGB("#efe6c8"))
	g.disc(mx-2, my-1, 1.3, hexRGB("#d6cba8"))
	g.disc(mx+2, my+2, 1, hexRGB("#d6cba8"))
	g.set(int(mx)+1, int(my)-3, hexRGB("#d6cba8"))
	// Now and then a star falls.
	if s := wrap(t, 10); s < 0.9 {
		n := int(math.Floor(t / 10))
		x0, y0 := float64(W)*(0.25+0.55*hash(n, 1, 401)), 2+5*hash(n, 2, 401)
		hx, hy := x0-26*s/0.9, y0+9*s/0.9
		for k := range 7 {
			g.mix(int(hx+float64(k)*2.9), int(hy-float64(k)), hexRGB("#ffffff"), 1-float64(k)/7)
		}
	}
	// The dunes, lit on the side of the moon, far to near.
	dune := func(ridge func(int) int, lit, shade, crest rgb) {
		for x := range W {
			top := ridge(x)
			c := lit
			if ridge(x+1) < top {
				c = shade
			}
			for y := top; y < H; y++ {
				g.set(x, y, mixRGB(c, shade, float64(y-top)/18))
			}
			g.set(x, top, crest)
		}
	}
	dune(far, hexRGB("#3d3558"), hexRGB("#2a2542"), hexRGB("#5a4f78"))
	dune(mid, hexRGB("#4b3f62"), hexRGB("#332b4a"), hexRGB("#75668f"))
	// The caravan, walking right along the crest, roped one to the next,
	// its tracks behind it.
	// It crosses the dune in front of the moon at the still frame.
	lead := int(wrap((t-animStill)*2.2+float64(cx+38), float64(W+70))) - 18
	for i := range 3 {
		x := lead - i*19
		feet := mid(x + 8)
		top := feet - 13
		for _, leg := range []struct{ x, ph float64 }{{3, 0}, {5, math.Pi}, {8, math.Pi}, {10, 0}} {
			swing := math.Round(math.Sin(t*5+float64(i)*1.3+leg.ph) * 1.2)
			hip := top + 10
			for y := hip; y <= feet; y++ {
				k := float64(y-hip) / float64(max(feet-hip, 1))
				g.set(x+int(leg.x+swing*k), y, camelPal['B'])
			}
		}
		g.sprite(x, top, camelArt, camelPal, false)
		g.sprite(x+3, top-5, riderArt, camelPal, false)
		if i > 0 {
			// The rope from this camel's head to the saddle ahead.
			ax, ay := x+15, top+2
			bx, by := lead-(i-1)*19+2, mid(lead-(i-1)*19+8)-7
			for k := 0; k <= 8; k++ {
				u := float64(k) / 8
				g.set(int(float64(ax)+float64(bx-ax)*u), int(float64(ay)+float64(by-ay)*u+2*math.Sin(math.Pi*u)), hexRGB("#6f5f52"))
			}
		}
	}
	for k := 1; k <= 16; k++ {
		x := lead - 2*19 - 2*k
		g.mix(x, mid(x), hexRGB("#332b4a"), 0.7*(1-float64(k)/16))
	}
	// The fox, peeking over the near dune now and then, blinking.
	foxX := cx + 18
	rise := 0.0
	s := wrap(t, 14)
	switch {
	case s > 6 && s < 6.6:
		rise = (s - 6) / 0.6
	case s >= 6.6 && s < 11:
		rise = 1
	case s >= 11 && s < 11.6:
		rise = 1 - (s-11)/0.6
	}
	if rise > 0 {
		art := foxArt
		if s > 8.2 && s < 8.4 {
			art = append([]string(nil), foxArt...)
			art[4] = "..FFFFF.." // a blink
		}
		g.sprite(foxX, near(foxX+4)-int(math.Round(6*rise)), art, foxPal, false)
	}
	dune(near, hexRGB("#5d4c69"), hexRGB("#3c3250"), hexRGB("#8a789e"))
	// Sand blowing off the near crest.
	for i := range 3 {
		s := wrap(t*0.5+float64(i)*1.7, 6)
		if s > 2 {
			continue
		}
		x := int(wrap(float64(W)*hash(i, int(t/12), 402)+s*14, float64(W)))
		g.char(x, (near(x)-2)/2, '~', mixRGB(hexRGB("#2c2548"), hexRGB("#a493b5"), 1-s/2))
	}
}

// Neon City: a rooftop at night over the city, a figure leaning on its
// railing, breathing out in the cold, a satellite dish turning slowly, an
// air conditioner's fan spinning, a holographic koi swimming in the air
// over the street, drones blinking past, the towers lit window by window.
var neonCityAnim = &animation{id: "neon-city", name: "Neon City", fps: 15, tile: ui.Hex("#c2187a"), draw: drawNeonCity}

var (
	figureArt = []string{
		"..hh...",
		".hhhh..",
		".hhss..",
		"..ss...",
		".CCCC..",
		"CCCCCaa",
		"CCCCC..",
		"CCCCC..",
		"CCCC...",
		"CCCC...",
		".CCC...",
		".LL....",
		".L.L...",
		".bb.bb.",
	}
	figurePal = map[byte]rgb{'h': hexRGB("#1a1622"), 's': hexRGB("#b9887a"), 'C': hexRGB("#26243a"), 'a': hexRGB("#26243a"),
		'L': hexRGB("#1c1b2a"), 'b': hexRGB("#121119")}
	koiArt = [2][]string{{
		"....dd.....",
		"t..cmmcCCc.",
		"TTcCCCCmCCe",
		"t..cCCmCCc.",
		"....ff.....",
	}, {
		"....dd.....",
		"...cmmcCCc.",
		"tTcCCCCmCCe",
		"...cCCmCCc.",
		"t...ff.....",
	}}
	koiPal = map[byte]rgb{'C': hexRGB("#7ff6ff"), 'c': hexRGB("#3fc9e6"), 'm': hexRGB("#ff6ad5"), 'd': hexRGB("#b98cff"),
		'f': hexRGB("#b98cff"), 't': hexRGB("#ff8ae0"), 'T': hexRGB("#ff8ae0"), 'e': hexRGB("#ffffff")}
	dishArt = [3][]string{{
		".DD..",
		"DdDD.",
		"DdDDf",
		"DdDD.",
		".DD..",
	}, {
		"..D..",
		".DDD.",
		"DdddD",
		"..f..",
		"..s..",
	}, {
		"..DD.",
		".DDdD",
		"fDDdD",
		".DDdD",
		"..DD.",
	}}
	dishPal = map[byte]rgb{'D': hexRGB("#9aa0b8"), 'd': hexRGB("#5d6178"), 'f': hexRGB("#c7cbe0"), 's': hexRGB("#5d6178")}
)

func drawNeonCity(g *grid, t float64, _ animPalette) {
	W, H := g.w, g.h
	cx := W / 2
	parapet := H - 11
	g.gradient(0, H, hexRGB("#05061a"), hexRGB("#0d0a25"), hexRGB("#1c1036"), hexRGB("#35134b"))
	g.twinkles(t, 0.025, 410, func(int) int { return H / 3 }, hexRGB("#2a2350"), hexRGB("#8f86c9"))
	// The towers beyond the railing, their feet below it.
	lit := []rgb{hexRGB("#ffc46b"), hexRGB("#7fdcff"), hexRGB("#ff7ad9")}
	for x, i := cx-90, 0; x < W; i++ {
		w := 5 + int(8*hash(i, 1, 411))
		top := parapet - 6 - int(float64(min(H-12, 34))*hash(i, 2, 411))
		body := mixRGB(hexRGB("#0e0c1c"), hexRGB("#1a1430"), hash(i, 3, 411))
		for y := top; y < parapet; y++ {
			for xx := x; xx < x+w; xx++ {
				g.set(xx, y, body)
			}
			g.set(x, y, mixRGB(body, hexRGB("#3b2a5e"), 0.5))
		}
		for y := top + 2; y < parapet; y += 2 {
			for xx := x + 1; xx < x+w-1; xx += 2 {
				if hash(xx, y, 412) < 0.28 {
					on := true
					if hash(xx, y, 413) < 0.1 {
						on = hash(xx, y+int(t/3), 414) < 0.5
					}
					if on {
						g.set(xx, y, mixRGB(body, lit[int(hash(xx, y, 415)*3)], 0.35+0.6*hash(xx, y, 416)))
					}
				}
			}
		}
		if hash(i, 4, 411) < 0.35 && hash(i, int(t*15), 417) > 0.04 {
			c := lit[1+int(hash(i, 5, 411)*2)]
			for y := top + 1; y < parapet; y++ {
				g.set(x+w-1, y, c)
			}
		}
		if hash(i, 6, 411) < 0.3 && wrap(t+hash(i, 7, 411)*3, 2) < 0.7 {
			g.set(x+w/2, top-1, hexRGB("#ff3045"))
		}
		x += w + 1 + int(3*hash(i, 8, 411))
	}
	// The haze over the street.
	for y := parapet - 8; y < parapet; y++ {
		for x := range W {
			g.mix(x, y, hexRGB("#3a1650"), 0.05*float64(y-parapet+8))
		}
	}
	// Drones, blinking past.
	for i := range 2 {
		a := t*0.25 + float64(i)*3
		x := float64(cx) + 34*math.Sin(a)
		y := float64(parapet) - 18 - 8*float64(i) + 3*math.Sin(a*2)
		g.rect(int(x)-1, int(y), int(x)+2, int(y)+1, hexRGB("#3a3d58"))
		if wrap(t*2+float64(i), 1) < 0.5 {
			g.set(int(x)-1, int(y), hexRGB("#ff4a5a"))
		} else {
			g.set(int(x)+1, int(y), hexRGB("#4aff9a"))
		}
	}
	// The koi, swimming a slow loop in the air over the street, as a
	// hologram: its rows alternately fainter, flickering now and then.
	kx := float64(cx+4) + 18*math.Sin(t*0.32)
	ky := float64(parapet) - 13 + 3*math.Sin(t*0.64)
	left := math.Cos(t*0.32) < 0
	alpha := 0.8
	if hash(int(t*15), 0, 418) < 0.06 {
		alpha = 0.3
	}
	art := koiArt[int(t*4)%2]
	for j, row := range art {
		for i := 0; i < len(row); i++ {
			c, ok := koiPal[row[i]]
			if !ok {
				continue
			}
			x := int(kx) + i
			if left {
				x = int(kx) + len(row) - 1 - i
			}
			a := alpha
			if (int(ky)+j)%2 == 1 {
				a *= 0.6
			}
			g.mix(x, int(ky)+j, c, a)
		}
	}
	for k := 1; k <= 3; k++ {
		tx := float64(cx+4) + 18*math.Sin(t*0.32-float64(k)*0.12) + 5
		ty := float64(parapet) - 11 + 3*math.Sin(t*0.64-float64(k)*0.24)
		g.char(int(tx), int(ty)/2, '·', mixRGB(hexRGB("#1c1036"), hexRGB("#9ff6ff"), 0.8-float64(k)*0.2))
	}
	// The railing, at the far edge of the roof.
	for x := range W {
		g.set(x, parapet, hexRGB("#3d3952"))
		g.set(x, parapet+1, hexRGB("#2a2738"))
		g.set(x, parapet-5, hexRGB("#4b4763"))
		if x%7 == 3 {
			for y := parapet - 4; y < parapet; y++ {
				g.set(x, y, hexRGB("#3a3650"))
			}
		}
	}
	// The roof, wet, a puddle mirroring the koi's colors.
	for y := parapet + 2; y < H; y++ {
		for x := range W {
			g.set(x, y, mixRGB(hexRGB("#141220"), hexRGB("#1d1a2b"), noise(float64(x)/5, float64(y)/2, 419)))
		}
	}
	px, py := float64(cx+8), float64(H-3)
	for y := int(py - 2); y <= int(py+2); y++ {
		for x := int(px - 9); x <= int(px+9); x++ {
			if math.Hypot((float64(x)-px)/9, (float64(y)-py)/2) < 1 {
				k := 0.5 + 0.5*math.Sin(float64(x)*0.6+t*2)
				g.set(x, y, mixRGB(hexRGB("#2a2140"), mixRGB(hexRGB("#ff6ad5"), hexRGB("#7ff6ff"), k), 0.35))
			}
		}
	}
	// The air conditioner, its fan spinning.
	ax, ay := cx-30, H-9
	g.rect(ax, ay, ax+9, ay+7, hexRGB("#383848"))
	g.rect(ax, ay, ax+9, ay+1, hexRGB("#4a4a5c"))
	g.rect(ax+2, ay+2, ax+7, ay+6, hexRGB("#1f1f2a"))
	if int(t*12)%2 == 0 {
		g.set(ax+3, ay+3, hexRGB("#5a5a6e"))
		g.set(ax+5, ay+4, hexRGB("#5a5a6e"))
	} else {
		g.set(ax+5, ay+3, hexRGB("#5a5a6e"))
		g.set(ax+3, ay+4, hexRGB("#5a5a6e"))
	}
	// The satellite dish on its mast, turning slowly, its light blinking.
	dx := cx + 24
	for y := H - 13; y < H-1; y++ {
		g.set(dx+2, y, hexRGB("#4a4d62"))
	}
	g.sprite(dx, H-17, dishArt[[]int{0, 1, 2, 1}[int(wrap(t/3, 4))]], dishPal, false)
	if wrap(t, 1.4) < 0.5 {
		g.set(dx+2, H-18, hexRGB("#ff3045"))
	}
	// The figure, leaning on the railing, lit by the city: magenta from
	// behind, cyan from the street; breathing out now and then.
	fxx, fy := cx-10, H-14
	fig := figureArt
	if wrap(t, 11) < 2 {
		fig = append([]string(nil), figureArt...)
		fig[2] = ".hsss.." // turned toward the city
	}
	if int(t/1.2)%2 == 1 {
		fig = append([]string(nil), fig...)
		fig[10] = "..CCC.." // the coat, swaying
	}
	g.sprite(fxx, fy, fig, figurePal, false)
	for j, row := range fig {
		for i := 0; i < len(row); i++ {
			if row[i] == '.' {
				continue
			}
			if i == 0 || row[i-1] == '.' {
				g.mix(fxx+i, fy+j, hexRGB("#c13fa0"), 0.45)
			}
			if i == len(row)-1 || row[i+1] == '.' {
				g.mix(fxx+i, fy+j, hexRGB("#3fb7d9"), 0.4)
			}
		}
	}
	if s := wrap(t, 5); s < 1.6 {
		for k := range 3 {
			g.mix(fxx+5+int(s*3)+k, fy+2-int(s*2)-k/2, hexRGB("#c8c8e0"), 0.5*(1-s/1.6))
		}
	}
}

// Solarpunk: a village in a great tree at dusk, homes on its branches,
// one with a solar roof and a spinner turning, a rope bridge to the next
// tree hung with lanterns, a gondola gliding up and down its cable, a
// bird coming to land and flying off, a waterfall falling into a pond,
// fireflies.
var solarpunkAnim = &animation{id: "solarpunk", name: "Solarpunk", fps: 15, tile: ui.Hex("#2e9b57"), draw: drawSolarpunk}

func drawSolarpunk(g *grid, t float64, _ animPalette) {
	W, H := g.w, g.h
	cx := W / 2
	gy := H - 5
	tx := cx - 8
	g.gradient(0, gy, hexRGB("#1d2a52"), hexRGB("#3e4580"), hexRGB("#8a6890"), hexRGB("#e0907a"), hexRGB("#ffcf92"))
	g.twinkles(t, 0.03, 420, func(int) int { return gy / 3 }, hexRGB("#2e3a6e"), hexRGB("#e6e6ff"))
	g.glow(float64(cx+30), float64(gy-3), 22, hexRGB("#ffd49a"), 0.45)
	// The hills far off.
	for x := range W {
		top := gy - 7 - int(4*noise(float64(x)/14, 0, 421))
		for y := top; y < gy; y++ {
			g.set(x, y, mixRGB(hexRGB("#4a5578"), hexRGB("#3a4766"), float64(y-top)/6))
		}
	}
	// The ground, its flowers, and the pond, rippling.
	for y := gy; y < H; y++ {
		for x := range W {
			g.set(x, y, mixRGB(hexRGB("#2f5d3a"), hexRGB("#244a2f"), float64(y-gy)/5))
		}
	}
	for i := range W / 6 {
		x := int(float64(W) * hash(i, 1, 422))
		g.set(x, gy+1+int(3*hash(i, 2, 422)), []rgb{hexRGB("#f2c94c"), hexRGB("#e88aa8"), hexRGB("#f4f1e8")}[i%3])
	}
	px, py := float64(cx+26), float64(H-2)
	for y := gy; y < H; y++ {
		for x := int(px - 11); x <= int(px+11); x++ {
			if math.Hypot((float64(x)-px)/11, (float64(y)-py)/3) < 1 {
				g.set(x, y, hexRGB("#3c78a6"))
			}
		}
	}
	for k := range 2 {
		r := wrap(t*2+float64(k)*2.5, 5)
		for a := 0.0; a < math.Pi*2; a += 0.35 {
			g.mix(int(px-6+r*1.6*math.Cos(a)), int(py+r*0.45*math.Sin(a)), hexRGB("#9fd0ee"), 0.6*(1-r/5))
		}
	}
	// The cliff at the right, and the waterfall down it.
	wx := cx + 30
	for y := gy - 18; y < gy+1; y++ {
		for x := wx; x < wx+12; x++ {
			g.set(x, y, mixRGB(hexRGB("#4c4760"), hexRGB("#635c78"), noise(float64(x)/2, float64(y)/2, 423)))
		}
	}
	for x := wx; x < wx+12; x++ {
		g.set(x, gy-19, hexRGB("#3f7a46"))
	}
	for y := gy - 18; y < H-2; y++ {
		for x := wx - 3; x < wx; x++ {
			k := int(wrap(float64(y)-t*14+float64(x), 5))
			g.set(x, y, []rgb{hexRGB("#e6f6ff"), hexRGB("#a9dcf4"), hexRGB("#7fbde3"), hexRGB("#a9dcf4"), hexRGB("#cfeefc")}[k])
		}
	}
	if int(t*6)%2 == 0 {
		g.char(wx-4, (H-3)/2, '°', hexRGB("#e6f6ff"))
	} else {
		g.char(wx-1, (H-4)/2, '·', hexRGB("#e6f6ff"))
	}
	canopy := func(x, y, r float64) {
		g.disc(x, y, r, hexRGB("#1f4a2e"))
		g.disc(x-r*0.25, y-r*0.25, r*0.75, hexRGB("#2e6b3c"))
		g.disc(x-r*0.45, y-r*0.45, r*0.35, hexRGB("#4f8f4b"))
		for i := range int(r * r) {
			ax, ay := x+(hash(i, 1, int(x))-0.5)*r*1.6, y+(hash(i, 2, int(y))-0.5)*r*1.6
			if math.Hypot(ax-x, ay-y) < r-0.5 {
				g.set(int(ax), int(ay), hexRGB("#3a7d43"))
			}
		}
	}
	// The second tree, at the right, with its platform.
	rx := cx + 17
	g.rect(rx-1, gy-12, rx+1, gy, hexRGB("#5b3b28"))
	canopy(float64(rx), float64(gy-17), 6)
	g.rect(rx-4, gy-12, rx+5, gy-11, hexRGB("#7a4f30"))
	// The great tree: roots, trunk, branches and canopy.
	for y := gy - 24; y < gy; y++ {
		w := 3 + (y-(gy-24))/8
		for x := tx - w; x <= tx+w; x++ {
			c := hexRGB("#5b3b28")
			switch {
			case x == tx-w:
				c = hexRGB("#7a5236")
			case x == tx+w:
				c = hexRGB("#3e281c")
			case hash(x, y, 424) < 0.15:
				c = hexRGB("#4a3020")
			}
			g.set(x, y, c)
		}
	}
	for _, r := range [][2]int{{-6, 0}, {-5, -1}, {6, 0}, {5, -1}, {-7, 0}} {
		g.set(tx+r[0], gy+r[1], hexRGB("#4a3020"))
	}
	g.line(float64(tx-2), float64(gy-14), float64(tx-13), float64(gy-21), hexRGB("#5b3b28"))
	g.line(float64(tx-2), float64(gy-13), float64(tx-13), float64(gy-20), hexRGB("#5b3b28"))
	g.line(float64(tx+2), float64(gy-17), float64(tx+12), float64(gy-24), hexRGB("#5b3b28"))
	g.line(float64(tx+2), float64(gy-16), float64(tx+12), float64(gy-23), hexRGB("#5b3b28"))
	for _, c := range [][3]float64{{0, -31, 11}, {-12, -26, 8}, {11, -28, 9}, {-5, -38, 8}, {6, -37, 7}} {
		canopy(float64(tx)+c[0], float64(gy)+c[1], c[2])
	}
	lamp := func(x, y int, k float64) {
		f := 0.75 + 0.25*noise(t*3+k, 0.5, 425)
		g.glow(float64(x)+0.5, float64(y)+0.5, 4, hexRGB("#ffcf7a"), 0.35*f)
		g.set(x, y, mixRGB(hexRGB("#ff9a4a"), hexRGB("#ffd27a"), f))
	}
	// The left home, on its branch: plank walls, a mossy roof, a lit
	// round window and a door.
	hx, hy := tx-15, gy-20
	g.rect(hx-1, hy+5, hx+11, hy+6, hexRGB("#7a4f30"))
	g.rect(hx, hy, hx+9, hy+5, hexRGB("#b07a4e"))
	for y := hy + 1; y < hy+5; y += 2 {
		g.rect(hx, y, hx+9, y+1, hexRGB("#9a6842"))
	}
	for r := range 4 {
		g.rect(hx-1+r, hy-1-r, hx+10-r, hy-r, hexRGB("#8a3b2e"))
		g.set(hx+2+r*2, hy-1-r, hexRGB("#4b7d3e"))
	}
	lamp(hx+2, hy+2, 0)
	lamp(hx+3, hy+2, 0.3)
	g.rect(hx+6, hy+2, hx+8, hy+5, hexRGB("#5a3a26"))
	// The right home, higher: a solar roof, glinting, and a spinner.
	h2x, h2y := tx+6, gy-30
	g.rect(h2x-1, h2y+4, h2x+8, h2y+5, hexRGB("#7a4f30"))
	g.rect(h2x, h2y, h2x+7, h2y+4, hexRGB("#c08a5a"))
	g.rect(h2x-1, h2y-1, h2x+8, h2y, hexRGB("#2a4a78"))
	g.set(h2x+int(wrap(t*2, 9))-1, h2y-1, hexRGB("#cfe6ff"))
	lamp(h2x+2, h2y+2, 1)
	sp := int(t*6) % 4
	dxs, dys := []int{0, 1, 0, -1}[sp], []int{-1, 0, 1, 0}[sp]
	g.set(h2x+6, h2y-3, hexRGB("#e8e8e8"))
	g.set(h2x+6+dxs, h2y-3+dys, hexRGB("#e8e8e8"))
	g.set(h2x+6-dxs, h2y-3-dys, hexRGB("#e8e8e8"))
	g.set(h2x+6, h2y-2, hexRGB("#8a8a8a"))
	// The rope bridge to the second tree, sagging, lanterns along it.
	bx0, by0, bx1, by1 := tx+5, gy-14, rx-4, gy-12
	for x := bx0; x <= bx1; x++ {
		u := float64(x-bx0) / float64(max(bx1-bx0, 1))
		y := int(float64(by0) + float64(by1-by0)*u + 3*math.Sin(math.Pi*u))
		if x%2 == 0 {
			g.set(x, y, hexRGB("#8a5a3a"))
		}
		g.set(x, y-3, hexRGB("#d6be8e"))
	}
	for k, u := range []float64{0.25, 0.55, 0.85} {
		x := bx0 + int(float64(bx1-bx0)*u)
		y := int(float64(by0)+float64(by1-by0)*u+3*math.Sin(math.Pi*u)) - 2
		g.set(x, y, hexRGB("#d6be8e"))
		lamp(x, y+1, float64(k)+2)
	}
	lamp(tx-12, gy-19, 5)
	// The gondola, gliding along its cable from the left home to the
	// ground and back.
	cx0, cy0, cx1, cy1 := float64(hx-1), float64(hy+3), float64(cx-36), float64(gy-3)
	for k := 0; k <= 200; k++ {
		u := float64(k) / 200
		g.set(int(cx0+(cx1-cx0)*u), int(cy0+(cy1-cy0)*u+2*math.Sin(math.Pi*u)), hexRGB("#3a3a4a"))
	}
	g.rect(int(cx1)-3, gy-3, int(cx1)+3, gy, hexRGB("#8a5a3a"))
	g.rect(int(cx1)-4, gy-4, int(cx1)+4, gy-3, hexRGB("#6b3a2a"))
	s := wrap(t, 20)
	u := 0.0
	switch {
	case s < 8:
		u = smooth(s / 8)
	case s < 10:
		u = 1
	case s < 18:
		u = 1 - smooth((s-10)/8)
	}
	gxp := cx0 + (cx1-cx0)*u
	gyp := cy0 + (cy1-cy0)*u + 2*math.Sin(math.Pi*u)
	g.set(int(gxp), int(gyp)+1, hexRGB("#3a3a4a"))
	g.rect(int(gxp)-2, int(gyp)+2, int(gxp)+2, int(gyp)+3, hexRGB("#3a3a4a"))
	g.rect(int(gxp)-2, int(gyp)+3, int(gxp)+2, int(gyp)+6, hexRGB("#c4503f"))
	g.rect(int(gxp)-1, int(gyp)+4, int(gxp)+1, int(gyp)+5, hexRGB("#ffe6a8"))
	// A bird flies in, lands on the right branch, hops, and flies off.
	lx, ly := float64(tx+11), float64(gy-25)
	bs := wrap(t, 16)
	var bxp, byp float64
	flying := true
	switch {
	case bs < 4:
		k := bs / 4
		bxp, byp = float64(W)+4+(lx-float64(W)-4)*k, ly-10+10*smooth(k)
	case bs < 10:
		flying = false
		hop := math.Max(0, math.Sin((bs-4)*math.Pi))
		bxp, byp = lx-math.Floor((bs-4)/2), ly-math.Round(hop)
	case bs < 14:
		k := (bs - 10) / 4
		bxp, byp = lx-3-k*(lx+10), ly-k*k*14
	default:
		bxp = -20
	}
	bird := hexRGB("#2a2a3a")
	if flying {
		g.set(int(bxp), int(byp), bird)
		if int(t*8)%2 == 0 {
			g.set(int(bxp)-1, int(byp)-1, bird)
			g.set(int(bxp)+1, int(byp)-1, bird)
		} else {
			g.set(int(bxp)-1, int(byp), bird)
			g.set(int(bxp)+1, int(byp), bird)
		}
	} else {
		g.rect(int(bxp), int(byp)-1, int(bxp)+2, int(byp)+1, bird)
		g.set(int(bxp)-1, int(byp)-1, hexRGB("#e0a040"))
	}
	// Fireflies.
	for i := range 7 {
		x := float64(tx) + 30*(hash(i, 1, 426)-0.5) + 3*math.Sin(t*0.7+float64(i))
		y := float64(gy) - 2 - 20*hash(i, 2, 426) + 2*math.Sin(t*0.5+float64(i)*2)
		if math.Sin(t*2+float64(i)*1.7) > 0.2 {
			g.char(int(x), int(y)/2, '·', hexRGB("#ffe680"))
		}
	}
}

// Helix: a double helix, small and fine, turning slowly in the dark, its
// strands in the theme's accent and second color, the nearer brighter, a
// pulse running down them, the rungs between them characters lettered at
// their ends, motes drifting round it.
var helixAnim = &animation{id: "helix", name: "Helix", fps: 20, tile: ui.Hex("#6a4cc2"), draw: drawHelix}

func drawHelix(g *grid, t float64, pal animPalette) {
	W, H := g.w, g.h
	cx := float64(W) / 2
	g.rect(0, 0, W, H, pal.bg)
	glowC := mixRGB(pal.bg, pal.second, 0.5)
	// The helix, as tall as the pane allows, within reason.
	hh := min(max(float64(H)-8, 16), 56)
	top := (float64(H) - hh) / 2
	r := min(max(hh*0.17, 4), 9)
	period := hh / 2.1
	for y := int(top) - 6; y < int(top+hh)+6; y++ {
		v := 1.0
		if d := math.Max(top-float64(y), float64(y)-top-hh); d > 0 {
			v = math.Max(0, 1-d/6)
		}
		for x := int(cx - r*2.4); x <= int(cx+r*2.4); x++ {
			d := math.Abs(float64(x)-cx) / (r * 2.4)
			g.mix(x, y, glowC, 0.14*(1-d)*(1-d)*v*v)
		}
	}
	faint := mixRGB(pal.bg, pal.text, 0.25)
	for i := range 18 {
		a := t*(0.15+0.2*hash(i, 1, 430)) + 6.3*hash(i, 2, 430)
		x := cx + (r*2.5+10*hash(i, 3, 430))*math.Cos(a)
		y := wrap(float64(H)*hash(i, 4, 430)-t*(0.6+hash(i, 5, 430)), float64(H))
		c := faint
		if math.Sin(a) < 0 {
			c = mixRGB(pal.bg, faint, 0.5)
		}
		g.char(int(x), int(y)/2, []rune("·∙˙")[i%3], c)
	}
	angle := func(y float64) float64 { return (y-top)/period*2*math.Pi + t*0.9 }
	shade := func(c rgb, z float64) rgb { return mixRGB(pal.bg, c, 0.25+0.75*(z+1)/2) }
	// A strand's pixel, antialiased across: the pixel it falls in and its
	// neighbor, by how far it is between them.
	plot := func(x, y float64, c rgb, a float64) {
		x0 := math.Floor(x)
		g.mix(int(x0), int(y), c, a*(1-(x-x0)))
		g.mix(int(x0)+1, int(y), c, a*(x-x0))
	}
	pulse := wrap(t*10, hh+20) - 10
	for pass := range 2 {
		for y := top; y < top+hh; y++ {
			th := angle(y)
			off, z := r*math.Sin(th), math.Cos(th)
			for _, s := range []struct {
				sign float64
				c    rgb
			}{{1, pal.accent}, {-1, pal.second}} {
				zz := z * s.sign
				if (zz < 0) != (pass == 0) {
					continue
				}
				c := shade(s.c, zz)
				if zz > 0 && math.Abs(y-top-pulse) < 1.5 {
					c = mixRGB(c, rgb{255, 255, 255}, 0.6)
				}
				plot(cx+off*s.sign, y, c, 1)
				if zz > 0.3 {
					plot(cx+off*s.sign+0.8, y, c, 0.5)
				}
			}
		}
	}
	// The rungs, a row of cells apart, lettered at their ends where they
	// face the viewer.
	pairs := [4][2]rune{{'A', 'T'}, {'T', 'A'}, {'C', 'G'}, {'G', 'C'}}
	colors := [4]rgb{pal.ansi[2], pal.ansi[3], pal.ansi[5], pal.ansi[6]}
	for row := int(top)/2 + 1; row < int(top+hh)/2; row++ {
		y := float64(row*2) + 0.5
		th := angle(y)
		off, z := r*math.Sin(th), math.Cos(th)
		if math.Abs(off) < 2 {
			continue
		}
		kind := int(hash(row-int(top)/2, 3, 431) * 4)
		c := mixRGB(pal.bg, colors[kind], 0.35+0.4*(1-math.Abs(z)))
		lo, hi := cx-math.Abs(off), cx+math.Abs(off)
		for col := int(lo) + 2; col < int(hi); col++ {
			g.char(col, row, '─', c)
		}
		if math.Abs(math.Sin(th)) > 0.6 {
			a, b := pairs[kind][0], pairs[kind][1]
			if off < 0 {
				a, b = b, a
			}
			g.char(int(lo), row, a, shade(colors[kind], 0.4))
			g.char(int(hi)+1, row, b, shade(colors[kind], 0.4))
		}
	}
}
