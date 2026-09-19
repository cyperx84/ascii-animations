// Command 03-chain builds a three-part intro out of one effect each and
// plays it as a single Bubble Tea component.
//
//	go run ./examples/03-chain
//
// The chain is the CLI's `--then` in Go:
//
//	asciifx render reveal --banner FORGE --then fire --for 1.2s --filter 'not(ink)' --then shine
//
// Two pieces make it work. fx.Compose turns the steps into one Spec that
// finishes, and Step.Filter restricts what a step may write: fire computes
// over the whole buffer as usual and is then clipped to the cells the banner
// does not occupy, so the flames burn around the letters instead of through
// them. The filter belongs to its own step — a chain never propagates one.
//
// teafx.FromRun is what gets it on screen. A composition is a Spec that was
// never registered, so there is no name for teafx.New to look up.
package main

import (
	"fmt"
	"os"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	_ "github.com/cyperx84/ascii-animations/asciifx/effects"
	"github.com/cyperx84/ascii-animations/asciifx/fx"
	"github.com/cyperx84/ascii-animations/asciifx/teafx"
	"github.com/cyperx84/ascii-animations/asciifx/term"
	"github.com/cyperx84/ascii-animations/asciifx/tint"
)

const (
	bannerText = "FORGE"
	width      = 52
	height     = 9
)

var (
	noteStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("#6272a4"))
	stepStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("#ffb86c"))
	quietStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#44475a"))
)

type model struct {
	chain teafx.Model
	steps []string
}

func newModel(caps term.Caps) (model, error) {
	art, err := fx.Banner(bannerText, "block")
	if err != nil {
		return model{}, err
	}
	// not(ink) selects every cell the banner does not draw into. The
	// selector is asked about the cell *before* the step runs, so it
	// composes with the effect rather than fighting it. Built here from the
	// typed constructors; fx.ParseSelector("not(ink)") is the same thing
	// from a string, which is what the CLI does with --filter.
	spec, err := fx.Compose(
		fx.Step{Name: "reveal", Params: map[string]string{
			"pattern": "center", "palette": "fire", "easing": "out-cubic",
		}},
		fx.Step{Name: "fire", For: 1.4, Filter: fx.SelNot(fx.SelInk)},
		fx.Step{Name: "shine", For: 0.9, Params: map[string]string{"palette": "sunset"}},
	)
	if err != nil {
		return model{}, err
	}
	run, err := fx.NewRun(spec, fx.Options{
		W: width, H: height, Seed: 7,
		Content: fx.Text(art, tint.MustHex("#ffd7a1")),
	})
	if err != nil {
		return model{}, err
	}
	chain, err := teafx.FromRun(run)
	if err != nil {
		return model{}, err
	}
	chain.Loop = true
	chain.SetCaps(caps)
	return model{
		chain: chain,
		steps: []string{"reveal", "fire ∧ not(ink)", "shine"},
	}, nil
}

func (m model) Init() tea.Cmd { return m.chain.Init() }

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if k, ok := msg.(tea.KeyPressMsg); ok {
		switch k.String() {
		case "q", "ctrl+c", "esc":
			return m, tea.Quit
		case "r":
			return m, m.chain.Restart()
		case "l":
			m.chain.Loop = !m.chain.Loop
			if m.chain.Loop {
				return m, m.chain.Restart()
			}
			return m, nil
		}
	}
	var cmd tea.Cmd
	m.chain, cmd = m.chain.Update(msg)
	return m, cmd
}

func (m model) View() tea.View { return tea.NewView(m.render()) }

func (m model) render() string {
	// Which step is on screen, derived from the run's own tick. A
	// composition is one run, so the tick keeps counting across steps.
	run := m.chain.Run()
	at := 0
	if frames := run.Frames(); frames > 0 {
		at = min(len(m.steps)*run.Tick()/max(frames, 1), len(m.steps)-1)
	}
	labels := make([]string, len(m.steps))
	for i, s := range m.steps {
		if i == at {
			labels[i] = stepStyle.Render("▸ " + s)
		} else {
			labels[i] = quietStyle.Render("  " + s)
		}
	}
	loop := "off"
	if m.chain.Loop {
		loop = "on"
	}
	return m.chain.View() + "\n" +
		lipgloss.JoinHorizontal(lipgloss.Top, labels...) + "\n\n" +
		noteStyle.Render(fmt.Sprintf("  r replay · l loop (%s) · q quit", loop)) + "\n"
}

func main() {
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
