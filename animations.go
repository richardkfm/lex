package main

import (
	"image"
	"math"
	"time"

	"github.com/egoist/mygo/ui"
)

// Animation panes: a pane that plays an animation in place of a terminal,
// which Lex draws itself on a grid of the terminals' cells, as a program
// of terminal art would: each cell eight pixels, two across and four
// down, as braille's dots, and characters over them for the details, as
// stars, spores and birds. An animation is drawn from the time and the
// grid's size alone, so that any frame draws at once and the pane may
// take any size.
// The palette opens one beside the focused pane ("Animation: Aurora"), the
// pane's header switches between them, and the layout keeps them. They
// play while they show, and hold still while the system asks for less
// motion.

// An animation is one a pane plays: draw paints its frame at t seconds
// on g, in the colors of the window's theme for those that take them.
type animation struct {
	id, name string
	// fps is how many frames a second it steps at; tile colors its tile
	// in the tabs.
	fps  float64
	tile ui.Color
	draw func(g *grid, t float64, pal animPalette)
}

// animations are the animations, in the order the palette and the
// header's arrows go through them.
var animations = []*animation{duneSeaAnim, neonGridAnim, gardenCityAnim, helixAnim, auroraAnim, oscilloscopeAnim, tracerouteAnim}

// animationOf returns the animation of id, the first for one Lex does not
// know, as of a layout of a later version.
func animationOf(id string) *animation {
	for _, a := range animations {
		if a.id == id {
			return a
		}
	}
	return animations[0]
}

// animStill is the time of the frame shown while the system asks for less
// motion.
const animStill = 12.0

// animEpoch is when the animations' clock starts: every pane playing one
// plays it in step.
var animEpoch = time.Now()

// animPalette are the colors of the window's theme an animation may paint
// with: its window's and panes' faces, its text, its accent and second
// color, and its terminals' sixteen; and whether the places take them
// (animcolors.go).
type animPalette struct {
	id                             string
	bg, card, text, accent, second rgb
	ansi                           [16]rgb
	dark, themed                   bool
}

func paletteOf(t *theme) animPalette {
	p := animPalette{id: t.id, bg: hexRGB(t.bg), card: hexRGB(t.card), text: hexRGB(t.text), accent: hexRGB(t.accent),
		second: hexRGB(t.second), dark: t.dark}
	for i, c := range t.pal {
		p.ansi[i] = hexRGB(c)
	}
	return p
}

// rgb is a color an animation paints with.
type rgb [3]uint8

// hexRGB returns the color "#rrggbb".
func hexRGB(s string) rgb {
	c := rgbOf(s)
	return rgb{uint8(c[0]*255 + 0.5), uint8(c[1]*255 + 0.5), uint8(c[2]*255 + 0.5)}
}

// mixRGB returns the color k of the way from a to b.
func mixRGB(a, b rgb, k float64) rgb {
	k = min(max(k, 0), 1)
	return rgb{
		uint8(float64(a[0]) + (float64(b[0])-float64(a[0]))*k + 0.5),
		uint8(float64(a[1]) + (float64(b[1])-float64(a[1]))*k + 0.5),
		uint8(float64(a[2]) + (float64(b[2])-float64(a[2]))*k + 0.5),
	}
}

// designCols×designRows cells are what the scenes are composed on, and
// drawn in the pixels of: a pane of another size centers them across and
// shows more round them, of the sky above mostly, or less of their edges.
const designCols, designRows = 64, 16

// grid is a frame of an animation being painted: cols×rows cells, w×h
// pixels, two to a cell across and four down, about square in a cell of
// the terminals' fonts, as braille's dots, and characters over cells.
// Scenes paint it in the design's pixels, whose origin is ox, oy pixels
// into it: it spans the design's pixels from x0 to x1 across and y0 to y1
// down. Its colors add up in px, as light does, and come to img once
// painted (finish).
type grid struct {
	cols, rows     int
	w, h           int
	ox, oy         int
	x0, x1, y0, y1 float64
	px             []float32
	img            *image.RGBA
	glyphs         []glyph
}

