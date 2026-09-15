// Package ease provides easing curves: the Penner set, CSS-style cubic-bezier
// and a damped spring. Every Func maps [0,1] to roughly [0,1] with f(0)=0 and
// f(1)=1; Back, Elastic and Spring overshoot in between.
package ease

import (
	"fmt"
	"math"
	"sort"
	"strings"
)

// Func is an easing curve.
type Func func(t float64) float64

const (
	c1 = 1.70158
	c2 = c1 * 1.525
	c3 = c1 + 1
	c4 = (2 * math.Pi) / 3
	c5 = (2 * math.Pi) / 4.5
)

func Linear(t float64) float64 { return t }

func InQuad(t float64) float64    { return t * t }
func OutQuad(t float64) float64   { return 1 - (1-t)*(1-t) }
func InOutQuad(t float64) float64 { return inOut(t, InQuad) }

func InCubic(t float64) float64    { return t * t * t }
func OutCubic(t float64) float64   { return 1 - math.Pow(1-t, 3) }
func InOutCubic(t float64) float64 { return inOut(t, InCubic) }

func InQuart(t float64) float64    { return math.Pow(t, 4) }
func OutQuart(t float64) float64   { return 1 - math.Pow(1-t, 4) }
func InOutQuart(t float64) float64 { return inOut(t, InQuart) }

func InQuint(t float64) float64    { return math.Pow(t, 5) }
func OutQuint(t float64) float64   { return 1 - math.Pow(1-t, 5) }
func InOutQuint(t float64) float64 { return inOut(t, InQuint) }

func InSine(t float64) float64    { return 1 - math.Cos(t*math.Pi/2) }
func OutSine(t float64) float64   { return math.Sin(t * math.Pi / 2) }
func InOutSine(t float64) float64 { return -(math.Cos(math.Pi*t) - 1) / 2 }

func InExpo(t float64) float64 {
	if t == 0 {
		return 0
	}
	return math.Pow(2, 10*t-10)
}

func OutExpo(t float64) float64 {
	if t == 1 {
		return 1
	}
	return 1 - math.Pow(2, -10*t)
}

func InOutExpo(t float64) float64 { return inOut(t, InExpo) }

func InCirc(t float64) float64    { return 1 - math.Sqrt(1-t*t) }
func OutCirc(t float64) float64   { return math.Sqrt(1 - (t-1)*(t-1)) }
func InOutCirc(t float64) float64 { return inOut(t, InCirc) }

func InBack(t float64) float64  { return c3*t*t*t - c1*t*t }
func OutBack(t float64) float64 { return 1 + c3*math.Pow(t-1, 3) + c1*math.Pow(t-1, 2) }

func InOutBack(t float64) float64 {
	if t < 0.5 {
		return (math.Pow(2*t, 2) * ((c2+1)*2*t - c2)) / 2
	}
	return (math.Pow(2*t-2, 2)*((c2+1)*(t*2-2)+c2) + 2) / 2
}

func InElastic(t float64) float64 {
	if t == 0 || t == 1 {
		return t
	}
	return -math.Pow(2, 10*t-10) * math.Sin((t*10-10.75)*c4)
}

func OutElastic(t float64) float64 {
	if t == 0 || t == 1 {
		return t
	}
	return math.Pow(2, -10*t)*math.Sin((t*10-0.75)*c4) + 1
}

func InOutElastic(t float64) float64 {
	switch {
	case t == 0 || t == 1:
		return t
	case t < 0.5:
		return -(math.Pow(2, 20*t-10) * math.Sin((20*t-11.125)*c5)) / 2
	default:
		return (math.Pow(2, -20*t+10)*math.Sin((20*t-11.125)*c5))/2 + 1
	}
}

func OutBounce(t float64) float64 {
	const n1, d1 = 7.5625, 2.75
	switch {
	case t < 1/d1:
		return n1 * t * t
	case t < 2/d1:
		t -= 1.5 / d1
		return n1*t*t + 0.75
	case t < 2.5/d1:
		t -= 2.25 / d1
		return n1*t*t + 0.9375
	default:
		t -= 2.625 / d1
		return n1*t*t + 0.984375
	}
}

