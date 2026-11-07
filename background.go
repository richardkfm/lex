package main

import (
	"fmt"
	"image"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"math"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
	_ "golang.org/x/image/bmp"
	xdraw "golang.org/x/image/draw"
	_ "golang.org/x/image/webp"
)

// The background image: a picture of the user's, by its path in the
// settings, that the window shows behind its title bar and panes, the
// panes translucent over it (settings.Background).

// backdrop is the background image as the window draws it.
type backdrop struct {
	path string
	// bmp is the picture, blurred as the settings ask; w and h are its
	// own pixels, which "center" and "tile" show it at. art is one of
	// Lex's abstract backgrounds instead, which the frame draws (art.go).
	bmp  *ui.Bitmap
	w, h int
	art  *art
	// err tells why the picture could not be shown.
	err string
}

// bgSource is a picture decoded once, so that a change of blur blurs it
// again without reading it anew.
type bgSource struct {
	path string
	img  *image.RGBA
	w, h int
	err  error
}

// maxBackground is the longest side a picture keeps: larger ones are
// shrunk once read, as no screen shows more of them.
const maxBackground = 3840

// imageFilter is what the dialog choosing a background shows: pictures
// of the formats Lex reads, in either case, as GTK's patterns heed it.
var imageFilter = func() mygo.FileFilter {
	f := mygo.FileFilter{Name: "Images"}
	for _, e := range []string{"png", "jpg", "jpeg", "webp", "gif", "bmp"} {
		f.Extensions = append(f.Extensions, e)
		if runtime.GOOS != "windows" && runtime.GOOS != "darwin" {
			f.Extensions = append(f.Extensions, strings.ToUpper(e))
		}
	}
	return f
}()

// background returns the background image to draw, nil for none, and
// starts making it when the settings ask for another picture or blur.
// While a new blur is made, the picture shows with the last.
func (a *App) background() *backdrop {
	path := prefs.Background
	if path == "" {
		return nil
	}
	if p, ok := patternOf(path); ok && p.art != nil {
		// Drawn by the painter as it draws the window, in the theme's
		// colors (drawBackground).
		if a.bg == nil || a.bg.path != path {
			a.bg, a.bgKey = &backdrop{path: path, art: p.art}, ""
		}
		return a.bg
	}
	blur := prefs.backgroundBlur()
	key := fmt.Sprintf("%d|%s", blur, path)
	if a.bgKey != key {
		a.bgKey = key
		gen := a.bgGen.Add(1)
		go a.loadBackground(path, blur, key, gen)
	}
	if a.bg != nil && a.bg.path == path {
		return a.bg
	}
	return nil
}

// loadBackground reads and blurs the picture, off the main thread, and
// gives it to the window unless the settings asked for another since.
func (a *App) loadBackground(path string, blur int, key string, gen int64) {
	src := a.backgroundSource(path)
	b := &backdrop{path: path, w: src.w, h: src.h}
	switch {
	case src.err != nil:
		b.err = src.err.Error()
	case a.bgGen.Load() != gen:
		return
	default:
		b.bmp = backdropBitmap(path, src.img, blur)
	}
	a.laterService(func() {
		if a.bgKey == key {
			a.bg = b
		}
	})
}

// backdropBitmap returns the bitmap of a background's picture, blurred by
// blur DIPs: a scene's pixel art (scenes.go) shows its pixels as squares
// unless it is blurred.
func backdropBitmap(path string, img *image.RGBA, blur int) *ui.Bitmap {
	if p, ok := patternOf(path); ok && p.scene != nil {
		if blur > 0 {
			return ui.NewBitmap(blurredScene(img, blur))
		}
		return ui.NewBitmap(img).Pixelated()
	}
	if blur > 0 {
		img = blurred(img, float64(blur))
	}
	return ui.NewBitmap(img)
}

// backgroundSource returns the picture at path, decoded and shrunk, from
// the last one read when it is the same.
func (a *App) backgroundSource(path string) *bgSource {
	a.bgMu.Lock()
	defer a.bgMu.Unlock()
	if s := a.bgSrc; s != nil && s.path == path {
		return s
	}
	s := &bgSource{path: path}
	if p, ok := patternOf(path); ok {
		s.img = p.render(patternW, patternH, patternW/1920.0)
		s.w, s.h = s.img.Bounds().Dx(), s.img.Bounds().Dy()
	} else {
		s.img, s.w, s.h, s.err = decodePicture(path)
	}
	a.bgSrc = s
	return s
}

