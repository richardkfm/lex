package main

import (
	"image"
	"math"
	"time"

	"github.com/egoist/mygo/ui"
)

// Animation panes: a pane that plays an animation in place of a terminal,
// which Lex draws itself on a grid of the terminals' cells, as a program
// of terminal art would: each cell two pixels high, as the half blocks
// ▀ ▄ make them, and characters over them for the details, as rain, stars
// and letters. An animation is drawn from the time and the grid's size
// alone, so that any frame draws at once and the pane may take any size.
// The palette opens one beside the focused pane ("Animation: Helix"), the
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
var animations = []*animation{helixAnim, neonCityAnim, solarpunkAnim, saharaAnim}

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
// color, and its terminals' sixteen.
type animPalette struct {
	bg, card, text, accent, second rgb
	ansi                           [16]rgb
	dark                           bool
}

func paletteOf(t *theme) animPalette {
	p := animPalette{bg: hexRGB(t.bg), card: hexRGB(t.card), text: hexRGB(t.text), accent: hexRGB(t.accent),
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

// grid is a frame of an animation being painted: cols×rows cells, w×h
// pixels, two to a cell, one above the other, and characters over cells.
type grid struct {
	cols, rows int
	w, h       int
	img        *image.RGBA
	glyphs     []glyph
}

// glyph is a character over a cell, in a color.
type glyph struct {
	col, row int
	ch       rune
	c        rgb
}

func newGrid(cols, rows int) *grid {
	return &grid{cols: cols, rows: rows, w: cols, h: rows * 2, img: image.NewRGBA(image.Rect(0, 0, cols, rows*2))}
}

// reset clears the grid's characters, for the next frame.
func (g *grid) reset() { g.glyphs = g.glyphs[:0] }

func (g *grid) in(x, y int) bool { return x >= 0 && y >= 0 && x < g.w && y < g.h }

// set paints the pixel at (x, y).
func (g *grid) set(x, y int, c rgb) {
	if !g.in(x, y) {
		return
	}
	o := g.img.PixOffset(x, y)
	g.img.Pix[o], g.img.Pix[o+1], g.img.Pix[o+2], g.img.Pix[o+3] = c[0], c[1], c[2], 255
}

// at returns the color of the pixel at (x, y).
func (g *grid) at(x, y int) rgb {
	x, y = min(max(x, 0), g.w-1), min(max(y, 0), g.h-1)
	o := g.img.PixOffset(x, y)
	return rgb{g.img.Pix[o], g.img.Pix[o+1], g.img.Pix[o+2]}
}

// mix lays c over the pixel at (x, y), as opaque as a.
func (g *grid) mix(x, y int, c rgb, a float64) {
	if g.in(x, y) && a > 0 {
		g.set(x, y, mixRGB(g.at(x, y), c, a))
	}
}

// dither paints c at (x, y) where a passes the pixel's threshold in a 4×4
// Bayer matrix: a share a of an area's pixels take it, evenly spread, as
// pixel art shades.
func (g *grid) dither(x, y int, c rgb, a float64) {
	if a > bayer4(x, y) {
		g.set(x, y, c)
	}
}

// rect paints the pixels from (x0, y0) to (x1, y1), excluded.
func (g *grid) rect(x0, y0, x1, y1 int, c rgb) {
	for y := max(y0, 0); y < min(y1, g.h); y++ {
		for x := max(x0, 0); x < min(x1, g.w); x++ {
			g.set(x, y, c)
		}
	}
}

// gradient paints rows y0 to y1, excluded, in the colors stops, top to
// bottom, a row a shade: at pixels as large as a cell's half, a dither
// would show as a checkerboard.
func (g *grid) gradient(y0, y1 int, stops ...rgb) {
	n := len(stops) - 1
	for y := max(y0, 0); y < min(y1, g.h); y++ {
		t := float64(y-y0) / float64(max(y1-y0-1, 1)) * float64(n)
		i := min(int(t), n-1)
		c := mixRGB(stops[i], stops[i+1], t-float64(i))
		for x := range g.w {
			g.set(x, y, c)
		}
	}
}

// disc paints a disc of radius r round (cx, cy).
func (g *grid) disc(cx, cy, r float64, c rgb) {
	for y := int(cy - r - 1); y <= int(cy+r+1); y++ {
		for x := int(cx - r - 1); x <= int(cx+r+1); x++ {
			if math.Hypot(float64(x)+0.5-cx, float64(y)+0.5-cy) <= r {
				g.set(x, y, c)
			}
		}
	}
}

// glow lights the pixels round (cx, cy), out to r, toward c, at most a.
func (g *grid) glow(cx, cy, r float64, c rgb, a float64) {
	for y := int(cy - r); y <= int(cy+r); y++ {
		for x := int(cx - r); x <= int(cx+r); x++ {
			if d := math.Hypot(float64(x)+0.5-cx, float64(y)+0.5-cy) / r; d < 1 {
				g.mix(x, y, c, a*(1-d)*(1-d))
			}
		}
	}
}

// line paints a line of pixels from (x0, y0) to (x1, y1).
func (g *grid) line(x0, y0, x1, y1 float64, c rgb) {
	n := int(math.Max(math.Abs(x1-x0), math.Abs(y1-y0))) + 1
	for i := 0; i <= n; i++ {
		k := float64(i) / float64(n)
		g.set(int(math.Floor(x0+(x1-x0)*k)), int(math.Floor(y0+(y1-y0)*k)), c)
	}
}

// char puts the character ch over cell (col, row).
func (g *grid) char(col, row int, ch rune, c rgb) {
	if col >= 0 && row >= 0 && col < g.cols && row < g.rows {
		g.glyphs = append(g.glyphs, glyph{col, row, ch, c})
	}
}

// text puts s over the cells from (col, row) on, a character a cell.
func (g *grid) text(col, row int, s string, c rgb) {
	for _, ch := range s {
		g.char(col, row, ch, c)
		col++
	}
}

// bayer4 is the threshold of (x, y) in a 4×4 Bayer matrix, from 0 to 1.
func bayer4(x, y int) float64 {
	m := [16]float64{0, 8, 2, 10, 12, 4, 14, 6, 3, 11, 1, 9, 15, 7, 13, 5}
	return (m[(y&3)*4+x&3] + 0.5) / 16
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

// wrap returns v within [0, n).
func wrap(v, n float64) float64 {
	v = math.Mod(v, n)
	if v < 0 {
		v += n
	}
	return v
}

// animPlayer plays an animation in a pane: the grid it paints, and the
// bitmap of its pixels, in two by turns, as a bitmap takes its picture
// without a copy.
type animPlayer struct {
	id         string
	theme      string
	cols, rows int
	step       int64
	grids      [2]*grid
	cur        int
	bmp        *ui.Bitmap
}

// frame returns the grid of animation an at t, cols×rows cells, in
// theme th: painted again only when its step, its size or the theme
// changed.
func (pl *animPlayer) frame(an *animation, cols, rows int, t float64, th *theme) *grid {
	step := int64(math.Floor(t * an.fps))
	if pl.bmp != nil && pl.id == an.id && pl.theme == th.id && pl.cols == cols && pl.rows == rows && pl.step == step {
		return pl.grids[pl.cur]
	}
	if pl.cols != cols || pl.rows != rows || pl.grids[0] == nil {
		pl.grids = [2]*grid{newGrid(cols, rows), newGrid(cols, rows)}
		pl.bmp = nil
	}
	pl.id, pl.theme, pl.cols, pl.rows, pl.step = an.id, th.id, cols, rows, step
	pl.cur ^= 1
	g := pl.grids[pl.cur]
	g.reset()
	an.draw(g, float64(step)/an.fps, paletteOf(th))
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

// animHeight is about how many pixels high the scenes are drawn for, and
// animWidth how many wide at least: a pane larger than twice that shows
// them twice as large, its cells and characters too, and so on, rather
// than as small scenes in a wide sky.
const animHeight, animWidth = 46, 72

// drawAnimation draws the animation of pane pn over r: its pixels, as
// squares, filling r, and its characters over them, in the terminals'
// font, centered in their cells.
func (a *App) drawAnimation(p *ui.Painter, r ui.Rect, pn *Pane, th *theme) {
	an := animationOf(pn.anim)
	cw, ch := cellSize(p)
	cols, rows := max(int(r.W/cw), 8), max(int(r.H/ch), 4)
	scale := max(1, min(rows*2/animHeight, cols/animWidth))
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
	g := pn.player.frame(an, cols, rows, t, th)
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