// glyph is a character over a cell, in a color.
type glyph struct {
	col, row int
	ch       rune
	c        rgb
}

func newGrid(cols, rows int) *grid {
	g := &grid{cols: cols, rows: rows, w: cols * 2, h: rows * 4}
	dc, dr := cols-designCols, rows-designRows
	ox, oy := dc/2, dr/2
	if dr > 0 {
		oy = dr - dr/3 // two rows above for one below
	}
	g.ox, g.oy = ox*2, oy*4
	g.x0, g.x1 = float64(-g.ox), float64(g.w-g.ox)
	g.y0, g.y1 = float64(-g.oy), float64(g.h-g.oy)
	g.px = make([]float32, g.w*g.h*3)
	g.img = image.NewRGBA(image.Rect(0, 0, g.w, g.h))
	return g
}

// wd and hd are how many of the design's pixels the grid spans, across
// and down, and area how many times the design's it is.
func (g *grid) wd() float64   { return g.x1 - g.x0 }
func (g *grid) hd() float64   { return g.y1 - g.y0 }
func (g *grid) area() float64 { return g.wd() * g.hd() / (designCols * 2 * designRows * 4) }

// paint paints the frame of an at t: its pixels unpainted until it paints
// them, which finish leaves clear, and no characters.
func (g *grid) paint(an *animation, t float64, pal animPalette) {
	g.glyphs = g.glyphs[:0]
	nan := float32(math.NaN())
	for i := range g.px {
		g.px[i] = nan
	}
	an.draw(g, t, pal)
	g.finish()
}

// finish brings the pixels to img.
func (g *grid) finish() {
	for i, j := 0, 0; i < len(g.px); i, j = i+3, j+4 {
		if math.IsNaN(float64(g.px[i])) {
			g.img.Pix[j], g.img.Pix[j+1], g.img.Pix[j+2], g.img.Pix[j+3] = 0, 0, 0, 0
			continue
		}
		g.img.Pix[j], g.img.Pix[j+1], g.img.Pix[j+2], g.img.Pix[j+3] = uint8(g.px[i]+0.5), uint8(g.px[i+1]+0.5), uint8(g.px[i+2]+0.5), 255
	}
}

// pix returns where the pixel of the design at (x, y) is in px, if in the
// grid.
func (g *grid) pix(x, y float64) (int, bool) {
	ix, iy := int(math.Floor(x))+g.ox, int(math.Floor(y))+g.oy
	if ix < 0 || iy < 0 || ix >= g.w || iy >= g.h {
		return 0, false
	}
	return (iy*g.w + ix) * 3, true
}

// set paints the pixel at (x, y).
func (g *grid) set(x, y float64, c rgb) {
	if i, ok := g.pix(x, y); ok {
		g.px[i], g.px[i+1], g.px[i+2] = float32(c[0]), float32(c[1]), float32(c[2])
	}
}

// get returns the color of the pixel at (x, y), of the nearest in the
// grid for one out of it.
func (g *grid) get(x, y float64) rgb {
	x = min(max(x, g.x0), g.x1-1)
	y = min(max(y, g.y0), g.y1-1)
	i, ok := g.pix(x, y)
	if !ok || math.IsNaN(float64(g.px[i])) {
		return rgb{}
	}
	return rgb{uint8(g.px[i] + 0.5), uint8(g.px[i+1] + 0.5), uint8(g.px[i+2] + 0.5)}
}

// mix lays c over the pixel at (x, y), as opaque as a.
func (g *grid) mix(x, y float64, c rgb, a float64) {
	if a <= 0 {
		return
	}
	if i, ok := g.pix(x, y); ok {
		k := float32(min(a, 1))
		g.px[i] += (float32(c[0]) - g.px[i]) * k
		g.px[i+1] += (float32(c[1]) - g.px[i+1]) * k
		g.px[i+2] += (float32(c[2]) - g.px[i+2]) * k
	}
}

