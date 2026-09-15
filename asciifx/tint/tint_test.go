package tint

import (
	"math"
	"testing"
)

func TestOKLabRoundTrip(t *testing.T) {
	for _, c := range []Color{RGB(0, 0, 0), RGB(255, 255, 255), RGB(255, 0, 0), RGB(18, 200, 77), RGB(90, 30, 250)} {
		got := c.OKLab().Color()
		if d := math.Abs(float64(got.R)-float64(c.R)) + math.Abs(float64(got.G)-float64(c.G)) + math.Abs(float64(got.B)-float64(c.B)); d > 3 {
			t.Errorf("%v round-tripped to %v", c, got)
		}
	}
}

func TestLerpEndpointsAndUnset(t *testing.T) {
	a, b := MustHex("#ff0000"), MustHex("#0000ff")
	if Lerp(a, b, 0) != a || Lerp(a, b, 1) != b {
		t.Fatal("endpoints not preserved")
	}
	if Lerp(None, b, 0.3) != b || Lerp(a, None, 0.7) != a {
		t.Fatal("unset endpoint should take the other colour")
	}
	// OKLab midpoint of red and blue should not collapse to muddy dark grey.
	if mid := Lerp(a, b, 0.5); mid.OKLab().L < 0.5 {
		t.Errorf("midpoint too dark: %v", mid)
	}
}

func TestGradientSampling(t *testing.T) {
	g := Palettes["fire"]
	if g.At(0) != g[0] || g.At(1) != g[len(g)-1] {
		t.Fatal("gradient endpoints wrong")
	}
	if g.Cyclic(0) != g.Cyclic(1) {
		t.Fatal("cyclic gradient must wrap seamlessly")
	}
	if n := len(g.Steps(37)); n != 37 {
		t.Fatalf("Steps returned %d", n)
	}
}

func TestParsePalette(t *testing.T) {
	if _, err := ParsePalette("synthwave"); err != nil {
		t.Fatal(err)
	}
	g, err := ParsePalette("#ff0000,#00f")
	if err != nil || len(g) != 2 || g[1] != RGB(0, 0, 255) {
		t.Fatalf("hex stops: %v %v", g, err)
	}
	if _, err := ParsePalette("nope"); err == nil {
		t.Fatal("unknown palette accepted")
	}
}

func TestQuantize(t *testing.T) {
	if To16(RGB(250, 0, 0)) != 9 {
		t.Errorf("bright red -> %d", To16(RGB(250, 0, 0)))
	}
	if i := To256(RGB(255, 255, 255)); i != 231 && i != 255 {
		t.Errorf("white -> %d", i)
	}
	if To256(RGB(10, 10, 10)) < 16 {
		t.Error("256 quantisation must skip themeable base colours")
	}
}

func TestPositionRange(t *testing.T) {
	for _, d := range Directions {
		for y := 0; y < 7; y++ {
			for x := 0; x < 13; x++ {
				if v := Position(Direction(d), x, y, 13, 7); v < 0 || v > 1 {
					t.Fatalf("%s(%d,%d)=%v", d, x, y, v)
				}
			}
		}
	}
}
