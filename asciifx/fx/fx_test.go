package fx

import (
	"math/rand/v2"
	"strconv"
	"strings"
	"testing"

	"github.com/cyperx84/ascii-animations/asciifx/cell"
)

func TestPatternsInRange(t *testing.T) {
	for _, name := range PatternNames() {
		p, err := ParsePattern(name, 3)
		if err != nil {
			t.Fatal(err)
		}
		for _, size := range [][2]int{{1, 1}, {2, 1}, {17, 5}} {
			for y := 0; y < size[1]; y++ {
				for x := 0; x < size[0]; x++ {
					if v := p.Order(x, y, size[0], size[1]); v < 0 || v > 1 {
						t.Fatalf("%s(%d,%d in %v)=%v", name, x, y, size, v)
					}
				}
			}
		}
	}
}

func TestLocalCoversWholeRange(t *testing.T) {
	for _, soft := range []float64{0.01, 0.3, 1} {
		if Local(0, 0, soft) != 0 {
			t.Errorf("soft=%v: first cell visible at p=0", soft)
		}
		if Local(1, 1, soft) != 1 {
			t.Errorf("soft=%v: last cell incomplete at p=1", soft)
		}
	}
}

// counter is a finite test effect that writes its tick into the buffer.
type counter struct{ d float64 }

func (c counter) Duration() float64 { return c.d }
func (c counter) Step(f *Frame) {
	f.Buf.WriteString(0, 0, strings.Repeat("#", min(f.Tick, f.Buf.W)), f.Buf.At(0, 0).FG)
}

func testSpec(name string) *Spec {
	return &Spec{
		Name: name, Kind: Transition, Content: true, FPS: 10, DefW: 8, DefH: 1,
		Params: []Param{FloatParam("duration", 0.5, 0.1, 5, "")},
		New: func(p Values, w, h int, _ *rand.Rand) (Effect, error) {
			return counter{p.Float("duration")}, nil
		},
	}
}

func TestRunFramesAndSeek(t *testing.T) {
	r, err := NewRun(testSpec("counter"), Options{})
	if err != nil {
		t.Fatal(err)
	}
	if r.Frames() != 6 {
		t.Fatalf("0.5s at 10fps should be 6 frames, got %d", r.Frames())
	}
	b, _ := r.Seek(3)
	if got := b.Lines()[0]; got != "###     " {
		t.Fatalf("tick 3 = %q", got)
	}
	b, _ = r.Seek(1) // seeking backwards replays
	if got := b.Lines()[0]; got != "#       " {
		t.Fatalf("tick 1 after rewind = %q", got)
	}
	r.Seek(5)
	if !r.Done() {
		t.Fatal("run not done at last frame")
	}
}

func TestResolveRejectsUnknownAndInvalid(t *testing.T) {
	s := testSpec("x")
	if _, err := s.Resolve(map[string]string{"durration": "1"}); err == nil || !strings.Contains(err.Error(), "duration") {
		t.Fatalf("typo should list valid params, got %v", err)
	}
	if _, err := s.Resolve(map[string]string{"duration": "99"}); err == nil {
		t.Fatal("out-of-range value accepted")
	}
}

func TestSequenceAndHold(t *testing.T) {
	seq := Sequence(counter{0.2}, Hold(counter{0.2}, 0.3))
	if d := seq.Duration(); d < 0.69 || d > 0.71 {
		t.Fatalf("duration %v", d)
	}
	f := &Frame{Buf: cell.New(8, 1), Dt: 0.1}
	f.Tick = 6 // 0.6s: inside the second child's hold, clamped to its tick 2
	seq.Step(f)
	if got := f.Buf.Lines()[0]; got != "##      " {
		t.Fatalf("held frame = %q", got)
	}
}

func TestTextCentresAndSanitises(t *testing.T) {
	b := cell.New(10, 3)
	Text("hi😀", cell.Blank.FG)(b)
	if got := b.Lines()[1]; got != "   hi?    " {
		t.Fatalf("got %q", got)
	}
}

