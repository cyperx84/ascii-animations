package teafx

import (
	"image/color"
	"strings"
	"testing"

	"charm.land/lipgloss/v2"
	uv "github.com/charmbracelet/ultraviolet"

	"github.com/cyperx84/ascii-animations/asciifx/cell"
	"github.com/cyperx84/ascii-animations/asciifx/fx"
	"github.com/cyperx84/ascii-animations/asciifx/tint"
)

func TestCellConversionRoundTrips(t *testing.T) {
	cases := []struct {
		name string
		in   cell.Cell
	}{
		{"plain space", cell.Blank},
		// A zero rune and a space are the same cell to a buffer: the buffer
		// renders 0 as ' ', so the zero cell comes back as a space.
		{"zero rune becomes a space", cell.Cell{}},
		{"fg only", cell.Cell{Rune: 'x', FG: tint.RGB(18, 52, 86)}},
		{"bg only", cell.Cell{Rune: 'x', BG: tint.RGB(200, 100, 50)}},
		{"both colours", cell.Cell{Rune: '░', FG: tint.RGB(1, 2, 3), BG: tint.RGB(4, 5, 6)}},
		{"bold", cell.Cell{Rune: 'b', Attr: cell.Bold}},
		{"dim", cell.Cell{Rune: 'd', Attr: cell.Dim}},
		{"italic", cell.Cell{Rune: 'i', Attr: cell.Italic}},
		{"underline", cell.Cell{Rune: 'u', Attr: cell.Underline}},
		{"reverse", cell.Cell{Rune: 'r', Attr: cell.Reverse}},
		{"half block", cell.Cell{Rune: '▀', FG: tint.RGB(9, 9, 9)}},
		{"braille", cell.Cell{Rune: '⠿', FG: tint.RGB(7, 7, 7)}},
	}
	for _, c := range cases {
		got := FromUV(ToUV(c.in))
		// A buffer's zero rune means blank and is rendered as a space, so the
		// conversion normalises it. Everything else must survive exactly.
		want := c.in
		if want.Rune == 0 {
			want.Rune = ' '
		}
		if got != want {
			t.Errorf("%s: round trip gave %+v, want %+v", c.name, got, want)
		}
	}
}

func TestConversionKeepsUnsetDistinctFromBlack(t *testing.T) {
	// Unset means "the terminal's own foreground", which is not black. Losing
	// that distinction is what makes a fade end on a wrong colour.
	if u := ToUV(cell.Cell{Rune: 'x'}); u.Style.Fg != nil {
		t.Fatalf("an unset colour became %v, want nil", u.Style.Fg)
	}
	black := ToUV(cell.Cell{Rune: 'x', FG: tint.RGB(0, 0, 0)})
	if black.Style.Fg == nil {
		t.Fatal("black was lost as unset")
	}
	if got := FromUV(black).FG; got != tint.RGB(0, 0, 0) {
		t.Fatalf("black round-tripped to %v", got)
	}
	if got := FromUV(&uv.Cell{Content: "x"}).FG; got != tint.None {
		t.Fatalf("a nil colour round-tripped to %v, want None", got)
	}
}

func TestConversionReplacesUnsafeContent(t *testing.T) {
	// A cell buffer holds one rune, so a cluster must not silently shift the
	// columns after it, and must not lose the marks that make it a different
	// character.
	for _, content := range []string{"😀", "e\u0301", "ab", "\n"} {
		if got := FromUV(&uv.Cell{Content: content}).Rune; got != '?' {
			t.Errorf("content %q became %q, want ?", content, got)
		}
	}
	if got := FromUV(&uv.Cell{Content: ""}).Rune; got != ' ' {
		t.Errorf("empty content became %q, want a space", got)
	}
	for _, content := range []string{"▀", "⠿", "#", " "} {
		if got := FromUV(&uv.Cell{Content: content}).Rune; got != []rune(content)[0] {
			t.Errorf("safe content %q became %q", content, got)
		}
	}
}

// TestModelComposesIntoALipglossCanvas is the integration this file exists for:
// an asciifx effect laid out alongside other lipgloss content, with no second
// renderer and no second cell type for the caller to learn.
// Every widget here is pinned with At, because Compose hands a drawable the
// whole canvas.
func TestModelComposesIntoALipglossCanvas(t *testing.T) {
	left := cell.New(4, 2)
	left.WriteString(0, 0, "AB", tint.RGB(255, 0, 0))
	left.WriteString(0, 1, "CD", tint.RGB(0, 255, 0))

	canvas := lipgloss.NewCanvas(12, 3)
	canvas.Compose(At(UV(left), uv.Rect(0, 0, 4, 2)))
	canvas.Compose(At(UV(markerBuffer(6, 1, "RIGHT", tint.RGB(0, 0, 255))), uv.Rect(6, 0, 6, 1)))

	// The styles land in the canvas cell buffer, which is what any later
	// compose step sees. Checking the rendered string alone would hide a
	// profile-dependent mismatch.
	got := canvas.CellAt(0, 0)
	if got.Content != "A" {
		t.Fatalf("cell (0,0) = %q, want A", got.Content)
	}
	r, g, bl, _ := got.Style.Fg.RGBA()
	if r>>8 != 255 || g>>8 != 0 || bl>>8 != 0 {
		t.Fatalf("cell (0,0) colour = %d,%d,%d, want 255,0,0", r>>8, g>>8, bl>>8)
	}
	// The second compose went to its own topleft, not over the first widget.
	if c := canvas.CellAt(6, 0); c.Content != "R" {
		t.Fatalf("cell (6,0) = %q, want R", c.Content)
	}
	if c := canvas.CellAt(4, 0); c.Content != " " {
		t.Fatalf("cell (4,0) = %q, want the gap left by the layout", c.Content)
	}
	// And the string path still carries the glyphs.
	out := canvas.Render()
	for _, want := range []string{"AB", "CD", "RIGHT"} {
		if !strings.Contains(out, want) {
			t.Errorf("rendered canvas lacks %q:\n%s", want, out)
		}
	}
}

