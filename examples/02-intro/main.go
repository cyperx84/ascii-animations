// Command 02-intro plays a banner reveal as a startup intro, then hands the
// screen to the app.
//
//	go run ./examples/02-intro
//
// It shows the three things an intro needs and one that is easy to forget:
//
//   - teafx.Model is an ordinary Bubble Tea component. Return its Init from
//     yours, forward messages to its Update, put its View in your layout.
//   - Done reports when a finite effect has shown its last frame, which is
//     the cue to start the real UI.
//   - Restart replays it, and kills the ticks already in flight so the old
//     chain cannot race the new one.
//   - SetCaps hands the model the terminal's own verdict. Under CI, a pipe,
//     TERM=dumb or ASCIIFX_REDUCED_MOTION it shows one static frame and never
//     ticks at all, and over SSH or tmux it drops to the capped frame rate.
//     Nothing else in a Bubble Tea program does this for you.
//
// Try it: ASCIIFX_REDUCED_MOTION=1 go run ./examples/02-intro
package main

import (
	"fmt"
	"os"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	_ "github.com/cyperx84/ascii-animations/asciifx/effects"
	"github.com/cyperx84/ascii-animations/asciifx/fx"
	"github.com/cyperx84/ascii-animations/asciifx/spinner"
	"github.com/cyperx84/ascii-animations/asciifx/teafx"
	"github.com/cyperx84/ascii-animations/asciifx/term"
	"github.com/cyperx84/ascii-animations/asciifx/tint"
)

var (
	noteStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#6272a4"))
	appStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("#8be9fd"))
)

type model struct {
	intro   teafx.Model
	spinner spinner.Model
	// started flips once the intro is done, so the spinner does no work
	// while it is hidden.
	started bool
}

func newModel(caps term.Caps) (model, error) {
	art, err := fx.Banner("ASCIIFX", "block")
	if err != nil {
		return model{}, err
	}
	intro, err := teafx.New("reveal", fx.Options{
		W: 44, H: 7, Seed: 1,
		Content: fx.Text(art, tint.None),
		Params:  map[string]string{"palette": "synthwave", "pattern": "center", "easing": "out-cubic"},
	})
	if err != nil {
		return model{}, err
	}
	// The whole terminal-safety contract, in one call.
	intro.SetCaps(caps)
	spin := spinner.New(
		spinner.WithSpinner(spinner.Dots),
		spinner.WithLabel("Warming up the pixels"),
		spinner.WithPalette("synthwave"),
		// The spinner needs the same verdict the intro got. It is the one
		// widget that animates for as long as the program runs, so it is the
		// one that matters most under reduced motion.
		spinner.WithCaps(caps),
	)
	return model{intro: intro, spinner: spin}, nil
}

func (m model) Init() tea.Cmd { return m.intro.Init() }

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if k, ok := msg.(tea.KeyPressMsg); ok {
		switch k.String() {
		case "q", "ctrl+c", "esc":
			return m, tea.Quit
		case "r":
			// Replay. Restart returns the command that resumes ticking, or
			// nil when the terminal asked for no motion.
			return m, m.intro.Restart()
		}
	}
	var cmds []tea.Cmd
	var cmd tea.Cmd
	m.intro, cmd = m.intro.Update(msg)
	cmds = append(cmds, cmd)
	if m.intro.Done() && !m.started {
		m.started = true
		cmds = append(cmds, m.spinner.Tick)
	}
	m.spinner, cmd = m.spinner.Update(msg)
	cmds = append(cmds, cmd)
	return m, tea.Batch(cmds...)
}

func (m model) View() tea.View { return tea.NewView(m.render()) }

func (m model) render() string {
	s := m.intro.View() + "\n"
	if m.started {
		s += "\n  " + m.spinner.View() + "\n\n" + appStyle.Render("  the app starts here") + "\n"
	}
	return s + noteStyle.Render("\n  r replay · q quit") + "\n"
}

func main() {
	// Detect reads the environment only; it never touches the terminal, so
	// it is safe to call before deciding whether to animate at all.
	m, err := newModel(term.Detect(os.Stdout))
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if _, err := tea.NewProgram(m).Run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
