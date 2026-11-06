package main

import (
	"image"
	"image/color"
	"image/png"
	"math"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/egoist/mygo/ui"
)

// TestMigrateAppearance turns the appearance of earlier versions into
// themes, and leaves a theme chosen since alone.
func TestMigrateAppearance(t *testing.T) {
	for _, tc := range []struct {
		in   settings
		want settings
	}{
		{settings{}, settings{}},
		{settings{Appearance: "dark"}, settings{}},
		{settings{Appearance: "light"}, settings{Theme: "aurora-light"}},
		{settings{Appearance: "system"}, settings{FollowSystem: true}},
		{settings{Appearance: "light", Theme: "ember"}, settings{Theme: "ember"}},
	} {
		got := tc.in
		got.migrate()
		if got != tc.want {
			t.Errorf("%+v migrated to %+v, want %+v", tc.in, got, tc.want)
		}
	}
}

// TestThemeFor picks the theme of each side, and falls back on Aurora's
// for ids unknown or of the other side.
func TestThemeFor(t *testing.T) {
	for _, tc := range []struct {
		s           settings
		dark, light string
	}{
		{settings{}, "aurora", "aurora"},
		{settings{Theme: "paper"}, "paper", "paper"},
		{settings{Theme: "gone"}, "aurora", "aurora"},
		{settings{FollowSystem: true}, "aurora", "aurora-light"},
		{settings{FollowSystem: true, DarkTheme: "ember", LightTheme: "paper"}, "ember", "paper"},
		{settings{FollowSystem: true, DarkTheme: "paper", LightTheme: "tidepool"}, "aurora", "aurora-light"},
	} {
		if d, l := tc.s.themeFor(true).id, tc.s.themeFor(false).id; d != tc.dark || l != tc.light {
			t.Errorf("%+v: dark %s, light %s; want %s, %s", tc.s, d, l, tc.dark, tc.light)
		}
	}
}

// TestThemeContrast keeps every theme's text readable on its panes: the
// text, muted and faint text at 4.5:1 (WCAG AA), and the accent and the
// terminals' colors, but the black and white slots that are the
// background's, at 3:1.
func TestThemeContrast(t *testing.T) {
	for _, th := range themes {
		card := ui.Hex(th.card)
		check := func(what, hex string, least float64) {
			if r := contrast(ui.Hex(hex), card); r < least {
				t.Errorf("%s: %s %s is %.2f:1 on %s, under %.1f:1", th.id, what, hex, r, th.card, least)
			}
		}
		check("text", th.text, 4.5)
		check("muted", th.muted, 4.5)
		check("faint", th.faint, 4.5)
		check("accent", th.accent, 3)
		for i, c := range th.pal {
			if i == 0 || i == 7 || i == 8 || i == 15 {
				continue
			}
			check("color "+string(rune('0'+i%10)), c, 3)
		}
		if r := contrast(ui.Hex(th.onAccent), ui.Hex(th.accent)); r < 4.5 {
			t.Errorf("%s: text on the accent is %.2f:1", th.id, r)
		}
	}
}

func contrast(a, b ui.Color) float64 {
	l := func(c ui.Color) float64 {
		ch := func(v uint8) float64 {
			s := float64(v) / 255
			if s <= 0.03928 {
				return s / 12.92
			}
			return math.Pow((s+0.055)/1.055, 2.4)
		}
		return 0.2126*ch(c.R) + 0.7152*ch(c.G) + 0.0722*ch(c.B)
	}
	la, lb := l(a), l(b)
	return (max(la, lb) + 0.05) / (min(la, lb) + 0.05)
}

// TestThemeCards chooses themes with their cards, then follows the
// system: a card then takes its side.
func TestThemeCards(t *testing.T) {
	saved := prefs
	t.Cleanup(func() { prefs = saved; applyFont() })
	prefs = settings{FontSize: defaultFontSize}
	_, tt := newTestApp(t)
	tt.Frame()
	click := clicker(t, tt)
	click("Settings")
	click("Ember")
	if prefs.Theme != "ember" {
		t.Fatalf("Ember's card made the theme %q", prefs.Theme)
	}
	click("Match system appearance")
	if !prefs.FollowSystem || prefs.DarkTheme != "ember" {
		t.Fatalf("following the system: %+v", prefs)
	}
	click("Paper")
	if prefs.LightTheme != "paper" || prefs.DarkTheme != "ember" {
		t.Fatalf("Paper's card following the system: %+v", prefs)
	}
	// The window shows one of the two, as the tester is dark or light.
	sides := []string{prefs.themeFor(true).id, prefs.themeFor(false).id}
	click("Match system appearance")
	if prefs.FollowSystem {
		t.Fatal("the switch did not stop following the system")
	}
	if !slices.Contains(sides, prefs.Theme) {
		t.Fatalf("stopping following the system showed %q, neither of %q", prefs.Theme, sides)
	}
}

