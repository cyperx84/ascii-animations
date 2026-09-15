package fx

import (
	"fmt"
	"math"
	"sort"
	"strconv"
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
	"checkerboard": func(uint64) Pattern {
		return PatternFunc(func(x, y, _, _ int) float64 { return float64((x + y) & 1) })
	},
	"wave": func(uint64) Pattern {
		// Two wavelengths horizontally over one vertically: a diagonal band
		// whose crests reach 1 and troughs 0.
		return PatternFunc(func(x, y, w, h int) float64 {
			phase := 2*unit(x, w) + unit(y, h)
			return 0.5 + 0.5*math.Sin(2*math.Pi*phase)
		})
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

// PatternGrammar describes the expressions ParsePattern accepts, for the
// catalog and for error hints.
func PatternGrammar() string {
	return "a pattern name, or invert(A), min(A,B), max(A,B), blend(A,B,T) with T in [0,1]"
}

// patternOp is a combinator over one or two child patterns.
type patternOp struct {
	op   string
	a, b Pattern
	t    float64
}

func (p patternOp) Order(x, y, w, h int) float64 {
	switch p.op {
	case "invert":
		return 1 - p.a.Order(x, y, w, h)
	case "min":
		return math.Min(p.a.Order(x, y, w, h), p.b.Order(x, y, w, h))
	case "max":
		return math.Max(p.a.Order(x, y, w, h), p.b.Order(x, y, w, h))
	case "blend":
		av, bv := p.a.Order(x, y, w, h), p.b.Order(x, y, w, h)
		return av + (bv-av)*p.t
	}
	return 0
}

// combinators are the recognised operator names.
var combinators = map[string]bool{"invert": true, "min": true, "max": true, "blend": true}

// ParsePattern resolves a pattern expression: a base pattern name such as
// "center", or a combinator over expressions such as "min(invert(center),
// wave)" or "blend(dissolve,spiral,0.25)".
//
// Combinators follow the shader-style model the Rust and Python engines use:
// a pattern is a spatial ordering in [0,1], and combining orderings mixes how
// one effect spreads without adding a new effect. blend weights are constant
// because an ordering has no time axis; vary the effect's own easing instead
// when a mix must move.
func ParsePattern(expr string, seed uint64) (Pattern, error) {
	p, rest, err := parsePatternExpr(expr, seed)
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(rest) != "" {
		return nil, fmt.Errorf("unexpected %q after the pattern: %s", strings.TrimSpace(rest), PatternGrammar())
	}
	return p, nil
}

func parsePatternExpr(s string, seed uint64) (Pattern, string, error) {
	s = strings.TrimLeft(s, " \t")
	name, rest := patternIdent(s)
	if name == "" {
		return nil, "", fmt.Errorf("expected %s", PatternGrammar())
	}
	lower := strings.ToLower(name)
	if !strings.HasPrefix(rest, "(") {
		mk, ok := patterns[lower]
		if !ok {
			if combinators[lower] {
				return nil, "", fmt.Errorf("pattern %q is a combinator: write %s(...)", name, lower)
			}
			return nil, "", unknownPattern(name)
		}
		return mk(seed), rest, nil
	}
	if !combinators[lower] {
		return nil, "", fmt.Errorf("unknown pattern combinator %q: use %s", name, PatternGrammar())
	}
	inner := rest[1:]
	a, inner, err := parsePatternExpr(inner, seed)
	if err != nil {
		return nil, "", fmt.Errorf("%s: %w", lower, err)
	}
	if lower == "invert" {
		inner, err = patternExpect(inner, ')')
		if err != nil {
			return nil, "", fmt.Errorf("invert takes one pattern: %w", err)
		}
		return patternOp{op: lower, a: a}, inner, nil
	}
	inner, err = patternExpect(inner, ',')
	if err != nil {
		return nil, "", fmt.Errorf("%s takes two patterns: %w", lower, err)
	}
	b, inner, err := parsePatternExpr(inner, seed)
	if err != nil {
		return nil, "", fmt.Errorf("%s: %w", lower, err)
	}
	op := patternOp{op: lower, a: a, b: b}
	if lower == "blend" {
		inner, err = patternExpect(inner, ',')
		if err != nil {
			return nil, "", fmt.Errorf("blend takes a weight: %w", err)
		}
		var t float64
		t, inner, err = patternFloat(inner)
		if err != nil {
			return nil, "", fmt.Errorf("blend: %w", err)
		}
		if t < 0 || t > 1 {
			return nil, "", fmt.Errorf("blend weight %g is outside [0,1]", t)
		}
		op.t = t
	}
	inner, err = patternExpect(inner, ')')
	if err != nil {
		return nil, "", fmt.Errorf("%s: %w", lower, err)
	}
	return op, inner, nil
}

// patternIdent reads a leading name: letters, digits, '-' and '_'.
func patternIdent(s string) (name, rest string) {
	i := 0
	for i < len(s) {
		c := s[i]
		if c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' || c >= '0' && c <= '9' || c == '-' || c == '_' {
			i++
			continue
		}
		break
	}
	return s[:i], s[i:]
}

func patternExpect(s string, want byte) (string, error) {
	s = strings.TrimLeft(s, " \t")
	if s == "" || s[0] != want {
		return s, fmt.Errorf("expected %q, got %q", string(want), truncate(s))
	}
	return s[1:], nil
}

func patternFloat(s string) (float64, string, error) {
	s = strings.TrimLeft(s, " \t")
	i := 0
	for i < len(s) && s[i] != ',' && s[i] != ')' {
		i++
	}
	field := strings.TrimSpace(s[:i])
	v, err := strconv.ParseFloat(field, 64)
	if err != nil || math.IsNaN(v) || math.IsInf(v, 0) {
		return 0, s, fmt.Errorf("want a number in [0,1], got %q", field)
	}
	return v, s[i:], nil
}

func truncate(s string) string {
	if len(s) > 24 {
		return s[:24] + "..."
	}
	return s
}

func unknownPattern(name string) error {
	all := PatternNames()
	if s := closest(name, all); s != "" {
		return fmt.Errorf("unknown pattern %q: did you mean %q? %s", name, s, PatternGrammar())
	}
	return fmt.Errorf("unknown pattern %q: use one of %s", name, strings.Join(all, ", "))
}

// closest returns the name within a small edit distance, or "".
func closest(name string, names []string) string {
	best, bestD := "", 1<<30
	lname := strings.ToLower(name)
	for _, n := range names {
		d := editDistance(lname, n)
		if strings.HasPrefix(n, lname) && len(lname) >= 3 {
			d = min(d, 1)
		}
		if d < bestD {
			best, bestD = n, d
		}
	}
	if bestD <= max(2, len(name)/3) {
		return best
	}
	return ""
}

func editDistance(a, b string) int {
	prev := make([]int, len(b)+1)
	cur := make([]int, len(b)+1)
	for j := range prev {
		prev[j] = j
	}
	for i := 1; i <= len(a); i++ {
		cur[0] = i
		for j := 1; j <= len(b); j++ {
			cost := 1
			if a[i-1] == b[j-1] {
				cost = 0
			}
			cur[j] = min(prev[j]+1, cur[j-1]+1, prev[j-1]+cost)
		}
		prev, cur = cur, prev
	}
	return prev[len(b)]
}