// decodePicture reads a picture, shrunk to maxBackground at most, and
// its own size.
func decodePicture(path string) (*image.RGBA, int, int, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, 0, 0, fmt.Errorf("could not open %s", filepath.Base(path))
	}
	defer f.Close()
	img, _, err := image.Decode(f)
	if err != nil {
		return nil, 0, 0, fmt.Errorf("%s is not a picture Lex can read", filepath.Base(path))
	}
	b := img.Bounds()
	w, h := b.Dx(), b.Dy()
	if w == 0 || h == 0 {
		return nil, 0, 0, fmt.Errorf("%s is empty", filepath.Base(path))
	}
	k := min(1, float64(maxBackground)/float64(max(w, h)))
	dst := image.NewRGBA(image.Rect(0, 0, max(1, int(float64(w)*k)), max(1, int(float64(h)*k))))
	xdraw.BiLinear.Scale(dst, dst.Bounds(), img, b, xdraw.Src, nil)
	return dst, w, h, nil
}

// blurred returns the picture blurred by sigma DIPs of a window as wide
// as a full HD screen, a Gaussian made of three box blurs. It shrinks the
// picture first, so that the blur spans a few of its pixels: that costs
// little, and looks the same once the window stretches it again.
func blurred(src *image.RGBA, sigma float64) *image.RGBA {
	b := src.Bounds()
	sigma *= max(1, float64(b.Dx())/1920)
	shrink := max(1, sigma/3)
	w, h := max(1, int(float64(b.Dx())/shrink)), max(1, int(float64(b.Dy())/shrink))
	small := image.NewRGBA(image.Rect(0, 0, w, h))
	xdraw.BiLinear.Scale(small, small.Bounds(), src, b, xdraw.Src, nil)
	tmp := make([]uint8, len(small.Pix))
	for _, size := range boxesForGauss(sigma/shrink, 3) {
		r := (size - 1) / 2
		boxBlur(small.Pix, tmp, h, w, r, w*4, 4) // across
		boxBlur(tmp, small.Pix, w, h, r, 4, w*4) // down
	}
	return small
}

// boxesForGauss returns the widths of n box blurs that, one after
// another, blur about as a Gaussian of sigma does.
func boxesForGauss(sigma float64, n int) []int {
	ideal := math.Sqrt(12*sigma*sigma/float64(n) + 1)
	wl := int(math.Floor(ideal))
	if wl%2 == 0 {
		wl--
	}
	wl = max(wl, 1)
	wu := wl + 2
	m := int(math.Round((12*sigma*sigma - float64(n*wl*wl) - 4*float64(n*wl) - 3*float64(n)) / (-4*float64(wl) - 4)))
	sizes := make([]int, n)
	for i := range sizes {
		if i < m {
			sizes[i] = wl
		} else {
			sizes[i] = wu
		}
	}
	return sizes
}

// boxBlur averages each pixel of src with the r on either side of it
// along lines, into dst, as the edge pixels repeat beyond the picture:
// lines of length pixels, step bytes from one pixel to the next and
// stride from one line to the next, of four bytes a pixel.
func boxBlur(src, dst []uint8, lines, length, r, stride, step int) {
	if r <= 0 {
		copy(dst, src)
		return
	}
	div := 2*r + 1
	for l := range lines {
		base := l * stride
		for ch := range 4 {
			at := func(i int) int { return int(src[base+min(max(i, 0), length-1)*step+ch]) }
			sum := 0
			for i := -r; i <= r; i++ {
				sum += at(i)
			}
			for i := range length {
				dst[base+i*step+ch] = uint8(sum / div)
				sum += at(i+r+1) - at(i-r)
			}
		}
	}
}