// TestFontPicker lists the system's monospaced fonts for the terminals,
// and the others on asking, and keeps the one clicked.
func TestFontPicker(t *testing.T) {
	saved, font := prefs, termFont
	t.Cleanup(func() { prefs, termFont = saved, font })
	prefs = settings{FontSize: defaultFontSize}
	applyFont()
	a, tt := newTestApp(t)
	a.fontsAsked = true
	a.fonts = []ui.FontFamily{
		{Name: "JetBrains Mono", Monospaced: true, Registered: true},
		{Name: "Fira Code", Monospaced: true},
		{Name: "Noto Sans"},
	}
	tt.Frame()
	click := clicker(t, tt)
	click("Settings")
	click("Text")
	click("JetBrains Mono") // the button of the terminals' font
	if _, ok := tt.Find("Noto Sans"); ok {
		t.Error("the terminals' fonts list a proportional font")
	}
	click("Fira Code")
	if prefs.Font != "Fira Code" || !strings.HasPrefix(termFont.Family, "Fira Code,") {
		t.Fatalf("Fira Code made the font %q, the terminals' %q", prefs.Font, termFont.Family)
	}
	click("Fira Code")
	click("Show proportional fonts")
	if _, ok := tt.Find("Noto Sans"); !ok {
		t.Error("no proportional font after asking for them")
	}
	click("JetBrains Mono")
	if prefs.Font != "" || termFont.Family != "JetBrains Mono, monospace" {
		t.Errorf("the built-in font is kept as %q, the terminals' %q", prefs.Font, termFont.Family)
	}
	click("System") // the button of the window's font
	click("Noto Sans")
	if prefs.UIFont != "Noto Sans" {
		t.Errorf("the interface font is %q", prefs.UIFont)
	}
}

// TestBackground reads a picture, blurs it, and tells of one it cannot
// read.
func TestBackground(t *testing.T) {
	saved := prefs
	t.Cleanup(func() { prefs = saved })
	dir := t.TempDir()
	path := filepath.Join(dir, "wall.png")
	img := image.NewRGBA(image.Rect(0, 0, 80, 40))
	for y := range 40 {
		for x := range 80 {
			if (x/4+y/4)%2 == 0 {
				img.Set(x, y, color.White)
			} else {
				img.Set(x, y, color.Black)
			}
		}
	}
	f, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	png.Encode(f, img)
	f.Close()

	a := &App{}
	wait := func() *backdrop {
		t.Helper()
		for range 200 {
			a.runPosted()
			if b := a.background(); b != nil {
				return b
			}
			time.Sleep(10 * time.Millisecond)
		}
		t.Fatalf("no background for %s", prefs.Background)
		return nil
	}
	prefs = settings{Background: path}
	if b := wait(); b.bmp == nil || b.w != 80 || b.h != 40 || b.err != "" {
		t.Fatalf("the picture: %+v", b)
	}
	prefs.Background = filepath.Join(dir, "gone.png")
	if b := wait(); b.bmp != nil || !strings.Contains(b.err, "gone.png") {
		t.Fatalf("a picture that is not there: %+v", b)
	}
}

