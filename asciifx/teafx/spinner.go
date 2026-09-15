package teafx

import (
	"fmt"
	"sort"

	tea "charm.land/bubbletea/v2"

	"github.com/cyperx84/ascii-animations/asciifx/cell"
	"github.com/cyperx84/ascii-animations/asciifx/fx"
	"github.com/cyperx84/ascii-animations/asciifx/term"
)

// Spinner is a one-line activity indicator shaped like bubbles/spinner, so it
// replaces it without changing the structure of a program:
//
//	// before
//	sp := spinner.New()
//	sp.Spinner = spinner.Dot
//	cmd := sp.Tick
//	m.sp, cmd = m.sp.Update(msg)
//	out := m.sp.View()
//
//	// after
//	sp, err := teafx.NewSpinner("dots2", teafx.WithLabel("Compiling"))
//	cmd := sp.Tick
//	m.sp, cmd = m.sp.Update(msg)
//	out := m.sp.View()
//
// The differences are that NewSpinner takes a style name and can fail on a
// typo, and that View emits no padding: the string is exactly the glyph plus
// the label, so a layout can measure it with Width. With no label it is one
// column, as bubbles/spinner's is.
//
// It is backed by the spinner effect, so the frames, colours and shimmer are
// the ones `asciifx render spinner` and `asciifx play spinner` produce.
type Spinner struct {
	m     Model
	style string
	cfg   spinnerConfig
}

// SpinnerOption configures a Spinner.
type SpinnerOption func(*spinnerConfig)

type spinnerConfig struct {
	label   string
	palette string
	speed   float64
	shimmer float64
	fps     int
}

// WithLabel sets the text after the glyph. The default is no label, which
// matches bubbles/spinner and keeps a library from injecting words into a
// layout uninvited; the CLI's `asciifx play spinner` defaults to "Loading"
// because there a bare spinner is the whole program.
func WithLabel(text string) SpinnerOption {
	return func(c *spinnerConfig) { c.label = text }
}

// WithPalette sets the gradient the glyph cycles through and the label
// shimmers with. Accepts a palette name or comma-separated hex stops.
func WithPalette(name string) SpinnerOption {
	return func(c *spinnerConfig) { c.palette = name }
}

// WithSpeed multiplies the glyph's frame rate. 1 is the style's own pace.
func WithSpeed(mult float64) SpinnerOption {
	return func(c *spinnerConfig) { c.speed = mult }
}

// WithShimmer sets how many seconds the label highlight takes to cross, or 0 to
// turn the highlight off. With no shimmer a tick only changes the glyph, so
// WithFPS can drop much lower.
func WithShimmer(seconds float64) SpinnerOption {
	return func(c *spinnerConfig) { c.shimmer = seconds }
}

// WithFPS sets the tick rate. The default is the effect's own 30, which keeps
// the label shimmer smooth. A TUI re-renders on every tick of the whole view,
// so a glyph-only spinner can pass WithShimmer(0), WithFPS(12) and do a third
// of the work.
func WithFPS(fps int) SpinnerOption {
	return func(c *spinnerConfig) { c.fps = fps }
}

// NewSpinner builds a spinner in the named style. SpinnerStyles lists the
// names, which are the ones `asciifx info spinner` reports.
//
// An unknown style is an error rather than a silent fallback: a typo that spins
// the wrong glyphs for a week is worse than a startup error that names the
// valid ones.
func NewSpinner(style string, opts ...SpinnerOption) (Spinner, error) {
	s := Spinner{style: style}
	for _, o := range opts {
		o(&s.cfg)
	}
	if err := s.build(); err != nil {
		return Spinner{}, err
	}
	return s, nil
}

// SpinnerStyles lists the available style names, sorted.
func SpinnerStyles() []string {
	spec, err := fx.Lookup("spinner")
	if err != nil {
		return nil
	}
	for i := range spec.Params {
		if spec.Params[i].Name == "style" {
			out := append([]string(nil), spec.Params[i].Options...)
			sort.Strings(out)
			return out
		}
	}
	return nil
}