func TestBlitClipsToTheArea(t *testing.T) {
	big := cell.New(10, 4)
	big.WriteString(0, 0, "0123456789", tint.None)

	canvas := lipgloss.NewCanvas(6, 2)
	// A 3x1 slot at (1,0): only "012" may appear, and nothing may spill.
	Blit(canvas, uv.Rect(1, 0, 3, 1), big)

	for x, want := range map[int]string{0: " ", 1: "0", 2: "1", 3: "2", 4: " "} {
		if got := canvas.CellAt(x, 0).Content; got != want {
			t.Errorf("cell (%d,0) = %q, want %q", x, got, want)
		}
	}
	if got := canvas.CellAt(0, 1).Content; got != " " {
		t.Errorf("clipping wrote outside the area: (0,1) = %q", got)
	}
}

func TestSnapshotReadsWhatIsOnScreen(t *testing.T) {
	canvas := lipgloss.NewCanvas(6, 3)
	canvas.SetCell(1, 1, &uv.Cell{Content: "X", Width: 1, Style: uv.Style{Fg: mustColor(10, 20, 30)}})

	buf := cell.New(1, 1)
	Snapshot(buf, canvas, uv.Rect(1, 1, 3, 2))
	if buf.W != 3 || buf.H != 2 {
		t.Fatalf("snapshot size %dx%d, want 3x2", buf.W, buf.H)
	}
	got := buf.At(0, 0)
	if got.Rune != 'X' || got.FG != tint.RGB(10, 20, 30) {
		t.Fatalf("snapshot cell = %+v", *got)
	}
	if c := buf.At(2, 1); c.Rune != ' ' {
		t.Fatalf("blank area became %q", c.Rune)
	}
	// Off-screen regions must not panic or read garbage.
	Snapshot(buf, canvas, uv.Rect(4, 2, 10, 10))
	if buf.W != 10 || buf.H != 10 {
		t.Fatalf("clamped snapshot size %dx%d", buf.W, buf.H)
	}
}

// TestContentAnimatesExistingContent is the "wrap my own TUI panel in an
// effect" story: capture a region, hand it to a transition, and the final
// frame is the captured layout rather than a banner.
func TestContentAnimatesExistingContent(t *testing.T) {
	canvas := lipgloss.NewCanvas(20, 3)
	canvas.SetCell(0, 0, &uv.Cell{Content: "P", Width: 1})
	canvas.SetCell(1, 0, &uv.Cell{Content: "A", Width: 1})
	canvas.SetCell(2, 0, &uv.Cell{Content: "N", Width: 1})
	canvas.SetCell(3, 0, &uv.Cell{Content: "E", Width: 1})
	canvas.SetCell(4, 0, &uv.Cell{Content: "L", Width: 1})

	spec, err := fx.Lookup("reveal")
	if err != nil {
		t.Fatal(err)
	}
	r, err := fx.NewRun(spec, fx.Options{
		Seed: 1, W: 20, H: 3,
		Content: Content(canvas, uv.Rect(0, 0, 20, 3)),
	})
	if err != nil {
		t.Fatal(err)
	}
	last, err := r.Seek(r.Frames() - 1)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(strings.TrimSpace(last.Plain()), "PANEL") {
		t.Fatalf("final frame is not the captured content:\n%s", last.Plain())
	}
}

func TestModelIsADrawable(t *testing.T) {
	// Compile-time proof that a Model can be handed to any uv consumer.
	var _ uv.Drawable = Model{}

	m, err := New("spinner", fx.Options{W: 20, H: 1, Params: map[string]string{"style": "dots"}})
	if err != nil {
		t.Fatal(err)
	}
	canvas := lipgloss.NewCanvas(24, 2)
	canvas.Compose(At(m, uv.Rect(0, 0, 24, 2)))

	inked := 0
	for x := 0; x < 24; x++ {
		if c := canvas.CellAt(x, 0); c != nil && strings.TrimSpace(c.Content) != "" {
			inked++
		}
	}
	if inked == 0 {
		t.Fatal("a composed Model drew nothing")
	}
}

// TestAtClipsToTheSmallerArea covers the nesting case: a pinned slot inside a
// parent that hands out a smaller area must not paint outside it.
func TestAtClipsToTheSmallerArea(t *testing.T) {
	b := markerBuffer(8, 1, "abcdefgh", tint.None)
	canvas := lipgloss.NewCanvas(4, 1)
	// The slot wants columns 0..7 but the canvas only has 0..3.
	canvas.Compose(At(UV(b), uv.Rect(0, 0, 8, 1)))
	for x, want := range map[int]string{0: "a", 1: "b", 2: "c", 3: "d"} {
		if got := canvas.CellAt(x, 0).Content; got != want {
			t.Errorf("cell (%d,0) = %q, want %q", x, got, want)
		}
	}
}

func markerBuffer(w, h int, text string, fg tint.Color) *cell.Buffer {
	b := cell.New(w, h)
	b.WriteString(0, 0, text, fg)
	return b
}

func mustColor(r, g, b uint8) color.Color {
	return color.RGBA{R: r, G: g, B: b, A: 0xFF}
}
