package ease

import (
	"math"
	"testing"
)

func TestEndpoints(t *testing.T) {
	for _, name := range Names() {
		f, err := Parse(name)
		if err != nil {
			t.Fatal(err)
		}
		if v := f(0); math.Abs(v) > 1e-6 {
			t.Errorf("%s(0)=%v", name, v)
		}
		if v := f(1); math.Abs(v-1) > 1e-6 {
			t.Errorf("%s(1)=%v", name, v)
		}
	}
}

func TestCubicBezier(t *testing.T) {
	f, err := Parse("cubic-bezier(0.25, 0.1, 0.25, 1)")
	if err != nil {
		t.Fatal(err)
	}
	if v := f(0.5); v < 0.7 || v > 0.9 {
		t.Errorf("css ease at 0.5 = %v, want ~0.8", v)
	}
	lin := CubicBezier(0, 0, 1, 1)
	if v := lin(0.3); math.Abs(v-0.3) > 1e-3 {
		t.Errorf("linear bezier(0.3)=%v", v)
	}
}

func TestParseUnknown(t *testing.T) {
	if _, err := Parse("wobbly"); err == nil {
		t.Fatal("unknown easing accepted")
	}
}
