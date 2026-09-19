// Command 04-panels places effects as widgets in a layout, both ways in.
//
//	go run ./examples/04-panels
//
// An effect is a widget, not a takeover. There are two doors, and pressing p
// switches between them so you can see they paint the same pixels:
//
//   - Strings. Model.View returns a styled string, so it drops into a
//     lipgloss Layer like any other component. This is the path to use in a
//     TUI: no new cell type, no new renderer, positioning as usual.
//   - Drawables. Model implements uv.Drawable, so teafx.At pins it to a
//     rectangle on a lipgloss Canvas and it composes cell by cell. Use it
//     inside your own Drawable widget, or on a canvas of drawables.
//
// The rule when mixing them on one Canvas: Canvas.Compose hands every
// drawable the whole canvas, and a Layer draws its content at the area it is
// handed — its own X and Y are read by Compositor, not by Compose. Compose a
// pile of positioned Layers onto a Canvas and they all land on top of each
// other at the origin. teafx.At is the fix, and it pins any drawable, not
// just an effect: this program puts its chrome through it too.
package main

import (
	"fmt"
	"os"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	uv "github.com/charmbracelet/ultraviolet"

	_ "github.com/cyperx84/ascii-animations/asciifx/effects"
	"github.com/cyperx84/ascii-animations/asciifx/fx"
	"github.com/cyperx84/ascii-animations/asciifx/spinner"
	"github.com/cyperx84/ascii-animations/asciifx/teafx"
	"github.com/cyperx84/ascii-animations/asciifx/term"
)

const (
	panelW  = 28
	panelH  = 8
	gap     = 2
	canvasW = panelW*2 + gap
	canvasH = panelH + 5
)

var (
	titleStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#bd93f9"))
	labelStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#8be9fd"))
	noteStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("#6272a4"))
)

// panel is one effect and the label above it.
type panel struct {
	label string
	model teafx.Model
	x, y  int
}

type model struct {
	panels  []panel
	spinner spinner.Model
	// drawables picks the Canvas path over the string path.
	drawables bool
}

func newModel(caps term.Caps) (model, error) {
	left, err := teafx.New("aurora", fx.Options{W: panelW, H: panelH, Seed: 1})
	if err != nil {
		return model{}, err
	}
	right, err := teafx.New("pipes", fx.Options{W: panelW, H: panelH, Seed: 4})
	if err != nil {
		return model{}, err
	}
	left.SetCaps(caps)
	right.SetCaps(caps)
	return model{
		panels: []panel{
			{label: "aurora", model: left, x: 0, y: 2},
			{label: "pipes", model: right, x: panelW + gap, y: 2},
		},
		spinner: spinner.New(spinner.WithSpinner(spinner.Dots2), spinner.WithLabel("compositing"),
			spinner.WithPalette("nord"), spinner.WithCaps(caps)),
	}, nil
}

func (m model) Init() tea.Cmd {
	cmds := []tea.Cmd{m.spinner.Tick}
	for _, p := range m.panels {
		cmds = append(cmds, p.model.Init())
	}
	return tea.Batch(cmds...)
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if k, ok := msg.(tea.KeyPressMsg); ok {
		switch k.String() {
		case "q", "ctrl+c", "esc":
			return m, tea.Quit
		case "p":
			m.drawables = !m.drawables
			return m, nil
		}
	}
	var cmds []tea.Cmd
	var cmd tea.Cmd
	// A slice of models needs a fresh slice: Update returns a value, and the
	// panels are copied into this model by value too.
	panels := make([]panel, len(m.panels))
	copy(panels, m.panels)
	for i := range panels {
		panels[i].model, cmd = panels[i].model.Update(msg)
		cmds = append(cmds, cmd)
	}
	m.panels = panels
	m.spinner, cmd = m.spinner.Update(msg)
	cmds = append(cmds, cmd)
	return m, tea.Batch(cmds...)
}

func (m model) View() tea.View { return tea.NewView(m.render()) }

func (m model) render() string {
	if m.drawables {
		return m.canvasPath()
	}
	return m.stringPath()
}

// placed is one piece of chrome and where it goes. The position is kept
// beside the layer rather than on it, because the two paths read it from
// different places: Compositor from the layer, Compose from a rectangle.
type placed struct {
	layer *lipgloss.Layer
	x, y  int
}

// chrome is the static text of the layout. Both paths take it, which is what
// makes the comparison honest.
func (m model) chrome() []placed {
	path := "strings: Model.View in a Layer"
	if m.drawables {
		path = "drawables: Canvas.Compose(teafx.At(…))"
	}
	out := []placed{
		{lipgloss.NewLayer(titleStyle.Render("asciifx as widgets")), 0, 0},
		{lipgloss.NewLayer(noteStyle.Render(path)), 0, panelH + 2},
		{lipgloss.NewLayer("  " + m.spinner.View()), 0, panelH + 3},
		{lipgloss.NewLayer(noteStyle.Render("p switch path · q quit")), 0, panelH + 4},
	}
	for _, p := range m.panels {
		out = append(out, placed{lipgloss.NewLayer(labelStyle.Render(p.label)), p.x, p.y - 1})
	}
	return out
}

// stringPath composes styled strings, which is what most TUIs want.
func (m model) stringPath() string {
	var layers []*lipgloss.Layer
	for _, c := range m.chrome() {
		layers = append(layers, c.layer.X(c.x).Y(c.y))
	}
	for _, p := range m.panels {
		layers = append(layers, lipgloss.NewLayer(p.model.View()).X(p.x).Y(p.y))
	}
	return lipgloss.NewCompositor(layers...).Render()
}

// canvasPath composes cell by cell. Every drawable is pinned, because
// Canvas.Compose hands each one the whole canvas: an unpinned Layer would
// paint at the origin no matter what X and Y it was given, and each one
// composed after it would paint over the last.
func (m model) canvasPath() string {
	c := lipgloss.NewCanvas(canvasW, canvasH)
	for _, ch := range m.chrome() {
		c.Compose(teafx.At(ch.layer, uv.Rect(ch.x, ch.y, ch.layer.Width(), ch.layer.Height())))
	}
	for _, p := range m.panels {
		c.Compose(teafx.At(p.model, uv.Rect(p.x, p.y, panelW, panelH)))
	}
	return c.Render()
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
