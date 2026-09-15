package tint

import (
	"math"
	"sync"
)

// ansi16 approximates the xterm default 16-colour palette. Users theme these,
// so the match is a best guess; semantic roles matter more than exact RGB.
var ansi16 = [16]Color{
	RGB(0, 0, 0), RGB(205, 0, 0), RGB(0, 205, 0), RGB(205, 205, 0),
	RGB(0, 0, 238), RGB(205, 0, 205), RGB(0, 205, 205), RGB(229, 229, 229),
	RGB(127, 127, 127), RGB(255, 0, 0), RGB(0, 255, 0), RGB(255, 255, 0),
	RGB(92, 92, 255), RGB(255, 0, 255), RGB(0, 255, 255), RGB(255, 255, 255),
}

var (
	labOnce  sync.Once
	lab16    [16]Lab
	lab256   [256]Lab
	cache256 sync.Map // uint32 -> uint8
	cache16  sync.Map
	// Dithering needs the two nearest entries and how far between them the
	// colour sits. That depends only on the colour, not the cell, so it is
	// cached once per colour: the cache stays exactly the size of the
	// undithered one however many cells are rendered. pair256/pair16 hold
	// pairEntry values.
	pair256 sync.Map
	pair16  sync.Map
)

func initLab() {
	for i, c := range ansi16 {
		lab16[i] = c.OKLab()
	}
	for i := 0; i < 256; i++ {
		lab256[i] = palette256(i).OKLab()
	}
}

func palette256(i int) Color {
	switch {
	case i < 16:
		return ansi16[i]
	case i < 232:
		i -= 16
		level := func(v int) uint8 {
			if v == 0 {
				return 0
			}
			return uint8(55 + v*40)
		}
		return RGB(level(i/36), level(i/6%6), level(i%6))
	default:
		v := uint8(8 + (i-232)*10)
		return RGB(v, v, v)
	}
}

// key packs a colour for the quantisation caches.
func key(c Color) uint32 { return uint32(c.R)<<16 | uint32(c.G)<<8 | uint32(c.B) }

// nearestLab returns the index of the closest entry to p.
func nearestLab(p Lab, table []Lab) int {
	best, bestD := 0, math.Inf(1)
	for i, q := range table {
		dl, da, db := p.L-q.L, p.A-q.A, p.B-q.B
		if d := dl*dl + da*da + db*db; d < bestD {
			best, bestD = i, d
		}
	}
	return best
}

// twoNearest returns the indices of the closest and second-closest entries.
func twoNearest(p Lab, table []Lab) (int, int) {
	i0, i1 := 0, 0
	d0, d1 := math.Inf(1), math.Inf(1)
	for i, q := range table {
		dl, da, db := p.L-q.L, p.A-q.A, p.B-q.B
		d := dl*dl + da*da + db*db
		switch {
		case d < d0:
			i0, i1, d0, d1 = i, i0, d, d0
		case d < d1:
			i1, d1 = i, d
		}
	}
	return i0, i1
}

// between returns where p projects onto the segment a-b, clamped to [0,1].
func between(p, a, b Lab) float64 {
	dl, da, db := b.L-a.L, b.A-a.A, b.B-a.B
	den := dl*dl + da*da + db*db
	if den == 0 {
		return 0
	}
	t := ((p.L-a.L)*dl + (p.A-a.A)*da + (p.B-a.B)*db) / den
	return math.Max(0, math.Min(1, t))
}

// pairEntry is a colour's two nearest palette entries and how far the colour
// projects from the first toward the second.
type pairEntry struct {
	i0, i1 int
	t      float64
}

// pairFor returns the cached pair for a colour. The cache is keyed by colour
// alone, so dithering does not multiply its size.
func pairFor(c Color, table []Lab, cache *sync.Map) pairEntry {
	k := key(c)
	if v, ok := cache.Load(k); ok {
		return v.(pairEntry)
	}
	p := c.OKLab()
	i0, i1 := twoNearest(p, table)
	e := pairEntry{i0: i0, i1: i1}
	if i0 != i1 {
		e.t = between(p, table[i0], table[i1])
	}
	cache.Store(k, e)
	return e
}

// To256At is To256 with ordered dithering for the value at cell (x,y). The
// result is a pure function of the colour, the pattern and the cell's
// position in it, so a still region quantises to a fixed stipple.
func To256At(c Color, x, y int, d Dither) uint8 {
	if d == NoDither {
		return To256(c)
	}
	labOnce.Do(initLab)
	e := pairFor(c, lab256[16:], &pair256)
	i := e.i0
	if e.i0 != e.i1 && d.threshold(x, y) < e.t {
		i = e.i1
	}
	return uint8(16 + i)
}

// To16At is To16 with ordered dithering for the value at cell (x,y).
func To16At(c Color, x, y int, d Dither) uint8 {
	if d == NoDither {
		return To16(c)
	}
	labOnce.Do(initLab)
	e := pairFor(c, lab16[:], &pair16)
	if e.i0 != e.i1 && d.threshold(x, y) < e.t {
		return uint8(e.i1)
	}
	return uint8(e.i0)
}

// To256 returns the nearest xterm-256 index by OKLab distance. The 16 base
// colours are skipped because themes remap them.
func To256(c Color) uint8 {
	labOnce.Do(initLab)
	if v, ok := cache256.Load(key(c)); ok {
		return v.(uint8)
	}
	idx := uint8(16 + nearestLab(c.OKLab(), lab256[16:]))
	cache256.Store(key(c), idx)
	return idx
}

// To16 returns the nearest of the 16 ANSI colours by OKLab distance.
func To16(c Color) uint8 {
	labOnce.Do(initLab)
	if v, ok := cache16.Load(key(c)); ok {
		return v.(uint8)
	}
	idx := uint8(nearestLab(c.OKLab(), lab16[:]))
	cache16.Store(key(c), idx)
	return idx
}