// light adds c, a of it, to the pixel at (x, y), as light falling on it
// does: lights adding up to white at most.
func (g *grid) light(x, y float64, c rgb, a float64) {
	if a <= 0 {
		return
	}
	if i, ok := g.pix(x, y); ok && !math.IsNaN(float64(g.px[i])) {
		k := float32(a)
		g.px[i] = min(g.px[i]+float32(c[0])*k, 255)
		g.px[i+1] = min(g.px[i+1]+float32(c[1])*k, 255)
		g.px[i+2] = min(g.px[i+2]+float32(c[2])*k, 255)
	}
}

// sky paints the rows above y1 in the colors stops, from the design's top
// to y1, the first above its top, as a pane taller than the design shows.
func (g *grid) sky(y1 float64, stops ...rgb) {
	n := float64(len(stops) - 1)
	for y := g.y0; y < min(y1, g.y1); y++ {
		k := min(max(y/max(y1-1, 1)*n, 0), n)
		i := min(int(k), len(stops)-2)
		c := mixRGB(stops[i], stops[i+1], k-float64(i))
		for x := g.x0; x < g.x1; x++ {
			g.set(x, y, c)
		}
	}
}

// fill paints every pixel c.
func (g *grid) fill(c rgb) {
	for i := 0; i < len(g.px); i += 3 {
		g.px[i], g.px[i+1], g.px[i+2] = float32(c[0]), float32(c[1]), float32(c[2])
	}
}

// disc lays a disc of radius r round (cx, cy), as opaque as a, its edge
// smoothed.
func (g *grid) disc(cx, cy, r float64, c rgb, a float64) {
	for y := math.Floor(cy - r - 1); y <= cy+r+1; y++ {
		for x := math.Floor(cx - r - 1); x <= cx+r+1; x++ {
			if d := math.Hypot(x+0.5-cx, y+0.5-cy); d <= r+0.5 {
				g.mix(x, y, c, a*min(max(r+0.5-d, 0), 1))
			}
		}
	}
}

// glow lights the pixels round (cx, cy), out to r, toward c, at most a.
func (g *grid) glow(cx, cy, r float64, c rgb, a float64) {
	for y := math.Floor(cy - r); y <= cy+r; y++ {
		for x := math.Floor(cx - r); x <= cx+r; x++ {
			if d := math.Hypot(x+0.5-cx, y+0.5-cy) / r; d < 1 {
				g.mix(x, y, c, a*(1-d)*(1-d))
			}
		}
	}
}

// ring lays a circle of radius r round (cx, cy).
func (g *grid) ring(cx, cy, r float64, c rgb, a float64) {
	for k := 0.0; k < r*6.3*2; k++ {
		an := k / (r * 2)
		g.splat(cx+r*math.Cos(an), cy+r*math.Sin(an), c, a)
	}
}

// splat lays a point at (x, y), between pixels, over the four round it.
func (g *grid) splat(x, y float64, c rgb, a float64) {
	fx, fy := x-0.5, y-0.5
	ix, iy := math.Floor(fx), math.Floor(fy)
	dx, dy := fx-ix, fy-iy
	g.mix(ix, iy, c, a*(1-dx)*(1-dy))
	g.mix(ix+1, iy, c, a*dx*(1-dy))
	g.mix(ix, iy+1, c, a*(1-dx)*dy)
	g.mix(ix+1, iy+1, c, a*dx*dy)
}

// aline lays a smooth line from (x0, y0) to (x1, y1).
func (g *grid) aline(x0, y0, x1, y1 float64, c rgb, a float64) {
	n := math.Ceil(math.Hypot(x1-x0, y1-y0)*1.5) + 1
	for i := 0.0; i <= n; i++ {
		k := i / n
		g.splat(x0+(x1-x0)*k, y0+(y1-y0)*k, c, a)
	}
}

// char puts the character ch over the cell of the design at (col, row).
func (g *grid) char(col, row float64, ch rune, c rgb) {
	cl, rw := int(math.Floor(col))+g.ox/2, int(math.Floor(row))+g.oy/4
	if cl >= 0 && rw >= 0 && cl < g.cols && rw < g.rows {
		g.glyphs = append(g.glyphs, glyph{cl, rw, ch, c})
	}
}

