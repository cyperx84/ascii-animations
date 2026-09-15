package peek

import (
	"fmt"
	"os"
	"strings"
	"testing"

	_ "github.com/cyperx84/ascii-animations/asciifx/effects"
	"github.com/cyperx84/ascii-animations/asciifx/fx"
	"github.com/cyperx84/ascii-animations/asciifx/term"
)

// TestPeekAmbient prints ambient effect frames as text and luma so a person
// or agent can eyeball their shapes. PEEK=name,name limits the effects and
// PEEKTICKS=a,b the ticks.
func TestPeekAmbient(t *testing.T) {
	names := []string{"matrix", "plasma", "starfield", "rain", "snow", "dna", "aurora", "donut", "pipes"}
	if v := os.Getenv("PEEK"); v != "" {
		names = strings.Split(v, ",")
	}
	ticks := []int{10, 60}
	if v := os.Getenv("PEEKTICKS"); v != "" {
		ticks = nil
		for _, s := range strings.Split(v, ",") {
			var n int
			fmt.Sscan(s, &n)
			ticks = append(ticks, n)
		}
	}
	for _, name := range names {
		for _, tick := range ticks {
			b, _, err := fx.Render(name, fx.Options{W: 72, H: 20, Seed: 3}, tick)
			if err != nil {
				t.Fatal(err)
			}
			t.Logf("=== %s tick %d plain\n%s", name, tick, b.Plain())
			t.Logf("=== %s tick %d luma\n%s", name, tick, term.Luma(b))
			var cols []string
			for _, pt := range [][2]int{{5, 3}, {36, 10}, {60, 16}, {20, 18}} {
				c := b.At(pt[0], pt[1])
				cols = append(cols, fmt.Sprintf("(%d,%d)%q fg=%s bg=%s", pt[0], pt[1], c.Rune, c.FG, c.BG))
			}
			t.Logf("=== %s tick %d colours: %s", name, tick, strings.Join(cols, "  "))
		}
	}
}