// build constructs the run from s.style and s.cfg, leaving it on the frame it
// was already showing.
//
// A spinner's output depends only on its tick, so rebuilding and seeking is
// exact: changing a label mid-spin never restarts the animation. The instance
// id is preserved across a rebuild so tick commands already in flight still
// match, which is what stops a SetLabel from freezing the spinner.
func (s *Spinner) build() error {
	spec, err := fx.Lookup("spinner")
	if err != nil {
		return err
	}
	params := map[string]string{"style": s.style, "label": s.cfg.label}
	if s.cfg.palette != "" {
		params["palette"] = s.cfg.palette
	}
	if s.cfg.speed > 0 {
		params["speed"] = fmt.Sprintf("%g", s.cfg.speed)
	}
	if s.cfg.shimmer > 0 {
		params["shimmer"] = fmt.Sprintf("%g", s.cfg.shimmer)
	}
	fps := s.cfg.fps
	if fps <= 0 {
		fps = spec.FPS
	}
	// Wide enough for the glyph, its gap and the label, so a long label is
	// never clipped by the buffer. View trims whatever is left over.
	w := 4
	if n := len([]rune(s.cfg.label)); n > 0 {
		w = max(n+2, 4)
	}
	run, err := fx.NewRun(spec, fx.Options{W: w, H: 1, Seed: 1, FPS: fps, Params: params})
	if err != nil {
		return err
	}
	tick := 0
	if s.m.run != nil {
		tick = max(s.m.run.Tick(), 0)
	}
	buf, err := run.Seek(tick)
	if err != nil {
		return err
	}
	id, gen := s.m.id, s.m.gen
	if id == 0 {
		// A distinct id per instance, so several spinners in one program do
		// not answer each other's ticks.
		id = lastID.Add(1)
	}
	s.m = Model{id: id, gen: gen, run: run, buf: buf}
	s.cfg.fps = fps
	return nil
}

// Tick returns the command that advances the spinner. Return it from Init and
// from every Update, exactly as bubbles/spinner's Tick is used.
func (s Spinner) Tick() tea.Cmd { return s.m.Init() }

// Update advances on this spinner's own tick message and ignores everything
// else, so it is safe to forward every message of a parent model.
func (s Spinner) Update(msg tea.Msg) (Spinner, tea.Cmd) {
	m, cmd := s.m.Update(msg)
	s.m = m
	return s, cmd
}

// View returns the current frame as styled text, with no trailing padding.
//
// Trailing blanks are dropped by measuring the frame and re-encoding the
// inked part, never by trimming the encoded string, because slicing encoded
// text can cut an escape sequence in half.
func (s Spinner) View() string {
	if s.m.buf == nil {
		return ""
	}
	trimmed := trimTrailingBlanks(s.m.buf)
	if trimmed.W == 0 {
		return ""
	}
	return term.ANSI(trimmed, term.TrueColor)
}

// Width is the number of columns View occupies.
func (s Spinner) Width() int {
	if s.m.buf == nil {
		return 0
	}
	return trimTrailingBlanks(s.m.buf).W
}

// SetLabel replaces the label text without restarting the animation, e.g. to
// report progress: "Compiling (3/12)".
func (s Spinner) SetLabel(text string) Spinner {
	next := s
	next.cfg.label = text
	if err := next.build(); err != nil {
		return s
	}
	return next
}

// SetStyle switches the frame set without restarting the animation. An unknown
// style leaves the spinner unchanged.
func (s Spinner) SetStyle(style string) Spinner {
	next := s
	next.style = style
	if err := next.build(); err != nil {
		return s
	}
	return next
}

// Style reports the current style name.
func (s Spinner) Style() string { return s.style }

// ID identifies this spinner's tick messages. A Bubble Tea program never needs
// it, because Tick and Update carry it; a caller driving the frames itself — a
// test, or an animation loop outside Bubble Tea — uses it to build the message
// Update expects.
func (s Spinner) ID() int64 { return s.m.id }

// trimTrailingBlanks returns the smallest buffer holding every inked column of
// b. Rows keep their positions; only whole trailing columns are dropped, so a
// one-line spinner measures exactly its content and a layout receives no
// padding it did not ask for.
func trimTrailingBlanks(b *cell.Buffer) *cell.Buffer {
	last := -1
	for y := 0; y < b.H; y++ {
		for x := b.W - 1; x > last; x-- {
			if c := b.Cells[y*b.W+x]; c.Rune != 0 && c.Rune != ' ' {
				last = x
				break
			}
		}
	}
	out := cell.New(last+1, b.H)
	for y := 0; y < b.H; y++ {
		for x := 0; x <= last; x++ {
			out.Cells[y*out.W+x] = b.Cells[y*b.W+x]
		}
	}
	return out
}
