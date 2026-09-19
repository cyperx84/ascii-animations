// Command 05-dashboard is the whole surface in one screen: a fake fleet
// dashboard whose panels are asciifx effects.
//
//	go run ./examples/05-dashboard
//
// What it uses, and why it is here rather than in a smaller example:
//
//   - A Canvas of drawables. Every widget is pinned with teafx.At, so the
//     effects composite cell by cell with the chrome instead of being pasted
//     over it.
//   - teafx.Content. Pressing space refreshes the node table, and the new
//     table is animated in by decrypt — the effect is handed the panel that
//     was just drawn, so it transforms the layout instead of replacing it.
//     That is the snapshot-and-transition loop, and it is the reason the
//     numbers resolve out of cipher noise rather than blinking.
//   - Several models at once, each with its own tick chain: an ambient
//     background that loops, a decrypt that runs once per refresh, and a
//     spinner.
//   - SetSize on resize, SetCaps for the terminal contract, and fx.Hash01 for
//     fake data that is the same on every run.
//
// Try it: ASCIIFX_REDUCED_MOTION=1 go run ./examples/05-dashboard
package main

import (
	"fmt"
	"os"
	"strings"
	"time"

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
	defaultW, defaultH = 84, 22
	minW, minH         = 60, 16
	tableW             = 38
	refreshEvery       = 6 * time.Second
)

// ambients are the backgrounds 'a' cycles through.
var ambients = []string{"aurora", "plasma", "matrix", "starfield", "rain"}

var (
	titleStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#bd93f9"))
	labelStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#8be9fd"))
	noteStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("#6272a4"))
	okStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("#50fa7b"))
	warnStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("#ffb86c"))
	downStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("#ff5555"))
)

// refreshMsg drives the automatic poll. gen names the timer chain it belongs
// to: a manual refresh starts a new one, and a message from the old chain is
// dropped rather than starting a third. Without that, pressing space five
// times leaves six timers running and the table polls six times as often as
// refreshEvery says — the same "duplicate chains run at their sum" failure
// teafx.TickMsg's tag exists to prevent.
type refreshMsg struct {
	time time.Time
	gen  uint64
}

type node struct {
	name    string
	state   string
	latency int
	load    int
}

type model struct {
	w, h int
	caps term.Caps

	ambient    teafx.Model
	ambientIdx int
	// reveal animates the node table on refresh. It is nil between
	// refreshes, because a finished transition has nothing left to say.
	reveal  *teafx.Model
	spinner spinner.Model

	nodes []node
	// generation seeds the fake data, so every refresh differs and every run
	// of the program is identical. It doubles as the timer chain's tag.
	generation uint64
}

func newModel(caps term.Caps) (model, error) {
	m := model{w: defaultW, h: defaultH, caps: caps}
	m.spinner = spinner.New(
		spinner.WithSpinner(spinner.Dots2),
		spinner.WithLabel("polling control plane"),
		spinner.WithPalette("nord"),
		spinner.WithCaps(caps),
	)
	if err := m.setAmbient(0); err != nil {
		return model{}, err
	}
	m.nodes = fakeNodes(m.generation)
	return m, nil
}

// setAmbient swaps the background effect, sized to the panel it fills.
func (m *model) setAmbient(idx int) error {
	area := m.ambientRect()
	a, err := teafx.New(ambients[idx], fx.Options{W: area.Dx(), H: area.Dy(), Seed: 11})
	if err != nil {
		return err
	}
	a.Loop = true
	a.SetCaps(m.caps)
	m.ambient, m.ambientIdx = a, idx
	return nil
}

// Geometry. Every rectangle is derived from the window size, so a resize is
// one recompute rather than a pile of offsets.
func (m model) tableRect() uv.Rectangle {
	return uv.Rect(1, 3, tableW, m.h-7)
}

func (m model) ambientRect() uv.Rectangle {
	w := m.w - tableW - 3
	if w < 10 {
		w = 10
	}
	return uv.Rect(tableW+2, 3, w, m.h-7)
}

func (m model) Init() tea.Cmd {
	return tea.Batch(m.ambient.Init(), m.spinner.Tick, m.refreshTick())
}

// refreshTick schedules the next automatic poll, tagged with the generation
// that scheduled it.
func (m model) refreshTick() tea.Cmd {
	gen := m.generation
	return tea.Tick(refreshEvery, func(t time.Time) tea.Msg {
		return refreshMsg{time: t, gen: gen}
	})
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd
	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		switch msg.String() {
		case "q", "ctrl+c", "esc":
			return m, tea.Quit
		// Bubble Tea v2 names this key "space", not " ".
		case "space", "enter":
			// refresh bumps the generation, so the timer in flight is stale
			// and the one scheduled here replaces it.
			return m, tea.Batch(m.refresh(), m.refreshTick())
		case "a":
			if err := m.setAmbient((m.ambientIdx + 1) % len(ambients)); err != nil {
				return m, tea.Quit
			}
			return m, m.ambient.Init()
		}
	case tea.WindowSizeMsg:
		m.w, m.h = max(msg.Width, minW), max(msg.Height, minH)
		// The ambient panel is not Fit, because it owns one region rather
		// than the screen: the parent decides its size.
		area := m.ambientRect()
		m.ambient.SetSize(area.Dx(), area.Dy())
		// A transition mid-flight is sized to the old panel; drop it rather
		// than draw it at the wrong width.
		m.reveal = nil
	case refreshMsg:
		if msg.gen != m.generation {
			// A tick from a chain a manual refresh superseded.
			return m, nil
		}
		return m, tea.Batch(m.refresh(), m.refreshTick())
	}

	var cmd tea.Cmd
	m.ambient, cmd = m.ambient.Update(msg)
	cmds = append(cmds, cmd)
	m.spinner, cmd = m.spinner.Update(msg)
	cmds = append(cmds, cmd)
	if m.reveal != nil {
		r := *m.reveal
		r, cmd = r.Update(msg)
		cmds = append(cmds, cmd)
		if r.Done() {
			m.reveal = nil
		} else {
			m.reveal = &r
		}
	}
	return m, tea.Batch(cmds...)
}

