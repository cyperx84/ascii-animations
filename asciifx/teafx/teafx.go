// Package teafx embeds asciifx effects in Bubble Tea v2 programs.
//
// A Model is an ordinary Bubble Tea component: return its Init command from
// your Init, forward messages to its Update, and place its View string in
// your layout.
//
//	m, err := teafx.New("reveal", fx.Options{Content: fx.Text("HELLO", tint.None)})
//	...
//	func (p parent) Init() tea.Cmd { return p.intro.Init() }
//	func (p parent) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
//		var cmd tea.Cmd
//		p.intro, cmd = p.intro.Update(msg)
//		return p, cmd
//	}
//
// Frames advance one tick per tick message at the run's FPS, so output is
// exactly what `asciifx render` shows for the same options.
package teafx

import (
	"errors"
	"sync/atomic"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/cyperx84/ascii-animations/asciifx/cell"
	"github.com/cyperx84/ascii-animations/asciifx/fx"
	"github.com/cyperx84/ascii-animations/asciifx/term"
)

var lastID atomic.Int64

// TickMsg advances the Model whose ID matches. Other instances ignore it, so
// several effects can run in one program without cross-talk.
//
// tag names the tick chain the message belongs to. A message carrying a stale
// tag is dropped, which is what stops a second Tick call from starting a second
// chain and doubling the frame rate for good. A zero ID or tag is a wildcard, so
// a message built by hand still works.
type TickMsg struct {
	ID  int64
	tag int
}

// Model is a Bubble Tea v2 component that plays one effect.
type Model struct {
	// Fit makes the effect resize to every tea.WindowSizeMsg. Leave it off
	// when the effect is one part of a larger layout and call SetSize.
	Fit bool
	// Loop restarts finite effects when they end instead of stopping.
	Loop bool

	id  int64
	tag int
	run *fx.Run
	buf *cell.Buffer
	// caps is the terminal contract this model honours, and capsSet says a
	// caller handed one over. They are separate because the zero Caps is
	// deliberately conservative — NoColor and not animated — so "no Caps
	// given" cannot be spelled as an empty one.
	caps    term.Caps
	capsSet bool
}

// New builds a model for a registered effect. opts is validated exactly as
// the CLI validates it; zero W/H use the effect's default size.
func New(effect string, opts fx.Options) (Model, error) {
	spec, err := fx.Lookup(effect)
	if err != nil {
		return Model{}, err
	}
	r, err := fx.NewRun(spec, opts)
	if err != nil {
		return Model{}, err
	}
	return FromRun(r)
}

// FromRun builds a model for a run the caller already has, which is the way
// in for anything Lookup cannot name: a composition from fx.Compose, or a run
// whose options were assembled elsewhere.
//
//	spec, err := fx.Compose(
//		fx.Step{Name: "reveal", Params: map[string]string{"pattern": "center"}},
//		fx.Step{Name: "fire", For: 1.2, Filter: fx.SelNot(fx.SelInk)},
//	)
//	run, err := fx.NewRun(spec, fx.Options{W: 44, H: 9, Content: fx.Text(art, tint.None)})
//	m, err := teafx.FromRun(run)
//
// The run is driven only through the model after this: the model owns the
// tick chain, so stepping the same run by hand as well would show two
// different frames to one component.
func FromRun(r *fx.Run) (Model, error) {
	if r == nil {
		return Model{}, errors.New("teafx: nil run")
	}
	buf, err := r.Seek(0)
	if err != nil {
		return Model{}, err
	}
	return Model{id: lastID.Add(1), run: r, buf: buf}, nil
}

// ID identifies this instance's tick messages.
func (m Model) ID() int64 { return m.id }

// Run exposes the underlying run, e.g. for Tick or Size.
func (m Model) Run() *fx.Run { return m.run }

// SetCaps gives the model the terminal contract term.Play honours, which a
// Bubble Tea program otherwise gets none of: the frame rate cap for slow
// transports, the static frame when motion is unwanted, and the colour
// profile and dither for View.
//
//	m.SetCaps(term.Detect(os.Stdout))
//
// Call it before Init. A Caps whose Animate is false makes the model show one
// frame — the same frame term.Play shows, term.StaticTick's — and Init then
// starts no tick chain, so a program under CI, a pipe, TERM=dumb or
// ASCIIFX_REDUCED_MOTION does no per-frame work at all.
//
// Nothing is honoured unless this is called, because the zero Caps means "no
// colour, do not animate": defaulting to it would freeze every model that
// never asked about the terminal.
func (m *Model) SetCaps(c term.Caps) {
	if m.run == nil {
		return
	}
	m.caps, m.capsSet = c, true
	// A new contract invalidates the ticks already in flight, so a model told
	// to stop animating stops even if it was mid-chain.
	m.tag++
	if !c.Animate {
		m.buf, _ = m.run.Seek(term.StaticTick(m.run))
	}
}