// drawBackground draws the background over r: still, or, while the
// settings move it, drifting slowly, a little larger than the window so
// that its edges never show, and playing when it is one of Lex's scenes.
// It moves while the window has the focus, as Ghostty's animated shaders
// do by default, and keeps still while the system asks for less motion
// or the window draws without a GPU, where drawing it all the time would
// cost too much.
func (a *App) drawBackground(p *ui.Painter, r ui.Rect, b *backdrop, fit string, th *theme) {
	moves := a.backgroundMoves(p)
	if moves {
		// The drift's clock runs only while the background moves, so that
		// it goes on from where it stopped, rather than jumping.
		now := p.Now()
		if !a.driftAt.IsZero() {
			a.drift += min(max(now.Sub(a.driftAt), 0), 100*time.Millisecond)
		}
		a.driftAt = now
		p.AnimationFrameRate(30)
	} else {
		a.driftAt = time.Time{}
	}
	shown := b
	if pt, ok := patternOf(b.path); ok && (pt.art != nil || pt.scene != nil && moves) {
		f := *b
		if pt.art != nil {
			t := artStill
			if moves {
				t = artTime(a.drift)
			}
			if a.art == nil {
				a.art = &artView{async: true}
			}
			f.bmp = a.art.bitmap(pt.art, pt.id, int(r.W*p.Scale()), int(r.H*p.Scale()), th, t, prefs.backgroundBlur())
			f.w, f.h = f.bmp.Size()
		} else {
			f.bmp = a.sceneBitmap(pt, a.drift)
		}
		shown = &f
	}
	if !moves {
		drawBackdrop(p, r, shown, fit)
		return
	}
	z, dx, dy := driftAt(a.drift, r)
	p.Transform(ui.Scaling(z, z).Then(ui.Translation(dx, dy)), r.X+r.W/2, r.Y+r.H/2, func() {
		drawBackdrop(p, r, shown, fit)
	})
}

// backgroundMoves tells whether the background moves in this frame.
func (a *App) backgroundMoves(p *ui.Painter) bool {
	return prefs.MovingBackground && a.focusedWin && !p.Preferences().ReduceMotion && (p.Accelerated() || moveOnCPU)
}

// moveOnCPU moves the background without a GPU too, for tests.
var moveOnCPU = false

// The drift: the background, driftZoom larger than the window, pans
// across the room that leaves on either side, along a Lissajous figure
// whose sides take driftX and driftY to sweep, so that it seems to wander
// rather than go back and forth: a few DIPs a second. It grows to that
// size over driftGrow, from the still background.
const (
	driftZoom = 0.07
	driftX    = 97 * time.Second
	driftY    = 131 * time.Second
	driftGrow = 3 * time.Second
)

// driftAt returns how the background over r is scaled, around r's
// center, and then moved, d into the drift.
func driftAt(d time.Duration, r ui.Rect) (zoom, dx, dy float32) {
	grow := smooth(min(d.Seconds()/driftGrow.Seconds(), 1))
	zoom = float32(1 + driftZoom*grow)
	// What the zoom adds on either side, less a little, so that no edge
	// shows at the turns.
	mx, my := float64(r.W)*driftZoom*grow/2*0.9, float64(r.H)*driftZoom*grow/2*0.9
	dx = float32(mx * math.Sin(2*math.Pi*d.Seconds()/driftX.Seconds()))
	dy = float32(my * math.Sin(2*math.Pi*d.Seconds()/driftY.Seconds()))
	return zoom, dx, dy
}

// drawBackdrop draws the picture over r as the settings fit it.
func drawBackdrop(p *ui.Painter, r ui.Rect, b *backdrop, fit string) {
	w, h := float32(b.w), float32(b.h)
	switch fit {
	case "fit":
		p.Image(b.bmp, r, ui.Contain)
	case "center":
		p.Image(b.bmp, ui.Rect{X: r.X + (r.W-w)/2, Y: r.Y + (r.H-h)/2, W: w, H: h}, ui.FillBox)
	case "tile":
		// A tiny picture tiles at 64 DIPs at least, as thousands of tiles
		// would cost more than they show.
		for w < 64 || h < 64 {
			w, h = w*2, h*2
		}
		for y := r.Y; y < r.Y+r.H; y += h {
			for x := r.X; x < r.X+r.W; x += w {
				p.Image(b.bmp, ui.Rect{X: x, Y: y, W: w, H: h}, ui.FillBox)
			}
		}
	default:
		p.Image(b.bmp, r, ui.Cover)
	}
}