func InBounce(t float64) float64 { return 1 - OutBounce(1-t) }

func InOutBounce(t float64) float64 {
	if t < 0.5 {
		return (1 - OutBounce(1-2*t)) / 2
	}
	return (1 + OutBounce(2*t-1)) / 2
}

// SmoothStep is the Hermite 3t²-2t³ curve.
func SmoothStep(t float64) float64 { return t * t * (3 - 2*t) }

func inOut(t float64, in Func) float64 {
	if t < 0.5 {
		return in(2*t) / 2
	}
	return 1 - in(2-2*t)/2
}

// CubicBezier returns a CSS cubic-bezier(x1,y1,x2,y2) curve.
func CubicBezier(x1, y1, x2, y2 float64) Func {
	bez := func(a, b, t float64) float64 {
		u := 1 - t
		return 3*u*u*t*a + 3*u*t*t*b + t*t*t
	}
	return func(x float64) float64 {
		if x <= 0 || x >= 1 {
			return x
		}
		lo, hi := 0.0, 1.0
		t := x
		for i := 0; i < 32; i++ {
			if v := bez(x1, x2, t); math.Abs(v-x) < 1e-6 {
				break
			} else if v < x {
				lo = t
			} else {
				hi = t
			}
			t = (lo + hi) / 2
		}
		return bez(y1, y2, t)
	}
}

// Spring returns an under-damped spring settling at 1. damping in (0,1):
// lower wobbles more. freq is the number of oscillations over the curve.
func Spring(damping, freq float64) Func {
	damping = math.Max(0.01, math.Min(0.999, damping))
	w := 2 * math.Pi * math.Max(freq, 0.1)
	wd := w * math.Sqrt(1-damping*damping)
	return func(t float64) float64 {
		if t <= 0 {
			return 0
		}
		if t >= 1 {
			return 1
		}
		decay := math.Exp(-damping * w * t * 2)
		return 1 - decay*(math.Cos(wd*t)+(damping*w/wd)*math.Sin(wd*t))
	}
}

var named = map[string]Func{
	"linear": Linear, "smooth-step": SmoothStep,
	"in-quad": InQuad, "out-quad": OutQuad, "in-out-quad": InOutQuad,
	"in-cubic": InCubic, "out-cubic": OutCubic, "in-out-cubic": InOutCubic,
	"in-quart": InQuart, "out-quart": OutQuart, "in-out-quart": InOutQuart,
	"in-quint": InQuint, "out-quint": OutQuint, "in-out-quint": InOutQuint,
	"in-sine": InSine, "out-sine": OutSine, "in-out-sine": InOutSine,
	"in-expo": InExpo, "out-expo": OutExpo, "in-out-expo": InOutExpo,
	"in-circ": InCirc, "out-circ": OutCirc, "in-out-circ": InOutCirc,
	"in-back": InBack, "out-back": OutBack, "in-out-back": InOutBack,
	"in-elastic": InElastic, "out-elastic": OutElastic, "in-out-elastic": InOutElastic,
	"in-bounce": InBounce, "out-bounce": OutBounce, "in-out-bounce": InOutBounce,
	"spring": Spring(0.35, 2.5),
}

// Names lists every named curve, sorted.
func Names() []string {
	out := make([]string, 0, len(named))
	for n := range named {
		out = append(out, n)
	}
	sort.Strings(out)
	return out
}

// Parse resolves a curve name, or "cubic-bezier(x1,y1,x2,y2)".
func Parse(s string) (Func, error) {
	s = strings.ToLower(strings.TrimSpace(s))
	if f, ok := named[s]; ok {
		return f, nil
	}
	var x1, y1, x2, y2 float64
	if n, _ := fmt.Sscanf(strings.ReplaceAll(s, " ", ""), "cubic-bezier(%g,%g,%g,%g)", &x1, &y1, &x2, &y2); n == 4 {
		return CubicBezier(x1, y1, x2, y2), nil
	}
	return nil, fmt.Errorf("unknown easing %q: use one of %s, or cubic-bezier(x1,y1,x2,y2)",
		s, strings.Join(Names(), ", "))
}