// text puts s over the cells of the design from (col, row) rightward, its
// spaces left clear, and returns the column after it.
func (g *grid) text(col, row float64, s string, c rgb) float64 {
	for _, ch := range s {
		if ch != ' ' {
			g.char(col, row, ch, c)
		}
		col++
	}
	return col
}

// cellOf returns the cell of the design over the pixel at (x, y).
func cellOf(x, y float64) (float64, float64) { return math.Floor(x / 2), math.Floor(y / 4) }

// stars puts stars over the cells above sky(x), x a pixel: dots, a few
// crosses and sparkles, each twinkling at its pace, and a few fainter
// between them.
func (g *grid) stars(t, density float64, seed int, sky func(x float64) float64, dim, bright rgb) {
	for row := range g.rows {
		for col := range g.cols {
			c, r := col-g.ox/2, row-g.oy/4
			h := hash(c, r, seed)
			if h > density || float64(r*4+3) >= sky(float64(c*2)) {
				continue
			}
			b := 0.55 + 0.45*math.Sin(t*(0.8+2*hash(c, r, seed+1))+6.3*hash(c, r, seed+2))
			switch {
			case h < density*0.08:
				g.char(float64(c), float64(r), '✦', mixRGB(dim, bright, b))
			case h < density*0.22 && b > 0.5:
				g.char(float64(c), float64(r), '+', mixRGB(dim, bright, b*0.8))
			default:
				g.char(float64(c), float64(r), '·', mixRGB(dim, bright, b*0.7))
			}
		}
	}
	for k := range int(40 * g.area()) {
		x := math.Floor(g.x0 + g.wd()*hash(k, 1, seed+9))
		y := math.Floor(g.y0 + (32-g.y0)*hash(k, 2, seed+9))
		if y < sky(x)-2 {
			g.mix(x, y, bright, 0.25+0.3*math.Sin(t*1.3+float64(k)))
		}
	}
}

