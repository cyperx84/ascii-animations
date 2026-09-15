package fx

import (
	"fmt"
	"math"
	"sort"
	"strings"
)

// Pattern spreads a transition across space. Order returns where a cell sits
// in the reveal order, in [0,1]: 0 goes first, 1 goes last. Local turns that
// order into per-cell progress.
type Pattern interface {
	Order(x, y, w, h int) float64
}

// PatternFunc adapts a function to Pattern.
type PatternFunc func(x, y, w, h int) float64

func (p PatternFunc) Order(x, y, w, h int) float64 { return p(x, y, w, h) }

// Local maps global progress p to a cell's own progress. softness in (0,1]
// is the width of the moving edge as a fraction of the whole: small values
// give a hard wipe, large values a long gradient where many cells animate at
// once. This soft edge is what makes wipes look anti-aliased.
func Local(p, order, softness float64) float64 {
	softness = math.Max(1e-3, math.Min(1, softness))
	v := (p*(1+softness) - order) / softness
	return math.Max(0, math.Min(1, v))
}

func unit(v, size int) float64 {
	if size <= 1 {
		return 0
	}
	return float64(v) / float64(size-1)
}

// cellAspect is how many columns look as tall as one row.
const cellAspect = 2.0

func radial(x, y, w, h int) float64 {
	dx := float64(x) - float64(w-1)/2
	dy := (float64(y) - float64(h-1)/2) * cellAspect
	r := math.Hypot(float64(w-1)/2, float64(h-1)/2*cellAspect)
	if r == 0 {
		return 0
	}
	return math.Hypot(dx, dy) / r
}

var patterns = map[string]func(seed uint64) Pattern{
	"left": func(uint64) Pattern {
		return PatternFunc(func(x, _, w, _ int) float64 { return unit(x, w) })
	},
	"right": func(uint64) Pattern {
		return PatternFunc(func(x, _, w, _ int) float64 { return 1 - unit(x, w) })
	},
	"top": func(uint64) Pattern {
		return PatternFunc(func(_, y, _, h int) float64 { return unit(y, h) })
	},
	"bottom": func(uint64) Pattern {
		return PatternFunc(func(_, y, _, h int) float64 { return 1 - unit(y, h) })
	},
	"diagonal": func(uint64) Pattern {
		return PatternFunc(func(x, y, w, h int) float64 {
			span := float64(w-1) + float64(h-1)*cellAspect
			if span <= 0 {
				return 0
			}
			return (float64(x) + float64(y)*cellAspect) / span
		})
	},
	"center": func(uint64) Pattern { return PatternFunc(radial) },
	"edges": func(uint64) Pattern {
		return PatternFunc(func(x, y, w, h int) float64 { return 1 - radial(x, y, w, h) })
	},
	"diamond": func(uint64) Pattern {
		return PatternFunc(func(x, y, w, h int) float64 {
			dx := math.Abs(float64(x) - float64(w-1)/2)
			dy := math.Abs(float64(y)-float64(h-1)/2) * cellAspect
			m := float64(w-1)/2 + float64(h-1)/2*cellAspect
			if m == 0 {
				return 0
			}
			return (dx + dy) / m
		})
	},
	"middle-out": func(uint64) Pattern {
		return PatternFunc(func(x, _, w, _ int) float64 {
			if w <= 1 {
				return 0
			}
			return math.Abs(float64(x)-float64(w-1)/2) / (float64(w-1) / 2)
		})
	},
	"spiral": func(uint64) Pattern {
		return PatternFunc(func(x, y, w, h int) float64 {
			dx := float64(x) - float64(w-1)/2
			dy := (float64(y) - float64(h-1)/2) * cellAspect
			angle := (math.Atan2(dy, dx) + math.Pi) / (2 * math.Pi)
			r := math.Hypot(dx, dy) / math.Max(1, math.Hypot(float64(w-1)/2, float64(h-1)/2*cellAspect))
			return math.Mod(angle*0.35+r*0.65, 1)
		})
	},
	"dissolve": func(seed uint64) Pattern {
		return PatternFunc(func(x, y, _, _ int) float64 { return Hash01(x, y, seed) })
	},
	"rain": func(seed uint64) Pattern {
		return PatternFunc(func(x, y, _, h int) float64 {
			return unit(y, h)*0.55 + Hash01(x, 0, seed)*0.45
		})
	},
}

// PatternNames lists the built-in patterns, sorted.
func PatternNames() []string {
	out := make([]string, 0, len(patterns))
	for n := range patterns {
		out = append(out, n)
	}
	sort.Strings(out)
	return out
}

// ParsePattern resolves a pattern name. seed only affects random patterns.
func ParsePattern(name string, seed uint64) (Pattern, error) {
	mk, ok := patterns[strings.ToLower(strings.TrimSpace(name))]
	if !ok {
		return nil, fmt.Errorf("unknown pattern %q: use one of %s", name, strings.Join(PatternNames(), ", "))
	}
	return mk(seed), nil
}