// Init starts ticking.
func (m Model) Init() tea.Cmd { return m.tick() }

// fps is the rate to tick at: the run's own, lowered to the cap when the
// terminal asked for a lower one. A cap above the run's rate is not an
// instruction to speed up.
func (m Model) fps() int {
	f := m.run.FPS()
	if m.capsSet && m.caps.FPS > 0 && m.caps.FPS < f {
		return m.caps.FPS
	}
	return f
}

// tick schedules the next tick of the chain the model is currently on.
func (m Model) tick() tea.Cmd {
	if m.run == nil || m.static() {
		return nil
	}
	id, tag := m.id, m.tag
	return tea.Tick(time.Second/time.Duration(m.fps()), func(time.Time) tea.Msg {
		return TickMsg{ID: id, tag: tag}
	})
}

// static reports whether this model shows one frame instead of animating.
func (m Model) static() bool { return m.capsSet && !m.caps.Animate }

// Update advances on this model's own TickMsg and, when Fit is set, resizes
// on tea.WindowSizeMsg. Everything else is ignored.
func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	if m.run == nil {
		return m, nil
	}
	switch msg := msg.(type) {
	case TickMsg:
		if m.static() {
			return m, nil
		}
		if msg.ID != 0 && msg.ID != m.id {
			return m, nil
		}
		if msg.tag != 0 && msg.tag != m.tag {
			// A tick from a chain that has been superseded. Dropping it is what
			// collapses duplicate chains instead of running at their sum.
			return m, nil
		}
		m.tag++
		if m.run.Done() {
			if !m.Loop {
				return m, nil
			}
			m.buf, _ = m.run.Seek(0)
			return m, m.tick()
		}
		m.buf = m.run.Next()
		if m.run.Done() && !m.Loop {
			return m, nil
		}
		return m, m.tick()
	case tea.WindowSizeMsg:
		if m.Fit {
			m.SetSize(msg.Width, msg.Height)
		}
	}
	return m, nil
}

// View renders the current frame in truecolor. Bubble Tea v2's renderer
// downsamples to the terminal's colour profile, so no detection is needed
// here. After SetCaps the frame is encoded at that Caps' profile and dither
// instead, which is what a caller who overrode the profile — or who wants
// dithering rather than nearest-entry quantisation — is asking for.
func (m Model) View() string {
	if m.buf == nil {
		return ""
	}
	if m.capsSet {
		return term.ANSICaps(m.buf, m.caps)
	}
	return term.ANSI(m.buf, term.TrueColor)
}

// Done reports whether a finite, non-looping effect has shown its last
// frame. Looping and ambient effects are never done.
func (m Model) Done() bool {
	return m.run != nil && !m.Loop && m.run.Done()
}

// SetSize resizes the effect. Sizes below the effect's minimum are ignored.
// Transitions keep their place; ambient simulations restart at the new size.
func (m *Model) SetSize(w, h int) {
	if m.run == nil {
		return
	}
	if cw, ch := m.run.Size(); cw == w && ch == h {
		return
	}
	// Resize leaves the run untouched on error (e.g. below the minimum size).
	if err := m.run.Resize(w, h); err != nil {
		return
	}
	m.buf = m.run.Current()
	if m.run.Tick() < 0 {
		// Ambient runs rebuild from scratch; draw frame 0 so View is not blank.
		m.buf, _ = m.run.Seek(0)
	}
}

// Restart rewinds to the first frame and returns the command that resumes
// ticking. Bumping the tag makes every tick already in flight stale, so the
// old chain dies instead of racing the new one.
func (m *Model) Restart() tea.Cmd {
	if m.run == nil {
		return nil
	}
	m.tag++
	if m.static() {
		// Rewinding a static model would replace its one frame with frame
		// zero, which for a transition is a blank buffer.
		return nil
	}
	m.buf, _ = m.run.Seek(0)
	return m.tick()
}