// spans lays spans out along a row, from start to the right, i = 0, 1, …,
// and to its left, i = -1, -2, …, as far as the grid goes: each of the
// width and the gap after it size gives, laid by f. A pane wider than the
// design shows more of them, where they were.
func (g *grid) spans(start float64, size func(i int) (w, gap float64), f func(i int, x, w float64)) {
	for i, x := 0, start; x < g.x1; i++ {
		w, gap := size(i)
		f(i, x, w)
		x += w + gap
	}
	for i, x := -1, start; x > g.x0; i-- {
		w, gap := size(i)
		x -= w + gap
		f(i, x, w)
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

// fbm is noise of three octaves.
func fbm(x, y float64, seed int) float64 {
	return noise(x, y, seed)*0.6 + noise(x*2.1, y*2.1, seed+1)*0.28 + noise(x*4.3, y*4.3, seed+2)*0.12
}

// wrap returns v within [0, n).
func wrap(v, n float64) float64 {
	v = math.Mod(v, n)
	if v < 0 {
		v += n
	}
	return v
}

// clamp returns v within [lo, hi].
func clamp(v, lo, hi float64) float64 { return min(max(v, lo), hi) }

// animPlayer plays an animation in a pane: the grid it paints, and the
// bitmap of its pixels, in two by turns, as a bitmap takes its picture
// without a copy.
type animPlayer struct {
	id         string
	theme      string
	themed     bool
	cols, rows int
	step       int64
	grids      [2]*grid
	cur        int
	bmp        *ui.Bitmap
}

// frame returns the grid of animation an at t, cols×rows cells, in
// theme th, its places in the theme's colors if themed: painted again
// only when its step, its size or its colors changed.
func (pl *animPlayer) frame(an *animation, cols, rows int, t float64, th *theme, themed bool) *grid {
	step := int64(math.Floor(t * an.fps))
	if pl.bmp != nil && pl.id == an.id && pl.theme == th.id && pl.themed == themed && pl.cols == cols && pl.rows == rows && pl.step == step {
		return pl.grids[pl.cur]
	}
	if pl.cols != cols || pl.rows != rows || pl.grids[0] == nil {
		pl.grids = [2]*grid{newGrid(cols, rows), newGrid(cols, rows)}
		pl.bmp = nil
	}
	pl.id, pl.theme, pl.themed, pl.cols, pl.rows, pl.step = an.id, th.id, themed, cols, rows, step
	pl.cur ^= 1
	g := pl.grids[pl.cur]
	pal := paletteOf(th)
	pal.themed = themed
	g.paint(an, float64(step)/an.fps, pal)
	if pl.bmp == nil {
		pl.bmp = ui.NewBitmap(g.img).Pixelated()
	} else {
		pl.bmp.Update(g.img)
	}
	return g
}

// cellSize returns the size of the terminals' cells, in DIPs: a full block
// of their font, at their size.
func cellSize(p *ui.Painter) (float32, float32) {
	w, h := p.MeasureText(0, ui.Span{Text: "█", Font: termFont.Family, Size: termFont.Size})
	return max(w, 4), max(h*termFont.LineHeight, 8)
}

// drawAnimation draws the animation of pane pn over r: its pixels, as
// squares, filling r, and its characters over them, in the terminals'
// font, centered in their cells. A pane twice the design's size or more
// shows it twice as large, its cells and characters too, and so on.
func (a *App) drawAnimation(p *ui.Painter, r ui.Rect, pn *Pane, th *theme) {
	an := animationOf(pn.anim)
	cw, ch := cellSize(p)
	cols, rows := max(int(r.W/cw), 8), max(int(r.H/ch), 4)
	scale := max(1, min(cols/designCols, rows/designRows))
	cols, rows = max(cols/scale, 8), max(rows/scale, 4)
	size := termFont.Size * float32(scale)
	cw, ch = cw*float32(scale), ch*float32(scale)
	t := animStill
	if !p.Preferences().ReduceMotion {
		t = p.Now().Sub(animEpoch).Seconds()
		p.AnimationFrameRate(float32(an.fps))
	}
	if pn.player == nil {
		pn.player = &animPlayer{}
	}
	g := pn.player.frame(an, cols, rows, t, th, !prefs.NaturalScenes)
	p.Image(pn.player.bmp, r, ui.FillBox)
	sw, sh := r.W/float32(cols), r.H/float32(rows)
	for _, gl := range g.glyphs {
		x := r.X + float32(gl.col)*sw + (sw-cw)/2
		y := r.Y + float32(gl.row)*sh + (sh-ch)/2
		p.RichText(x, y, 0, ui.Span{Text: string(gl.ch), Font: termFont.Family, Size: size,
			Color: ui.RGBA(gl.c[0], gl.c[1], gl.c[2], 1)})
	}
}

// animProgram is how an animation pane shows in its header and its tab.
func animProgram(an *animation) program {
	return program{Name: an.name, Glyph: "activity-pulse", TileBg: an.tile, TileFg: ui.Hex("#ffffff")}
}

// newAnimPane makes a pane that plays the animation of id.
func (a *App) newAnimPane(id string) *Pane {
	return &Pane{ID: a.id(), anim: animationOf(id).id}
}

// splitAnimation opens a pane playing the animation of id right of the
// focused pane.
func (a *App) splitAnimation(id string) {
	a.splitWith(false, func(int, int) *Pane { return a.newAnimPane(id) })
}

// cycleAnimation has an animation pane play the animation dir after its
// own, round.
func (a *App) cycleAnimation(p *Pane, dir int) {
	at := 0
	for i, an := range animations {
		if an.id == p.anim {
			at = i
		}
	}
	p.anim = animations[(at+dir+len(animations))%len(animations)].id
	a.changed()
}

// animationCommands open a pane of each animation, from the palette and
// the Shell menu.
var animationCommands = func() []*command {
	var cmds []*command
	for _, an := range animations {
		cmds = append(cmds, &command{Title: "Animation: " + an.name, Run: func(a *App) { a.splitAnimation(an.id) }})
	}
	return cmds
}()