// TestBlur averages a checkerboard into gray, keeping the picture's
// shape.
func TestBlur(t *testing.T) {
	img := image.NewRGBA(image.Rect(0, 0, 200, 100))
	for y := range 100 {
		for x := range 200 {
			v := uint8(0)
			if (x+y)%2 == 0 {
				v = 255
			}
			img.SetRGBA(x, y, color.RGBA{v, v, v, 255})
		}
	}
	b := blurred(img, 9)
	w, h := b.Bounds().Dx(), b.Bounds().Dy()
	if w >= 200 || abs32(float32(w)/float32(h)-2) > 0.1 {
		t.Fatalf("the blurred picture is %d×%d", w, h)
	}
	c := b.RGBAAt(w/2, h/2)
	if c.R < 100 || c.R > 155 || c.A != 255 {
		t.Errorf("the middle of the blurred checkerboard is %v", c)
	}
	for _, sigma := range []float64{1, 3, 8} {
		var sum float64
		for _, s := range boxesForGauss(sigma, 3) {
			if s%2 == 0 {
				t.Errorf("an even box of %d for sigma %g", s, sigma)
			}
			sum += float64(s*s-1) / 12
		}
		if math.Abs(math.Sqrt(sum)-sigma) > 0.5 {
			t.Errorf("the boxes for sigma %g blur by %.2f", sigma, math.Sqrt(sum))
		}
	}
}

// clicker clicks the element of a label in the tester's frames.
func clicker(t *testing.T, tt *ui.Tester) func(string) {
	return func(label string) {
		t.Helper()
		r, ok := tt.Find(label)
		if !ok {
			t.Fatalf("no %q; texts %q", label, tt.Texts())
		}
		tt.Move(r.X+r.W/2, r.Y+r.H/2)
		tt.Frame()
		tt.Press(r.X+r.W/2, r.Y+r.H/2)
		tt.Frame()
		tt.Release(r.X+r.W/2, r.Y+r.H/2)
		tt.Frame()
	}
}

// TestOverImage backs the panes and the title bar with their colors,
// translucent as the settings ask while a background image shows.
func TestOverImage(t *testing.T) {
	saved := prefs
	t.Cleanup(func() { prefs = saved })
	prefs = settings{PaneOpacity: 0.6}
	k := themeColors["paper"]
	a := &App{}
	if got := a.overImage(k.bg); got != k.bg {
		t.Errorf("without an image the title bar's face is %v, not %v", got, k.bg)
	}
	a.bgShown = true
	if got := a.overImage(k.bg); got.A != 153 || got.R != k.bg.R {
		t.Errorf("over an image the title bar's face is %v", got)
	}
}

// TestPatterns draws each of Lex's backgrounds, opaque and not of one
// color, and names them as the settings keep them.
func TestPatterns(t *testing.T) {
	for _, p := range patterns {
		got, ok := patternOf(patternPrefix + p.id)
		if !ok || got.id != p.id {
			t.Errorf("%q names no pattern", patternPrefix+p.id)
		}
		img := p.render(192, 108, 0.1)
		lo, hi := 255, 0
		for i := 0; i < len(img.Pix); i += 4 {
			if img.Pix[i+3] != 255 {
				t.Fatalf("%s is not opaque", p.id)
			}
			l := int(img.Pix[i]) + int(img.Pix[i+1]) + int(img.Pix[i+2])
			lo, hi = min(lo, l), max(hi, l)
		}
		if hi-lo < 6 {
			t.Errorf("%s is all of one color (%d to %d)", p.id, lo, hi)
		}
	}
	if _, ok := patternOf("/home/me/lex:nebula.png"); ok {
		t.Error("a picture's path names a pattern")
	}
}

// TestBackgroundThumbs chooses a pattern and the picture of one's own
// with their thumbnails, and keeps that picture while a pattern shows.
func TestBackgroundThumbs(t *testing.T) {
	saved := prefs
	t.Cleanup(func() { prefs = saved })
	prefs = settings{FontSize: defaultFontSize, Background: "/nowhere/wall.png", CustomBackground: "/nowhere/wall.png", BackgroundFit: "tile"}
	_, tt := newTestApp(t)
	tt.Frame()
	click := clicker(t, tt)
	click("Settings")
	click("Nebula")
	if prefs.Background != "lex:nebula" || prefs.CustomBackground != "/nowhere/wall.png" || prefs.backgroundFit() != "fill" {
		t.Fatalf("after Nebula: %+v", prefs)
	}
	click("wall.png")
	if prefs.Background != "/nowhere/wall.png" || prefs.backgroundFit() != "tile" {
		t.Fatalf("after the picture: %+v", prefs)
	}
	click("None")
	if prefs.Background != "" {
		t.Fatalf("after None: %+v", prefs)
	}
}
