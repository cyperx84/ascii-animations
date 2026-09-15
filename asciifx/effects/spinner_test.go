package effects_test

import (
	"testing"
	"time"

	_ "github.com/cyperx84/ascii-animations/asciifx/effects"
	"github.com/cyperx84/ascii-animations/asciifx/fx"
	"github.com/cyperx84/ascii-animations/asciifx/spinner/styles"
)

// TestSpinnerDefaultsToTheCheapPath pins the spinner's defaults to the ones a
// caller inside someone else's view can afford.
//
// A spinner is drawn in a TUI that re-renders its whole screen on every tick,
// so the tick rate is a cost paid by the parent, not by the effect. The default
// must therefore be the cheapest rate that still shows every frame — and the
// highlight, which is the only thing that wants a faster one, must be off.
func TestSpinnerDefaultsToTheCheapPath(t *testing.T) {
	spec, err := fx.Lookup("spinner")
	if err != nil {
		t.Fatal(err)
	}
	v, err := spec.Resolve(nil)
	if err != nil {
		t.Fatal(err)
	}
	if got := v.String("shimmer"); got != "0" {
		t.Errorf("shimmer defaults to %q; the highlight should be opt-in so the cheap path is the default", got)
	}
	// 30fps was the old default: it costs a parent view twice the redraws for
	// a glyph that changes at most 14 times a second.
	if spec.FPS > 20 {
		t.Errorf("spec FPS is %d; a spinner does not need more than the fastest frame set", spec.FPS)
	}
}

// TestSpinnerDefaultRateSkipsNoFrame is the invariant behind the number chosen
// above: at the default tick rate, no frame set may advance by more than one
// frame per tick, or the glyph would visibly jump. Adding a faster style
// without raising the rate fails here rather than on a user's screen.
func TestSpinnerDefaultRateSkipsNoFrame(t *testing.T) {
	spec, err := fx.Lookup("spinner")
	if err != nil {
		t.Fatal(err)
	}
	tick := time.Second / time.Duration(spec.FPS)
	fastest, name := time.Duration(0), ""
	for n, st := range styles.All() {
		if fastest == 0 || st.Interval < fastest {
			fastest, name = st.Interval, n
		}
	}
	if fastest == 0 {
		t.Fatal("no spinner styles")
	}
	if tick > fastest {
		t.Errorf("at %d fps a tick is %v, longer than %q's %v interval: that style will skip frames",
			spec.FPS, tick, name, fastest)
	}
}

// TestSpinnerStylesMatchTheFrameSetTable keeps the style enum and the shared
// table from drifting: the enum is what `-p style=` validates against, and the
// table is what the drop-in spinner package offers, so a name in one and not the
// other is a name that works in one place and not the other.
func TestSpinnerStylesMatchTheFrameSetTable(t *testing.T) {
	spec, err := fx.Lookup("spinner")
	if err != nil {
		t.Fatal(err)
	}
	var enum []string
	for _, p := range spec.Params {
		if p.Name == "style" {
			enum = p.Options
		}
	}
	if len(enum) == 0 {
		t.Fatal("the spinner has no style param")
	}
	table := styles.Names()
	if len(enum) != len(table) {
		t.Fatalf("the enum lists %d styles and the table has %d", len(enum), len(table))
	}
	for i := range enum {
		if enum[i] != table[i] {
			t.Fatalf("style %d is %q in the enum and %q in the table", i, enum[i], table[i])
		}
	}
}