// chooseBackground asks for a picture with the system's dialog, and makes
// it the background.
func (a *App) chooseBackground() {
	win := a.win
	dir := ""
	if prefs.Background != "" {
		dir = filepath.Dir(prefs.Background)
	}
	go func() {
		paths, err := mygo.Dialog.Open(mygo.OpenDialogOptions{
			Parent:      win,
			Title:       "Choose a Background Image",
			ButtonLabel: "Choose",
			DefaultPath: dir,
			Filters:     []mygo.FileFilter{imageFilter},
		})
		if err != nil || len(paths) == 0 {
			return
		}
		a.laterService(func() { a.setBackground(paths[0]) })
	}()
}

// setBackground makes the picture at path, or one of Lex's patterns
// ("lex:nebula"), the background, none for "". The first background
// comes slightly dimmed, so that text stays readable over it. A picture
// of the user's is kept, so that the settings offer it again after
// another background.
func (a *App) setBackground(path string) {
	a.changeLook(func(s *settings) {
		if path != "" && s.Background == "" && s.PaneOpacity == 0 && s.BackgroundDim == 0 && s.BackgroundBlur == 0 {
			s.BackgroundDim = 0.2
		}
		s.Background = path
		if _, ok := patternOf(path); !ok && path != "" {
			s.CustomBackground = path
		}
	})
}

// backgroundFit is how the background fills the window: as the settings
// ask for a picture, and covering it for a pattern, which is drawn for
// it.
func (s settings) backgroundFit() string {
	if _, ok := patternOf(s.Background); ok {
		return "fill"
	}
	return s.BackgroundFit
}

// patternThumb returns what draws the little picture of a pattern the
// settings show: drawn once, or for an art, in the theme's colors, drawn
// again as the theme changes.
func (a *App) patternThumb(c *ui.Context, p pattern) func(*ui.Painter, float32, float32) *ui.Bitmap {
	if p.art != nil {
		th := prefs.themeFor(c.Theme().Dark)
		return func(pt *ui.Painter, w, h float32) *ui.Bitmap {
			if a.artThumbs == nil {
				a.artThumbs = map[string]*artView{}
			}
			v := a.artThumbs[p.id]
			if v == nil {
				v = &artView{}
				a.artThumbs[p.id] = v
			}
			// As though in a small window, shrunk.
			return v.bitmap(p.art, p.id, 480, int(480*h/w), th, artStill, 0)
		}
	}
	if a.thumbs == nil {
		a.thumbs = map[string]*ui.Bitmap{}
	}
	b, ok := a.thumbs[p.id]
	if !ok {
		b = ui.NewBitmap(p.render(104, 72, 0.5))
		if p.scene != nil {
			b.Pixelated()
		}
		a.thumbs[p.id] = b
	}
	return bitmapThumb(b)
}

// bitmapThumb returns what draws a thumbnail that is a bitmap, nil for
// none yet.
func bitmapThumb(b *ui.Bitmap) func(*ui.Painter, float32, float32) *ui.Bitmap {
	if b == nil {
		return nil
	}
	return func(*ui.Painter, float32, float32) *ui.Bitmap { return b }
}

// customThumb returns the little picture of the user's own background,
// which it starts making off the main thread the first time; nil until
// it is made, or when the picture cannot be read.
func (a *App) customThumb(path string) *ui.Bitmap {
	if a.thumbs == nil {
		a.thumbs = map[string]*ui.Bitmap{}
	}
	key := "file:" + path
	if b, ok := a.thumbs[key]; ok {
		return b
	}
	a.thumbs[key] = nil
	go func() {
		img, _, _, err := decodePicture(path)
		if err != nil {
			return
		}
		b := img.Bounds()
		small := image.NewRGBA(image.Rect(0, 0, 128, max(1, 128*b.Dy()/b.Dx())))
		xdraw.BiLinear.Scale(small, small.Bounds(), img, b, xdraw.Src, nil)
		bmp := ui.NewBitmap(small)
		a.laterService(func() { a.thumbs[key] = bmp })
	}()
	return nil
}

// backgroundBlur is the blur the settings ask for, in whole DIPs.
func (s settings) backgroundBlur() int {
	return int(math.Round(float64(min(max(s.BackgroundBlur, 0), 24))))
}

// backgroundDim is how much the settings dim the picture.
func (s settings) backgroundDim() float32 {
	return min(max(s.BackgroundDim, 0), 0.8)
}
