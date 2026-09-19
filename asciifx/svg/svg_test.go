package svg_test

import (
	"bytes"
	"encoding/xml"
	"io"
	"strings"
	"testing"

	"github.com/cyperx84/ascii-animations/asciifx/cell"
	_ "github.com/cyperx84/ascii-animations/asciifx/effects"
	"github.com/cyperx84/ascii-animations/asciifx/fx"
	"github.com/cyperx84/ascii-animations/asciifx/svg"
	"github.com/cyperx84/ascii-animations/asciifx/tint"
)

// frames renders an effect into the slice Encode wants, cloning as the
// package doc says to.
func frames(t *testing.T, effect string, n int, o fx.Options) []*cell.Buffer {
	t.Helper()
	spec, err := fx.Lookup(effect)
	if err != nil {
		t.Fatal(err)
	}
	run, err := fx.NewRun(spec, o)
	if err != nil {
		t.Fatal(err)
	}
	out := make([]*cell.Buffer, 0, n)
	for tick := 0; tick < n; tick++ {
		b, err := run.Seek(tick)
		if err != nil {
			t.Fatal(err)
		}
		out = append(out, b.Clone())
	}
	return out
}

// The document has to parse as XML, or no browser will show it.
func TestOutputIsWellFormedXML(t *testing.T) {
	var buf bytes.Buffer
	if err := svg.Encode(&buf, frames(t, "fire", 4, fx.Options{W: 12, H: 6, Seed: 1}), svg.Options{}); err != nil {
		t.Fatal(err)
	}
	dec := xml.NewDecoder(bytes.NewReader(buf.Bytes()))
	for {
		_, err := dec.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatalf("not well-formed: %v", err)
		}
	}
}

// One keyframe rule per frame, and one group per frame, is what makes
// exactly one frame visible at a time.
func TestEveryFrameGetsItsOwnKeyframe(t *testing.T) {
	const n = 7
	var buf bytes.Buffer
	if err := svg.Encode(&buf, frames(t, "matrix", n, fx.Options{W: 10, H: 5, Seed: 2}), svg.Options{FPS: 20}); err != nil {
		t.Fatal(err)
	}
	out := buf.String()
	for i := 0; i < n; i++ {
		for _, want := range []string{"@keyframes k" + itoa(i), `class="f f` + itoa(i) + `"`} {
			if !strings.Contains(out, want) {
				t.Errorf("missing %q", want)
			}
		}
	}
	if strings.Contains(out, "@keyframes k"+itoa(n)) {
		t.Error("a keyframe exists for a frame that does not")
	}
	// The cycle is the frame count over the rate, so the animation plays at
	// the speed the run was built for.
	if !strings.Contains(out, "animation:0.35s") {
		t.Errorf("wrong cycle length in:\n%s", out[:min(len(out), 400)])
	}
}

// Nothing may reach outside the file: no script, no font, no fetch. That is
// what lets it render inside an <img>.
func TestDocumentIsSelfContained(t *testing.T) {
	var buf bytes.Buffer
	if err := svg.Encode(&buf, frames(t, "aurora", 3, fx.Options{W: 14, H: 6, Seed: 5}), svg.Options{}); err != nil {
		t.Fatal(err)
	}
	// Past the opening tag, because the SVG namespace is a URL and is not a
	// fetch.
	body := buf.String()
	body = body[strings.Index(body, "\n")+1:]
	for _, bad := range []string{"<script", "xlink:href", "http://", "https://", "@import", "url("} {
		if strings.Contains(body, bad) {
			t.Errorf("document references %q", bad)
		}
	}
}

// Block glyphs are drawn as rectangles rather than text, because a font
// leaves a seam between rows where a terminal leaves none.
func TestBlockGlyphsBecomeRectangles(t *testing.T) {
	b := cell.New(3, 1)
	b.SetRune(0, 0, '█', tint.MustHex("#ff0000"))
	b.SetRune(1, 0, '█', tint.MustHex("#ff0000"))
	b.SetRune(2, 0, 'x', tint.MustHex("#00ff00"))
	var buf bytes.Buffer
	if err := svg.Encode(&buf, []*cell.Buffer{b}, svg.Options{Padding: 0, FontSize: 10, Quantize: 0}); err != nil {
		t.Fatal(err)
	}
	out := buf.String()
	// The two neighbouring blocks merge into one 12-wide rectangle.
	if !strings.Contains(out, `<rect x="0" y="0" width="12" height="12" fill="#ff0000"`) {
		t.Errorf("adjacent blocks did not merge into one rect:\n%s", out)
	}
	if !strings.Contains(out, `>x</tspan>`) {
		t.Errorf("the text glyph is missing:\n%s", out)
	}
	if strings.Contains(out, "█") {
		t.Error("a block glyph was also emitted as text, so it is drawn twice")
	}
}

func TestQuantizeMergesNeighbours(t *testing.T) {
	b := cell.New(2, 1)
	b.SetRune(0, 0, '█', tint.RGB(100, 100, 100))
	b.SetRune(1, 0, '█', tint.RGB(103, 100, 100))
	var loose, exact bytes.Buffer
	if err := svg.Encode(&loose, []*cell.Buffer{b}, svg.Options{Quantize: 16}); err != nil {
		t.Fatal(err)
	}
	if err := svg.Encode(&exact, []*cell.Buffer{b}, svg.Options{Quantize: 0}); err != nil {
		t.Fatal(err)
	}
	if got := strings.Count(loose.String(), "<rect"); got != 2 {
		// One background rect plus one merged block rect.
		t.Errorf("quantised output has %d rects, want 2", got)
	}
	if got := strings.Count(exact.String(), "<rect"); got != 3 {
		t.Errorf("exact output has %d rects, want 3", got)
	}
}

func TestFramesMustAgreeOnSize(t *testing.T) {
	err := svg.Encode(io.Discard, []*cell.Buffer{cell.New(4, 2), cell.New(5, 2)}, svg.Options{})
	if err == nil {
		t.Fatal("frames of different sizes should be an error, not a clipped document")
	}
}

func TestNoFrames(t *testing.T) {
	if err := svg.Encode(io.Discard, nil, svg.Options{}); err != svg.ErrNoFrames {
		t.Fatalf("got %v, want ErrNoFrames", err)
	}
}

// Markup in a frame must not become markup in the document.
func TestTextIsEscaped(t *testing.T) {
	b := cell.New(5, 1)
	b.WriteString(0, 0, "a<b&c", tint.None)
	var buf bytes.Buffer
	if err := svg.Encode(&buf, []*cell.Buffer{b}, svg.Options{}); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(buf.String(), "a&lt;b&amp;c") {
		t.Errorf("unescaped text in:\n%s", buf.String())
	}
}

func itoa(i int) string {
	if i == 0 {
		return "0"
	}
	var d []byte
	for i > 0 {
		d = append([]byte{byte('0' + i%10)}, d...)
		i /= 10
	}
	return string(d)
}
