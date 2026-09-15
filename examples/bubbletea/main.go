// Command bubbletea shows asciifx inside a Bubble Tea v2 program: a banner
// reveal plays as an intro, then a spinner with a label runs until you press
// q.
//
//	go run ./examples/bubbletea
package main

import (
	"fmt"
	"os"

	tea "charm.land/bubbletea/v2"

	_ "github.com/cyperx84/ascii-animations/asciifx/effects"
	"github.com/cyperx84/ascii-animations/asciifx/fx"
	"github.com/cyperx84/ascii-animations/asciifx/spinner"
	"github.com/cyperx84/ascii-animations/asciifx/teafx"
	"github.com/cyperx84/ascii-animations/asciifx/tint"
)

type model struct {
	intro   teafx.Model
	spinner spinner.Model
	// spinning flips once the intro is done; the spinner starts ticking
	// then, so it does no work while hidden.
	spinning bool
}

func newModel() (model, error) {
	art, err := fx.Banner("ASCIIFX", "block")
	if err != nil {
		return model{}, err
	}
	intro, err := teafx.New("reveal", fx.Options{
		W: 44, H: 7, Seed: 1,
		Content: fx.Text(art, tint.None),
		Params:  map[string]string{"palette": "synthwave", "pattern": "center"},
	})
	if err != nil {
		return model{}, err
	}
	// The spinner package is a drop-in for charm.land/bubbles/v2/spinner, so
	// this is the same call shape that package's users already write.
	spin := spinner.New(
		spinner.WithSpinner(spinner.Dots),
		spinner.WithLabel("Warming up the pixels"),
		spinner.WithPalette("synthwave"),
	)
	return model{intro: intro, spinner: spin}, nil
}

func (m model) Init() tea.Cmd { return m.intro.Init() }

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if k, ok := msg.(tea.KeyPressMsg); ok {
		switch k.String() {
		case "q", "ctrl+c", "esc":
			return m, tea.Quit
		}
	}
	var cmds []tea.Cmd
	var cmd tea.Cmd
	m.intro, cmd = m.intro.Update(msg)
	cmds = append(cmds, cmd)
	if m.intro.Done() && !m.spinning {
		m.spinning = true
		cmds = append(cmds, m.spinner.Tick)
	}
	m.spinner, cmd = m.spinner.Update(msg)
	cmds = append(cmds, cmd)
	return m, tea.Batch(cmds...)
}

func (m model) View() tea.View {
	s := m.intro.View() + "\n\n"
	if m.spinning {
		s += "  " + m.spinner.View() + "\n\n  press q to quit"
	}
	return tea.NewView(s)
}

func main() {
	m, err := newModel()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if _, err := tea.NewProgram(m).Run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
