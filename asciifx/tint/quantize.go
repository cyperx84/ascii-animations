package tint

import "sync"

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

func nearest(c Color, table []Lab) int {
	p := c.OKLab()
	best, bestD := 0, 1e9
	for i, q := range table {
		dl, da, db := p.L-q.L, p.A-q.A, p.B-q.B
		if d := dl*dl + da*da + db*db; d < bestD {
			best, bestD = i, d
		}
	}
	return best
}

func key(c Color) uint32 { return uint32(c.R)<<16 | uint32(c.G)<<8 | uint32(c.B) }

// To256 returns the nearest xterm-256 index by OKLab distance. The 16 base
// colours are skipped because themes remap them.
func To256(c Color) uint8 {
	labOnce.Do(initLab)
	if v, ok := cache256.Load(key(c)); ok {
		return v.(uint8)
	}
	idx := uint8(16 + nearest(c, lab256[16:]))
	cache256.Store(key(c), idx)
	return idx
}

// To16 returns the nearest of the 16 ANSI colours by OKLab distance.
func To16(c Color) uint8 {
	labOnce.Do(initLab)
	if v, ok := cache16.Load(key(c)); ok {
		return v.(uint8)
	}
	idx := uint8(nearest(c, lab16[:]))
	cache16.Store(key(c), idx)
	return idx
}