func TestResizeRejectsTooSmallAndKeepsState(t *testing.T) {
	s := testSpec("resize")
	s.MinW, s.MinH = 4, 1
	r, err := NewRun(s, Options{W: 8, H: 1})
	if err != nil {
		t.Fatal(err)
	}
	r.Seek(2)
	if err := r.Resize(2, 1); err == nil {
		t.Fatal("resize below MinW accepted")
	}
	if w, h := r.Size(); w != 8 || h != 1 || r.Tick() != 2 {
		t.Fatalf("failed resize changed state: %dx%d tick %d", w, h, r.Tick())
	}
	if got := r.Next().Lines()[0]; got != "###     " {
		t.Fatalf("run unusable after failed resize: %q", got)
	}
	if err := r.Resize(6, 1); err != nil || r.Tick() != 3 || r.Current().W != 6 {
		t.Fatalf("valid resize: err=%v tick=%d w=%d", err, r.Tick(), r.Current().W)
	}
}

func TestResizeKeepsStateWhenConstructorFails(t *testing.T) {
	s := testSpec("fragile")
	orig := s.New
	s.New = func(p Values, w, h int, rng *rand.Rand) (Effect, error) {
		if w > 10 {
			return nil, errTooWide
		}
		return orig(p, w, h, rng)
	}
	r, _ := NewRun(s, Options{W: 8, H: 1})
	r.Seek(1)
	if err := r.Resize(20, 1); err == nil {
		t.Fatal("constructor error swallowed")
	}
	if w, _ := r.Size(); w != 8 || r.Current().W != 8 || r.Tick() != 1 {
		t.Fatalf("state changed after failed build: w=%d buf=%d tick=%d", w, r.Current().W, r.Tick())
	}
}

var errTooWide = &sizeErr{}

type sizeErr struct{}

func (*sizeErr) Error() string { return "too wide" }

func TestNumericValidationEdgeCases(t *testing.T) {
	s := testSpec("num")
	for _, bad := range []string{"NaN", "nan", "Inf", "-Inf", "+Inf", "1e400"} {
		if _, err := s.Resolve(map[string]string{"duration": bad}); err == nil {
			t.Errorf("%s accepted", bad)
		}
	}
	lo := 1.0
	minOnly := Param{Name: "n", Type: Float, Default: "2", Min: &lo}
	hi := 5.0
	maxOnly := Param{Name: "m", Type: Int, Default: "2", Max: &hi}
	one := &Spec{Name: "bounds", Params: []Param{minOnly, maxOnly}}
	if _, err := one.Resolve(map[string]string{"n": "0"}); err == nil || !strings.Contains(err.Error(), "minimum") {
		t.Errorf("min-only bound: %v", err)
	}
	if _, err := one.Resolve(map[string]string{"m": "9"}); err == nil || !strings.Contains(err.Error(), "maximum") {
		t.Errorf("max-only bound: %v", err)
	}
	if _, err := one.Resolve(map[string]string{"n": "1e9", "m": "-3"}); err != nil {
		t.Errorf("unbounded side rejected: %v", err)
	}
	if _, err := one.Resolve(map[string]string{"m": "2.5"}); err == nil {
		t.Error("fractional int accepted")
	}
}

func TestFramesReachEndForFractionalDurations(t *testing.T) {
	for _, c := range []struct {
		dur float64
		fps int
	}{{1.6, 30}, {0.15, 20}, {0.33, 30}, {2.4, 24}, {0.5, 10}} {
		s := testSpec("frac")
		s.FPS = c.fps
		r, err := NewRun(s, Options{Params: map[string]string{"duration": strconv.FormatFloat(c.dur, 'g', -1, 64)}})
		if err != nil {
			t.Fatal(err)
		}
		last := float64(r.Frames()-1) / float64(c.fps)
		if last < c.dur-1e-9 || last >= c.dur+1/float64(c.fps) {
			t.Errorf("%gs@%dfps: %d frames, last tick at %gs", c.dur, c.fps, r.Frames(), last)
		}
	}
}
