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
type TickMsg struct {
	ID  int64
	gen int64
}

// Model is a Bubble Tea v2 component that plays one effect.
type Model struct {
	// Fit makes the effect resize to every tea.WindowSizeMsg. Leave it off
	// when the effect is one part of a larger layout and call SetSize.
	Fit bool
	// Loop restarts finite effects when they end instead of stopping.
	Loop bool

	id  int64
	gen int64
	run *fx.Run
	buf *cell.Buffer
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

// Init starts ticking.
func (m Model) Init() tea.Cmd { return m.tick() }

func (m Model) tick() tea.Cmd {
	if m.run == nil {
		return nil
	}
	id, gen := m.id, m.gen
	return tea.Tick(time.Second/time.Duration(m.run.FPS()), func(time.Time) tea.Msg {
		return TickMsg{ID: id, gen: gen}
	})
}

// Update advances on this model's own TickMsg and, when Fit is set, resizes
// on tea.WindowSizeMsg. Everything else is ignored.
func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	if m.run == nil {
		return m, nil
	}
	switch msg := msg.(type) {
	case TickMsg:
		if msg.ID != m.id || msg.gen != m.gen {
			return m, nil
		}
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
// here.
func (m Model) View() string {
	if m.buf == nil {
		return ""
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
// ticking. Ticks already in flight from before the restart are discarded.
func (m *Model) Restart() tea.Cmd {
	if m.run == nil {
		return nil
	}
	m.gen++
	m.buf, _ = m.run.Seek(0)
	return m.tick()
}
