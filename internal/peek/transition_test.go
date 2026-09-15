package peek

import (
	"fmt"
	"os"
	"strings"
	"testing"

	_ "github.com/cyperx84/ascii-animations/asciifx/effects"
	"github.com/cyperx84/ascii-animations/asciifx/fx"
	"github.com/cyperx84/ascii-animations/asciifx/term"
	"github.com/cyperx84/ascii-animations/asciifx/tint"
)

// TestPeekTransition prints frames of the transition, content-loop and
// spinner effects so a reader can check each phase looks as intended. Set
// PEEK=name[,name] to limit the output. Finite effects must end on their
// content exactly.
func TestPeekTransition(t *testing.T) {
	banner, err := fx.Banner("ASCIIFX", "block")
	if err != nil {
		t.Fatal(err)
	}
	content := fx.Text(banner, tint.None)
	want := fx.Text(banner, tint.None)
	only := os.Getenv("PEEK")
	show := func(name string) bool { return only == "" || strings.Contains(","+only+",", ","+name+",") }

	for _, name := range []string{"decrypt", "typewriter", "glitch", "beams"} {
		if !show(name) {
			continue
		}
		spec, err := fx.Lookup(name)
		if err != nil {
			t.Error(err)
			continue
		}
		r, err := fx.NewRun(spec, fx.Options{W: 44, H: 7, Seed: 3, Content: content})
		if err != nil {
			t.Fatal(err)
		}
		n := r.Frames()
		for _, frac := range []float64{0.1, 0.35, 0.5, 0.65, 0.85} {
			tick := int(frac * float64(n-1))
			b, _ := r.Seek(tick)
			fmt.Printf("--- %s tick %d/%d plain\n%s\n--- luma\n%s\n", name, tick, n-1, b.Plain(), term.Luma(b))
		}
		last, _ := r.Seek(n - 1)
		ref := last.Clone()
		ref.Clear()
		want(ref)
		if last.Plain() != ref.Plain() {
			t.Errorf("%s: last frame differs from content:\n%s", name, last.Plain())
		}
		fmt.Printf("--- %s last tick %d luma\n%s\n", name, n-1, term.Luma(last))
	}

	for _, name := range []string{"shine", "rainbow"} {
		if !show(name) {
			continue
		}
		for _, tick := range []int{0, 20, 40} {
			b, _, err := fx.Render(name, fx.Options{W: 44, H: 7, Seed: 3, Content: content}, tick)
			if err != nil {
				t.Error(err)
				break
			}
			fmt.Printf("--- %s tick %d luma\n%s\n", name, tick, term.Luma(b))
		}
	}

	if show("spinner") {
		for _, style := range []string{"dots", "arc", "bar", "braille-snake", "star", "blocks"} {
			var rows []string
			for tick := 0; tick < 12; tick += 2 {
				b, _, err := fx.Render("spinner", fx.Options{W: 20, H: 1, Seed: 3, Params: map[string]string{"style": style, "label": "Loading"}}, tick)
				if err != nil {
					t.Error(err)
					break
				}
				rows = append(rows, b.Plain())
			}
			fmt.Printf("--- spinner %s\n%s\n", style, strings.Join(rows, "|\n"))
		}
	}
}