// refresh replaces the fleet data and animates the new table in from the
// cells it will occupy.
func (m *model) refresh() tea.Cmd {
	m.generation++
	m.nodes = fakeNodes(m.generation)
	area := m.tableRect()

	// Draw the new table on a canvas of its own, then hand those cells to
	// the transition. Content snapshots once, so what decrypt resolves to is
	// exactly what the panel will hold when it is done.
	scratch := lipgloss.NewCanvas(area.Dx(), area.Dy())
	scratch.Compose(lipgloss.NewLayer(m.tableString()).X(0).Y(0))

	r, err := teafx.New("decrypt", fx.Options{
		W: area.Dx(), H: area.Dy(), Seed: m.generation,
		Content: teafx.Content(scratch, scratch.Bounds()),
		Params: map[string]string{
			"pattern": "left", "palette": "matrix", "final": "content",
			"duration": "1.1", "scramble": "0.15",
		},
	})
	if err != nil {
		return nil
	}
	r.SetCaps(m.caps)
	m.reveal = &r
	return r.Init()
}

func (m model) View() tea.View {
	v := tea.NewView(m.render())
	// The dashboard owns the window, so it takes the alternate screen: in
	// Bubble Tea v2 that is a property of the view, not a program option.
	v.AltScreen = true
	return v
}

func (m model) render() string {
	c := lipgloss.NewCanvas(m.w, m.h)
	table, ambient := m.tableRect(), m.ambientRect()

	// Everything is pinned, chrome included. Canvas.Compose hands each
	// drawable the whole canvas, and a Layer paints every cell of the area
	// it is handed: compose one unpinned and it clears the canvas and draws
	// at the origin, whatever X and Y it was given. Those are read by
	// Compositor, not by Compose.
	text(c, 1, 0, titleStyle.Render("FLEET"))
	text(c, 8, 0, noteStyle.Render(fmt.Sprintf("poll #%d · %d nodes", m.generation, len(m.nodes))))
	text(c, 1, 2, labelStyle.Render("nodes"))
	text(c, ambient.Min.X, 2, labelStyle.Render(ambients[m.ambientIdx]))
	text(c, 2, m.h-3, m.spinner.View())
	text(c, 2, m.h-2, noteStyle.Render("space refresh · a background · q quit"))

	c.Compose(teafx.At(m.ambient, ambient))
	if m.reveal != nil {
		c.Compose(teafx.At(*m.reveal, table))
	} else {
		text(c, table.Min.X, table.Min.Y, m.tableString())
	}
	return c.Render()
}

// text draws a styled string at one position, which is teafx.At applied to a
// Layer: the pin is what keeps it inside its own rectangle.
func text(c *lipgloss.Canvas, x, y int, s string) {
	l := lipgloss.NewLayer(s)
	c.Compose(teafx.At(l, uv.Rect(x, y, l.Width(), l.Height())))
}

// tableString is the node table as plain, padded lines. Padding matters: the
// snapshot a transition animates is cells, so a short line would hand it
// blanks where the table used to be.
func (m model) tableString() string {
	rows := make([]string, 0, len(m.nodes)+1)
	rows = append(rows, labelStyle.Render(pad(fmt.Sprintf("%-12s %-9s %6s %5s", "NODE", "STATE", "LAT", "LOAD"), tableW)))
	for _, n := range m.nodes {
		state := fmt.Sprintf("%-9s", n.state)
		line := pad(fmt.Sprintf("%-12s %s %5dms %4d%%", n.name, state, n.latency, n.load), tableW)
		// Colour the state column after padding, because the escape codes
		// occupy no columns and would otherwise be counted as width. The
		// colour survives the transition: decrypt's final=content resolves
		// each cell to the colour the snapshot had.
		rows = append(rows, strings.Replace(line, state, stateStyle(n.state).Render(state), 1))
	}
	return strings.Join(rows, "\n")
}

func stateStyle(state string) lipgloss.Style {
	switch state {
	case "down":
		return downStyle
	case "degraded":
		return warnStyle
	}
	return okStyle
}

func pad(s string, w int) string {
	if n := w - len([]rune(s)); n > 0 {
		return s + strings.Repeat(" ", n)
	}
	return string([]rune(s)[:w])
}

// fakeNodes invents a fleet from the generation number. Hash01 is the
// engine's own per-cell noise: stateless, seeded, and identical on every run,
// so the demo records the same way twice.
func fakeNodes(gen uint64) []node {
	names := []string{"atlas-01", "atlas-02", "borg-07", "kiln-11", "kiln-12", "relay-03", "relay-04"}
	out := make([]node, len(names))
	for i, name := range names {
		h := fx.Hash01(i, 0, gen)
		state := "ok"
		switch {
		case h > 0.92:
			state = "down"
		case h > 0.78:
			state = "degraded"
		}
		out[i] = node{
			name:    name,
			state:   state,
			latency: 8 + int(fx.Hash01(i, 1, gen)*320),
			load:    int(fx.Hash01(i, 2, gen) * 100),
		}
	}
	return out
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
