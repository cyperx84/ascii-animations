package fx

import (
	"math"
	"math/rand/v2"
	"strings"
	"testing"
)

// orderAt is a convenience for sampling a pattern at one cell of a 11x11 area.
func orderAt(t *testing.T, expr string, x, y int) float64 {
	t.Helper()
	p, err := ParsePattern(expr, 1)
	if err != nil {
		t.Fatalf("ParsePattern(%q): %v", expr, err)
	}
	return p.Order(x, y, 11, 11)
}

func TestPatternShapesExist(t *testing.T) {
	// The shapes the engine survey calls for, minus the four cardinal
	// directions, which are the sweep family.
	for _, want := range []string{"center", "diamond", "spiral", "diagonal", "checkerboard", "wave", "dissolve"} {
		found := false
		for _, n := range PatternNames() {
			found = found || n == want
		}
		if !found {
			t.Errorf("pattern %q is missing; have %s", want, strings.Join(PatternNames(), ", "))
		}
	}
}

func TestPatternCombinators(t *testing.T) {
	// left runs 0 -> 1 across the width, right runs 1 -> 0.
	cases := []struct {
		expr string
		x, y int
		want float64
	}{
		{"left", 0, 0, 0},
		{"left", 10, 0, 1},
		{"invert(left)", 0, 0, 1},
		{"invert(left)", 10, 0, 0},
		{"min(left,right)", 0, 0, 0},
		{"min(left,right)", 5, 0, 0.5},
		{"min(left,right)", 10, 0, 0},
		{"max(left,right)", 0, 0, 1},
		{"max(left,right)", 5, 0, 0.5},
		{"max(left,right)", 10, 0, 1},
		// blend is a constant mix: 0.25 of the way from A to B.
		{"blend(left,right,0.25)", 0, 0, 0.25},
		{"blend(left,right,0.25)", 10, 0, 0.75},
		{"blend(left,right,0)", 0, 0, 0},
		{"blend(left,right,1)", 0, 0, 1},
		// Nesting is the point: a mixed ordering has no single name.
		{"invert(min(left,right))", 0, 0, 1},
		{"invert(min(left,right))", 5, 0, 0.5},
		{"min(invert(center),wave)", 0, 0, 0},
	}
	for _, c := range cases {
		if got := orderAt(t, c.expr, c.x, c.y); math.Abs(got-c.want) > 1e-9 {
			t.Errorf("%s at (%d,%d) = %g, want %g", c.expr, c.x, c.y, got, c.want)
		}
	}
}

func TestPatternCombinatorsStayInRange(t *testing.T) {
	// Every shape, combined every way, must still be an ordering in [0,1],
	// because Local() assumes that.
	var exprs []string
	for _, a := range PatternNames() {
		exprs = append(exprs, a, "invert("+a+")")
		for _, b := range PatternNames() {
			exprs = append(exprs, "min("+a+","+b+")", "max("+a+","+b+")", "blend("+a+","+b+",0.3)")
		}
	}
	for _, e := range exprs {
		p, err := ParsePattern(e, 7)
		if err != nil {
			t.Fatalf("ParsePattern(%q): %v", e, err)
		}
		for y := 0; y < 7; y++ {
			for x := 0; x < 7; x++ {
				if v := p.Order(x, y, 7, 7); v < 0 || v > 1 || math.IsNaN(v) {
					t.Fatalf("%s at (%d,%d) = %g, outside [0,1]", e, x, y, v)
				}
			}
		}
	}
}

func TestPatternWhitespaceIsIgnored(t *testing.T) {
	a := orderAt(t, "min(left,right)", 3, 4)
	b := orderAt(t, " min( left , right ) ", 3, 4)
	if a != b {
		t.Fatalf("whitespace changed the result: %g vs %g", a, b)
	}
}

func TestPatternErrors(t *testing.T) {
	cases := []struct {
		expr string
		want string
	}{
		{"nope", "did you mean"},
		{"centr", "did you mean"},
		{"min(left)", "two patterns"},
		{"invert(left,right)", "one pattern"},
		{"blend(left,right)", "weight"},
		{"blend(left,right,2)", "outside [0,1]"},
		{"blend(left,right,abc)", "want a number"},
		{"min(left,", ""},
		{"min(left,right", "expected"},
		{"min(left,right) extra", "unexpected"},
		{"center(", "unknown pattern combinator"},
		{"", "expected"},
	}
	for _, c := range cases {
		_, err := ParsePattern(c.expr, 1)
		if err == nil {
			t.Errorf("ParsePattern(%q) should fail", c.expr)
			continue
		}
		if c.want != "" && !strings.Contains(err.Error(), c.want) {
			t.Errorf("ParsePattern(%q) error %q lacks %q", c.expr, err, c.want)
		}
	}
}

// TestPatternParamAcceptsCombinators is the end-to-end contract: a spec that
// declares a pattern param validates a combinator expression exactly as it
// validates a bare name.
func TestPatternParamAcceptsCombinators(t *testing.T) {
	spec := &Spec{
		Name:   "t",
		Params: []Param{PatternParamOf("pattern", "center", "order")},
		New:    func(Values, int, int, *rand.Rand) (Effect, error) { return nil, nil },
	}
	v, err := spec.Resolve(map[string]string{"pattern": "blend(dissolve,center,0.4)"})
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if got := v.String("pattern"); got != "blend(dissolve,center,0.4)" {
		t.Fatalf("param round-trip lost the expression: %q", got)
	}
	if _, err := spec.Resolve(map[string]string{"pattern": "blend(center,0.4)"}); err == nil {
		t.Fatal("a malformed combinator should be rejected at validation time")
	}
}
