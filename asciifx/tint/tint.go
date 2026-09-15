// Package tint provides perceptual colour maths for terminal animation:
// OKLab interpolation, multi-stop gradients, spatial gradient mapping and
// quantisation to 256 and 16 colour palettes.
package tint

import (
	"fmt"
	"math"
	"strconv"
	"strings"
)

// Color is an sRGB colour. The zero value is "unset", meaning the terminal's
// default foreground or background; use Set to tell unset from black.
type Color struct {
	R, G, B uint8
	Valid   bool
}

// None is the terminal default colour.
var None = Color{}

// RGB returns a set colour.
func RGB(r, g, b uint8) Color { return Color{r, g, b, true} }

// Hex parses "#rrggbb", "rrggbb" or "#rgb".
func Hex(s string) (Color, error) {
	s = strings.TrimPrefix(strings.TrimSpace(s), "#")
	if len(s) == 3 {
		s = string([]byte{s[0], s[0], s[1], s[1], s[2], s[2]})
	}
	if len(s) != 6 {
		return None, fmt.Errorf("invalid hex colour %q: want #rrggbb", s)
	}
	v, err := strconv.ParseUint(s, 16, 32)
	if err != nil {
		return None, fmt.Errorf("invalid hex colour %q: want #rrggbb", s)
	}
	return RGB(uint8(v>>16), uint8(v>>8), uint8(v)), nil
}

// MustHex is Hex that panics; for package-level palettes.
func MustHex(s string) Color {
	c, err := Hex(s)
	if err != nil {
		panic(err)
	}
	return c
}

// String formats the colour as #rrggbb, or "none".
func (c Color) String() string {
	if !c.Valid {
		return "none"
	}
	return fmt.Sprintf("#%02x%02x%02x", c.R, c.G, c.B)
}

// Lab is a colour in the OKLab space.
type Lab struct{ L, A, B float64 }

func toLinear(v uint8) float64 {
	c := float64(v) / 255
	if c <= 0.04045 {
		return c / 12.92
	}
	return math.Pow((c+0.055)/1.055, 2.4)
}

func fromLinear(c float64) uint8 {
	if c <= 0.0031308 {
		c *= 12.92
	} else {
		c = 1.055*math.Pow(c, 1/2.4) - 0.055
	}
	return uint8(math.Round(clamp01(c) * 255))
}

// OKLab converts the colour to OKLab.
func (c Color) OKLab() Lab {
	r, g, b := toLinear(c.R), toLinear(c.G), toLinear(c.B)
	l := math.Cbrt(0.4122214708*r + 0.5363325363*g + 0.0514459929*b)
	m := math.Cbrt(0.2119034982*r + 0.6806995451*g + 0.1073969566*b)
	s := math.Cbrt(0.0883024619*r + 0.2817188376*g + 0.6299787005*b)
	return Lab{
		L: 0.2104542553*l + 0.7936177850*m - 0.0040720468*s,
		A: 1.9779984951*l - 2.4285922050*m + 0.4505937099*s,
		B: 0.0259040371*l + 0.7827717662*m - 0.8086757660*s,
	}
}

// Color converts an OKLab value back to sRGB, clipping out-of-gamut values.
func (p Lab) Color() Color {
	l := p.L + 0.3963377774*p.A + 0.2158037573*p.B
	m := p.L - 0.1055613458*p.A - 0.0638541728*p.B
	s := p.L - 0.0894841775*p.A - 1.2914855480*p.B
	l, m, s = l*l*l, m*m*m, s*s*s
	return RGB(
		fromLinear(4.0767416621*l-3.3077115913*m+0.2309699292*s),
		fromLinear(-1.2684380046*l+2.6097574011*m-0.3413193965*s),
		fromLinear(-0.0041960863*l-0.7034186147*m+1.7076147010*s),
	)
}

// Lerp blends a to b in OKLab. An unset endpoint takes the other's value.
func Lerp(a, b Color, t float64) Color {
	switch {
	case !a.Valid && !b.Valid:
		return None
	case !a.Valid:
		return b
	case !b.Valid:
		return a
	}
	t = clamp01(t)
	if t == 0 {
		return a
	}
	if t == 1 {
		return b
	}
	pa, pb := a.OKLab(), b.OKLab()
	return Lab{
		L: pa.L + (pb.L-pa.L)*t,
		A: pa.A + (pb.A-pa.A)*t,
		B: pa.B + (pb.B-pa.B)*t,
	}.Color()
}

// Scale multiplies lightness in OKLab: 0 is black, 1 unchanged, >1 brighter.
func Scale(c Color, k float64) Color {
	if !c.Valid {
		return c
	}
	p := c.OKLab()
	p.L = clamp01(p.L * k)
	return p.Color()
}

// Distance is the perceptual distance between two colours.
func Distance(a, b Color) float64 {
	pa, pb := a.OKLab(), b.OKLab()
	dl, da, db := pa.L-pb.L, pa.A-pb.A, pa.B-pb.B
	return math.Sqrt(dl*dl + da*da + db*db)
}

// Gradient is an ordered list of colour stops spaced evenly over [0,1].
type Gradient []Color

// At samples the gradient at t in [0,1].
func (g Gradient) At(t float64) Color {
	switch len(g) {
	case 0:
		return None
	case 1:
		return g[0]
	}
	t = clamp01(t)
	pos := t * float64(len(g)-1)
	i := int(pos)
	if i >= len(g)-1 {
		return g[len(g)-1]
	}
	return Lerp(g[i], g[i+1], pos-float64(i))
}

// Cyclic samples the gradient as a seamless loop, wrapping t into [0,1).
func (g Gradient) Cyclic(t float64) Color {
	if len(g) < 2 {
		return g.At(0)
	}
	t -= math.Floor(t)
	pos := t * float64(len(g))
	i := int(pos) % len(g)
	return Lerp(g[i], g[(i+1)%len(g)], pos-math.Floor(pos))
}

// Steps returns n evenly spaced samples.
func (g Gradient) Steps(n int) []Color {
	out := make([]Color, n)
	for i := range out {
		if n == 1 {
			out[i] = g.At(0)
			continue
		}
		out[i] = g.At(float64(i) / float64(n-1))
	}
	return out
}

// Direction selects how a gradient is laid across a rectangle.
type Direction string

const (
	Horizontal Direction = "horizontal"
	Vertical   Direction = "vertical"
	Diagonal   Direction = "diagonal"
	Radial     Direction = "radial"
)

// Directions lists the valid spatial gradient directions.
var Directions = []string{string(Horizontal), string(Vertical), string(Diagonal), string(Radial)}

// Position maps cell (x,y) in a w×h area to [0,1] along dir. Terminal cells
// are roughly twice as tall as wide, which radial and diagonal account for.
func Position(dir Direction, x, y, w, h int) float64 {
	fx, fy := norm(x, w), norm(y, h)
	switch dir {
	case Vertical:
		return fy
	case Diagonal:
		return (fx*float64(w) + fy*float64(h)*2) / (float64(w) + float64(h)*2)
	case Radial:
		dx := (fx - 0.5) * float64(w)
		dy := (fy - 0.5) * float64(h) * 2
		r := math.Hypot(float64(w)/2, float64(h))
		if r == 0 {
			return 0
		}
		return clamp01(math.Hypot(dx, dy) / r)
	default:
		return fx
	}
}

func norm(v, size int) float64 {
	if size <= 1 {
		return 0
	}
	return float64(v) / float64(size-1)
}

func clamp01(v float64) float64 {
	return math.Max(0, math.Min(1, v))
}
