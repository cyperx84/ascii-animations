package tint

import (
	"fmt"
	"math"
	"testing"
)

// blockError renders a gradient across a region, quantises each cell with the
// 16- or 256-colour palette, and compares the mean colour of each bs×bs block
// of cells against the mean of the ideal colours there. That is the quantity a
// viewer integrates: dithering makes individual cells less accurate on purpose
// so that the local average, which is what the eye sees, becomes more so.
func blockError(prof string, d Dither, w, h, bs int) float64 {
	g := Palettes["rainbow"]
	var total float64
	var blocks int
	for by := 0; by+bs <= h; by += bs {
		for bx := 0; bx+bs <= w; bx += bs {
			var want, got Lab
			for y := by; y < by+bs; y++ {
				for x := bx; x < bx+bs; x++ {
					c := g.At(float64(x) / float64(w-1))
					var q Color
					if prof == "16" {
						q = ansi16[To16At(c, x, y, d)]
					} else {
						q = palette256(int(To256At(c, x, y, d)))
					}
					p, r := c.OKLab(), q.OKLab()
					want.L, want.A, want.B = want.L+p.L, want.A+p.A, want.B+p.B
					got.L, got.A, got.B = got.L+r.L, got.A+r.A, got.B+r.B
				}
			}
			n := float64(bs * bs)
			dl, da, db := (want.L-got.L)/n, (want.A-got.A)/n, (want.B-got.B)/n
			total += math.Sqrt(dl*dl + da*da + db*db)
			blocks++
		}
	}
	return total / float64(blocks)
}

func TestDitherReducesPerceivedError(t *testing.T) {
	for _, prof := range []string{"16", "256"} {
		plain := blockError(prof, NoDither, 256, 8, 8)
		b4 := blockError(prof, Bayer4, 256, 8, 8)
		b8 := blockError(prof, Bayer8, 256, 8, 8)
		t.Logf("%s: none=%.5f bayer4=%.5f bayer8=%.5f", prof, plain, b4, b8)
		if b8 >= plain {
			t.Errorf("%s: bayer8 did not reduce block error (%.5f vs %.5f)", prof, b8, plain)
		}
		if b4 >= plain {
			t.Errorf("%s: bayer4 did not reduce block error (%.5f vs %.5f)", prof, b4, plain)
		}
	}
}

func TestDitherIsPositionStable(t *testing.T) {
	c := MustHex("#3a7bd5")
	for _, d := range []Dither{Bayer4, Bayer8} {
		for _, p := range [][2]int{{0, 0}, {7, 5}, {64, 33}} {
			first := To256At(c, p[0], p[1], d)
			for i := 0; i < 5; i++ {
				if got := To256At(c, p[0], p[1], d); got != first {
					t.Fatalf("%v at %v: %d then %d; dithering must be a pure function of colour and cell", d, p, first, got)
				}
			}
		}
	}
}

func TestDitherPatternIsPeriodic(t *testing.T) {
	c := MustHex("#3a7bd5")
	for _, tc := range []struct {
		d    Dither
		size int
	}{{Bayer4, 4}, {Bayer8, 8}} {
		for y := 0; y < tc.size; y++ {
			for x := 0; x < tc.size; x++ {
				want := To256At(c, x, y, tc.d)
				if got := To256At(c, x+tc.size, y+tc.size, tc.d); got != want {
					t.Fatalf("%v: cell (%d,%d) and (%d,%d) differ (%d vs %d)", tc.d, x, y, x+tc.size, y+tc.size, want, got)
				}
			}
		}
	}
}

func TestDitherOnlyPicksTheTwoNearest(t *testing.T) {
	g := Palettes["rainbow"]
	for x := 0; x < 240; x++ {
		c := g.At(float64(x) / 239)
		p := c.OKLab()
		i0, i1 := twoNearest(p, lab256[16:])
		for y := 0; y < 8; y++ {
			got := int(To256At(c, x, y, Bayer8)) - 16
			if got != i0 && got != i1 {
				t.Fatalf("cell (%d,%d): picked %d, which is neither of the two nearest (%d, %d)", x, y, got, i0, i1)
			}
		}
	}
}

func TestDitherMatchesProjectedFraction(t *testing.T) {
	// A colour sitting a third of the way from its nearest entry to the next
	// should send roughly a third of an 8x8 block to the further entry.
	g := Palettes["ocean"]
	c := g.At(0.45)
	i0, i1 := twoNearest(c.OKLab(), lab256[16:])
	target := between(c.OKLab(), lab256[16+i0], lab256[16+i1])
	if target < 0.08 || target > 0.42 {
		t.Skipf("probe colour projects at %.3f; not a useful fraction", target)
	}
	further := 0
	for y := 0; y < 8; y++ {
		for x := 0; x < 8; x++ {
			if int(To256At(c, x, y, Bayer8))-16 == i1 {
				further++
			}
		}
	}
	got := float64(further) / 64
	if math.Abs(got-target) > 0.15 {
		t.Errorf("further entry chosen %.2f of the time, projection is %.2f", got, target)
	}
}

func TestParseDither(t *testing.T) {
	for s, want := range map[string]Dither{
		"": NoDither, "none": NoDither, "off": NoDither,
		"bayer": Bayer4, "bayer4": Bayer4, "bayer8": Bayer8, "8": Bayer8,
	} {
		got, err := ParseDither(s)
		if err != nil || got != want {
			t.Errorf("ParseDither(%q) = %v, %v; want %v", s, got, err, want)
		}
	}
	if _, err := ParseDither("floyd"); err == nil {
		t.Error("ParseDither should reject error diffusion")
	}
	_ = fmt.Sprint
}
