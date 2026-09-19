// Command 01-spinner is the smallest way in: replace bubbles/spinner with
// this one and change nothing else.
//
//	go run ./examples/01-spinner
//
// The first spinner below is written exactly as upstream's own example writes
// it — same New, same WithSpinner, same Tick, same TickMsg. The import line is
// the only difference, which is the claim asciifx/spinner makes and the claim
// its compat_test.go checks against the compiler.
//
// The other two are what you get for staying: a label the spin colours, and a
// shimmer that walks a highlight along that label.
package main

import (
	"fmt"
	"os"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/cyperx84/ascii-animations/asciifx/spinner"
)

var (
	titleStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#bd93f9"))
	noteStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("#6272a4"))
	plainStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#8be9fd"))
)

type model struct {
	// upstream is the call shape a bubbles/spinner user already has.
	upstream spinner.Model
	// labelled and shimmering are the two opt-in extras.
	labelled   spinner.Model
	shimmering spinner.Model
}

func newModel() model {
	return model{
		upstream:   spinner.New(spinner.WithSpinner(spinner.MiniDot), spinner.WithStyle(plainStyle)),
		labelled:   spinner.New(spinner.WithSpinner(spinner.Dots2), spinner.WithLabel("resolving hosts"), spinner.WithPalette("nord")),
		shimmering: spinner.New(spinner.WithSpinner(spinner.Dots), spinner.WithLabel("uploading layers"), spinner.WithPalette("synthwave"), spinner.WithShimmer(1.4)),
	}
}

func (m model) Init() tea.Cmd {
	// Tick is a method value, so passing it directly is the upstream idiom.
	return tea.Batch(m.upstream.Tick, m.labelled.Tick, m.shimmering.Tick)
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if k, ok := msg.(tea.KeyPressMsg); ok {
		switch k.String() {
		case "q", "ctrl+c", "esc":
			return m, tea.Quit
		}
	}
	// Each model filters the tick messages that are not its own, so three
	// spinners in one program do not speed each other up.
	var cmds []tea.Cmd
	var cmd tea.Cmd
	m.upstream, cmd = m.upstream.Update(msg)
	cmds = append(cmds, cmd)
	m.labelled, cmd = m.labelled.Update(msg)
	cmds = append(cmds, cmd)
	m.shimmering, cmd = m.shimmering.Update(msg)
	cmds = append(cmds, cmd)
	return m, tea.Batch(cmds...)
}

func (m model) View() tea.View {
	return tea.NewView(m.render())
}

// render is View's body, split out so a test can call it without a terminal.
func (m model) render() string {
	rows := []string{
		titleStyle.Render("asciifx/spinner"),
		"",
		"  " + m.upstream.View() + " " + plainStyle.Render("working") + noteStyle.Render("   — unchanged bubbles/spinner code"),
		"  " + m.labelled.View() + noteStyle.Render("   — WithLabel + WithPalette"),
		"  " + m.shimmering.View() + noteStyle.Render("   — and WithShimmer"),
		"",
		noteStyle.Render("  q to quit"),
	}
	return lipgloss.JoinVertical(lipgloss.Left, rows...) + "\n"
}

func main() {
	if _, err := tea.NewProgram(newModel()).Run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
