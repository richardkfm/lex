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
	// own pixels, which "center" and "tile" show it at.
	bmp  *ui.Bitmap
	w, h int
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
		img := src.img
		if blur > 0 {
			img = blurred(img, float64(blur))
		}
		b.bmp = ui.NewBitmap(img)
	}
	a.laterService(func() {
		if a.bgKey == key {
			a.bg = b
		}
	})
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
	s.img, s.w, s.h, s.err = decodePicture(path)
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

// setBackground makes the picture at path the background, none for "".
// The first picture comes slightly dimmed, so that text stays readable
// over it.
func (a *App) setBackground(path string) {
	a.changeLook(func(s *settings) {
		if path != "" && s.Background == "" && s.PaneOpacity == 0 && s.BackgroundDim == 0 && s.BackgroundBlur == 0 {
			s.BackgroundDim = 0.2
		}
		s.Background = path
	})
}

// backgroundBlur is the blur the settings ask for, in whole DIPs.
func (s settings) backgroundBlur() int {
	return int(math.Round(float64(min(max(s.BackgroundBlur, 0), 24))))
}

// backgroundDim is how much the settings dim the picture.
func (s settings) backgroundDim() float32 {
	return min(max(s.BackgroundDim, 0), 0.8)
}
