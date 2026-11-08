package main

import (
	"math"
	"sync"
)

// Scenes in theme colors: a place (animscenes.go) gives each of its
// colors a role by its hue: its sky and shade take the theme's second
// color, its subject the accent, its lamps the terminals' yellow, its
// beacons their red, its whites the text color. The theme gives the hue,
// in OKLCH; the color keeps its lightness, so that depth, haze and glow
// hold in every theme and a scene stays a dark window in a light theme
// too, and its chroma up to the theme color's. Off (Settings ▸ Motion),
// the scenes keep their own colors.

// role is what a color of a scene paints, of the theme colors.
type role byte

const (
	roleSecond  role = iota // sky, water, shade
	roleAccent              // the subject
	roleYellow              // sun, lamps, windows
	roleRed                 // beacons
	roleMagenta             // a fringe of the subject
	roleText                // whites
)

// sceneRoles are the roles of a scene's colors: by hue, in degrees, for
// a color with a hue, the text color's for one about grey, and over for
// colors whose hue would give them another.
type sceneRoles struct {
	id    string
	byHue func(h float64) role
	over  map[string]role
}

// neutralChroma is the chroma under which a color has no hue to speak of
// and takes the role of the whites.
const neutralChroma = 0.037

func (r *sceneRoles) of(hex string, c rgb) role {
	if o, ok := r.over[hex]; ok {
		return o
	}
	_, ch, h := oklch(c)
	if ch < neutralChroma {
		return roleText
	}
	return r.byHue(math.Mod(h*180/math.Pi+360, 360))
}

func (p animPalette) roleColor(r role) rgb {
	switch r {
	case roleAccent:
		return p.accent
	case roleYellow:
		return p.ansi[3]
	case roleRed:
		return p.ansi[1]
	case roleMagenta:
		return p.ansi[5]
	case roleText:
		return p.text
	}
	return p.second
}

// tones keeps the colors scenes took, by theme, scene and color: a scene
// asks for the same few each frame.
var tones sync.Map

// inks returns how a scene of roles r gets its colors, "#rrggbb": in the
// theme's when the palette has them so, as they are else.
func (p animPalette) inks(r *sceneRoles) func(string) rgb {
	if !p.themed {
		return hexRGB
	}
	return func(hex string) rgb {
		key := p.id + "\x00" + r.id + "\x00" + hex
		if c, ok := tones.Load(key); ok {
			return c.(rgb)
		}
		c := hexRGB(hex)
		c = tone(c, p.roleColor(r.of(hex, c)))
		tones.Store(key, c)
		return c
	}
}

// tone returns c in the hue of to: its lightness as it is, its chroma up
// to that of to, and less where the screen cannot show that much.
func tone(c, to rgb) rgb {
	l, ch, _ := oklch(c)
	_, chTo, h := oklch(to)
	ch = min(ch, chTo)
	if !inGamut(linearOf(l, ch, h)) {
		lo, hi := 0.0, ch
		for range 16 {
			mid := (lo + hi) / 2
			if inGamut(linearOf(l, mid, h)) {
				lo = mid
			} else {
				hi = mid
			}
		}
		ch = lo
	}
	var out rgb
	for i, v := range linearOf(l, ch, h) {
		out[i] = uint8(clamp(srgbOf(clamp(v, 0, 1)), 0, 255) + 0.5)
	}
	return out
}

// oklch returns c's lightness, chroma and hue, in radians, in OKLCH.
func oklch(c rgb) (l, ch, h float64) {
	r, g, b := linearOf8(c[0]), linearOf8(c[1]), linearOf8(c[2])
	lc := math.Cbrt(0.4122214708*r + 0.5363325363*g + 0.0514459929*b)
	mc := math.Cbrt(0.2119034982*r + 0.6806995451*g + 0.1073969566*b)
	sc := math.Cbrt(0.0883024619*r + 0.2817188376*g + 0.6299787005*b)
	l = 0.2104542553*lc + 0.7936177850*mc - 0.0040720468*sc
	a := 1.9779984951*lc - 2.4285922050*mc + 0.4505937099*sc
	bb := 0.0259040371*lc + 0.7827717662*mc - 0.8086757660*sc
	return l, math.Hypot(a, bb), math.Atan2(bb, a)
}

// linearOf returns the linear sRGB of the OKLCH color l, ch, h, which may
// be out of the screen's range.
func linearOf(l, ch, h float64) [3]float64 {
	a, b := ch*math.Cos(h), ch*math.Sin(h)
	lc := math.Pow(l+0.3963377774*a+0.2158037573*b, 3)
	mc := math.Pow(l-0.1055613458*a-0.0638541728*b, 3)
	sc := math.Pow(l-0.0894841775*a-1.2914855480*b, 3)
	return [3]float64{
		4.0767416621*lc - 3.3077115913*mc + 0.2309699292*sc,
		-1.2684380046*lc + 2.6097574011*mc - 0.3413193965*sc,
		-0.0041960863*lc - 0.7034186147*mc + 1.7076147010*sc,
	}
}

func inGamut(v [3]float64) bool {
	for _, x := range v {
		if x < -1e-4 || x > 1+1e-4 {
			return false
		}
	}
	return true
}

func linearOf8(c uint8) float64 {
	v := float64(c) / 255
	if v <= 0.04045 {
		return v / 12.92
	}
	return math.Pow((v+0.055)/1.055, 2.4)
}

func srgbOf(v float64) float64 {
	if v <= 0.0031308 {
		return 255 * 12.92 * v
	}
	return 255 * (1.055*math.Pow(v, 1/2.4) - 0.055)
}

// The roles of the places' colors.
var (
	duneSeaRoles = &sceneRoles{id: "dune-sea", byHue: func(h float64) role {
		switch {
		case h >= 60 && h < 110:
			return roleYellow
		case h >= 30 && h < 60:
			return roleAccent
		}
		return roleSecond
	}, over: map[string]role{"#f08f63": roleYellow, "#ff9a5c": roleYellow}} // the sun and the glow round it, orange as the sand

	neonGridRoles = &sceneRoles{id: "neon-grid", byHue: func(h float64) role {
		switch {
		case h >= 180 && h < 250:
			return roleAccent
		case h >= 10 && h < 40:
			return roleRed
		case h >= 40 && h < 110:
			return roleYellow
		}
		return roleSecond
	}}

	gardenCityRoles = &sceneRoles{id: "garden-city", byHue: func(h float64) role {
		switch {
		case h >= 125 && h < 168:
			return roleAccent
		case h >= 40 && h < 110:
			return roleYellow
		case h < 40 || h >= 330:
			return roleRed
		}
		return roleSecond
	}}

	auroraRoles = &sceneRoles{id: "aurora", byHue: func(h float64) role {
		switch {
		case h >= 100 && h < 200:
			return roleAccent
		case h >= 280 && h < 320:
			return roleMagenta
		}
		return roleSecond
	}}
)
