// Mkicon draws Lex's icon, resources/icon.png, in patina's aurora colors
// and the flat manner of Linux icon themes such as Papirus: a black
// rounded square on a short shadow, with the app's panes on it as hairline
// cards, the focused one in the accent's edge with a prompt, a pulse in
// the second color on another, and lines of text on the third.
//
//	go run ./tools/mkicon
package main

import (
	"image"
	"image/color"
	"image/png"
	"log"
	"math"
	"os"

	"golang.org/x/image/vector"
)

const size = 1024

// aurora's colors.
var (
	black    = color.NRGBA{0x00, 0x00, 0x00, 255} // base00
	panel    = color.NRGBA{0x0b, 0x0a, 0x10, 255} // base01
	lavender = color.NRGBA{0xd8, 0xd4, 0xe6, 255} // base05
	green    = color.NRGBA{0x3c, 0xff, 0x9d, 255} // base0B, the accent
	violet   = color.NRGBA{0xb7, 0x7d, 0xff, 255} // base0E, the second color
)

func main() {
	img := image.NewRGBA(image.Rect(0, 0, size, size))
	// The square: a little inside the canvas, on a shadow a step below it,
	// as Papirus sets its icons, in a light rim that keeps it apart from
	// dark panels.
	x0, y0, w, rad := float32(80), float32(72), float32(864), float32(112)
	fill(img, rounded(x0, y0+16, w, w, rad), uniform(color.NRGBA{0, 0, 0, 50}))
	fill(img, rounded(x0, y0, w, w, rad), uniform(alpha(lavender, 46)))
	fill(img, rounded(x0+6, y0+6, w-12, w-12, rad-6), uniform(black))
	// The panes: one left above another, one right, as the app shows them.
	pad, gap := float32(64), float32(40)
	inner := w - 2*pad
	lw := inner*0.56 - gap/2
	th := inner*0.56 - gap/2
	px, py := x0+pad, y0+pad
	focused := card{px, py, lw, th}
	lower := card{px, py + th + gap, lw, inner - th - gap}
	right := card{px + lw + gap, py, inner - lw - gap, inner}
	focused.draw(img, green, 10)
	lower.draw(img, alpha(lavender, 40), 6)
	right.draw(img, alpha(lavender, 40), 6)
	// A prompt on the focused pane: a chevron and a cursor.
	cx, cy := focused.x+86, focused.y+focused.h/2
	for _, r := range stroke([][2]float32{{cx, cy - 64}, {cx + 68, cy}, {cx, cy + 64}}, 34) {
		fill(img, r, uniform(green))
	}
	fill(img, rounded(cx+112, cy+44, 112, 34, 8), uniform(lavender))
	// A pulse on the right pane, as a monitor draws it.
	r := right
	my := r.y + r.h*0.5
	pulse := [][2]float32{
		{r.x + 44, my}, {r.x + r.w*0.32, my}, {r.x + r.w*0.44, my - 120},
		{r.x + r.w*0.6, my + 120}, {r.x + r.w*0.7, my}, {r.x + r.w - 44, my},
	}
	for _, s := range stroke(pulse, 26) {
		fill(img, s, uniform(violet))
	}
	// Lines of text on the lower pane.
	for i, f := range []float32{0.7, 0.46, 0.6} {
		col := alpha(lavender, 110)
		if i == 1 {
			col = violet
		}
		fill(img, rounded(lower.x+52, lower.y+52+float32(i)*44, (lower.w-104)*f, 22, 11), uniform(col))
	}
	if err := os.MkdirAll("resources", 0o755); err != nil {
		log.Fatal(err)
	}
	f, err := os.Create("resources/icon.png")
	if err != nil {
		log.Fatal(err)
	}
	defer f.Close()
	if err := png.Encode(f, img); err != nil {
		log.Fatal(err)
	}
}

// card is a pane of the icon: a flat card in a rim.
type card struct{ x, y, w, h float32 }

func (c card) draw(img *image.RGBA, rim color.NRGBA, width float32) {
	const rad = 40
	fill(img, rounded(c.x, c.y, c.w, c.h, rad), uniform(rim))
	fill(img, rounded(c.x+width, c.y+width, c.w-2*width, c.h-2*width, rad-width), uniform(panel))
}

func uniform(c color.NRGBA) image.Image { return image.NewUniform(c) }

func alpha(c color.NRGBA, a uint8) color.NRGBA {
	c.A = a
	return c
}

func fill(dst *image.RGBA, r *vector.Rasterizer, src image.Image) {
	r.Draw(dst, dst.Bounds(), src, image.Point{})
}

// rounded is a rounded rectangle.
func rounded(x, y, w, h, rad float32) *vector.Rasterizer {
	r := vector.NewRasterizer(size, size)
	k := rad * 0.5523
	r.MoveTo(x+rad, y)
	r.LineTo(x+w-rad, y)
	r.CubeTo(x+w-rad+k, y, x+w, y+rad-k, x+w, y+rad)
	r.LineTo(x+w, y+h-rad)
	r.CubeTo(x+w, y+h-rad+k, x+w-rad+k, y+h, x+w-rad, y+h)
	r.LineTo(x+rad, y+h)
	r.CubeTo(x+rad-k, y+h, x, y+h-rad+k, x, y+h-rad)
	r.LineTo(x, y+rad)
	r.CubeTo(x, y+rad-k, x+rad-k, y, x+rad, y)
	r.ClosePath()
	return r
}

// stroke is a polyline of a width, with round joins and caps: shapes to
// fill one by one, as their windings would cancel where they overlap.
func stroke(pts [][2]float32, width float32) []*vector.Rasterizer {
	var out []*vector.Rasterizer
	hw := width / 2
	for i := 0; i+1 < len(pts); i++ {
		r := vector.NewRasterizer(size, size)
		out = append(out, r)
		a, b := pts[i], pts[i+1]
		dx, dy := b[0]-a[0], b[1]-a[1]
		l := float32(math.Hypot(float64(dx), float64(dy)))
		nx, ny := -dy/l*hw, dx/l*hw
		r.MoveTo(a[0]+nx, a[1]+ny)
		r.LineTo(b[0]+nx, b[1]+ny)
		r.LineTo(b[0]-nx, b[1]-ny)
		r.LineTo(a[0]-nx, a[1]-ny)
		r.ClosePath()
	}
	for _, p := range pts {
		r := vector.NewRasterizer(size, size)
		circle(r, p[0], p[1], hw)
		out = append(out, r)
	}
	return out
}

func circle(r *vector.Rasterizer, cx, cy, rad float32) {
	k := rad * 0.5523
	r.MoveTo(cx+rad, cy)
	r.CubeTo(cx+rad, cy+k, cx+k, cy+rad, cx, cy+rad)
	r.CubeTo(cx-k, cy+rad, cx-rad, cy+k, cx-rad, cy)
	r.CubeTo(cx-rad, cy-k, cx-k, cy-rad, cx, cy-rad)
	r.CubeTo(cx+k, cy-rad, cx+rad, cy-k, cx+rad, cy)
	r.ClosePath()
}
