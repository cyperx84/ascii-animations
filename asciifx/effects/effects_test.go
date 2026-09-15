package effects_test

import (
	"testing"

	"github.com/cyperx84/ascii-animations/asciifx/cell"
	_ "github.com/cyperx84/ascii-animations/asciifx/effects"
	"github.com/cyperx84/ascii-animations/asciifx/fx"
	"github.com/cyperx84/ascii-animations/asciifx/tint"
)

var sample = fx.Text("ASCIIFX\nterminal magic", tint.None)

// TestConformance holds every registered effect to the engine contract:
// defaults are valid, output is deterministic for a seed, every glyph is
// single-width, and finite effects end on their content.
func TestConformance(t *testing.T) {
	specs := fx.All()
	if len(specs) == 0 {
		t.Fatal("no effects registered")
	}
	for _, s := range specs {
		t.Run(s.Name, func(t *testing.T) {
			for _, size := range [][2]int{{s.DefW, s.DefH}, {max(s.MinW, 1), max(s.MinH, 1)}, {97, 31}} {
				opts := fx.Options{W: size[0], H: size[1], Seed: 7, Content: sample}
				a, err := fx.NewRun(s, opts)
				if err != nil {
					t.Fatalf("%dx%d: %v", size[0], size[1], err)
				}
				b, _ := fx.NewRun(s, opts)
				ticks := a.Frames()
				if ticks == 0 {
					ticks = 90
				}
				for i := 0; i < ticks; i++ {
					fa, fb := a.Next(), b.Next()
					if !fa.Equal(fb) {
						t.Fatalf("%dx%d tick %d: same seed produced different frames", size[0], size[1], i)
					}
					for _, c := range fa.Cells {
						if c.Rune != 0 && !cell.Safe(c.Rune) {
							t.Fatalf("tick %d: unsafe glyph %q (U+%04X)", i, c.Rune, c.Rune)
						}
					}
				}
				mid := ticks / 2
				seek, err := a.Seek(mid)
				if err != nil {
					t.Fatal(err)
				}
				c, _ := fx.NewRun(s, opts)
				var stepped *cell.Buffer
				for i := 0; i <= mid; i++ {
					stepped = c.Next()
				}
				if !seek.Equal(stepped) {
					t.Fatalf("Seek(%d) disagrees with stepping", mid)
				}
			}
			if s.Example == "" || s.Description == "" || len(s.Glyphs) == 0 {
				t.Error("spec needs Example, Description and Glyphs for the catalog")
			}
		})
	}
}
