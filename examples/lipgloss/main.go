// Command lipgloss shows asciifx effects as widgets inside a lipgloss v2
// layout: each effect is placed and clipped like any other component.
//
//	go run ./examples/lipgloss
//
// There are two ways in, and this program uses the first:
//
//   - Model.View() and term.ANSI return styled strings, which drop straight
//     into a lipgloss Layer and Compositor. No new cell type, no new renderer,
//     and positioning works as it does for any other string widget. This is
//     the path to use in a TUI.
//
//   - Model implements uv.Drawable, so teafx.At can pin it to a rectangle on a
//     lipgloss Canvas and it composes cell by cell. Use that inside your own
//     uv.Drawable widget, or on a canvas where every widget is a Drawable.
//     Do not mix it with string Layers on one Canvas: a Layer fills the whole
//     area it is handed, so composing one wipes the drawables beneath it.
package main

import (
	"fmt"
	"os"
	"time"

	"charm.land/lipgloss/v2"

	"github.com/cyperx84/ascii-animations/asciifx/cell"
	_ "github.com/cyperx84/ascii-animations/asciifx/effects"
	"github.com/cyperx84/ascii-animations/asciifx/fx"
	"github.com/cyperx84/ascii-animations/asciifx/teafx"
	"github.com/cyperx84/ascii-animations/asciifx/term"
)

const (
	frames  = 90
	canvasW = 60
	canvasH = 16
)

var (
	titleStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#bd93f9"))
	labelStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#8be9fd"))
	hintStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("#6272a4"))
	// frameStyle pads every frame to the same box, so a shorter frame cannot
	// leave cells from the previous one behind.
	frameStyle = lipgloss.NewStyle().Width(canvasW).Height(canvasH)
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run() error {
	// Ambient effects sized to their own slot, not to the screen.
	fire, err := newRun("fire", 26, 8)
	if err != nil {
		return err
	}
	plasma, err := newRun("plasma", 26, 8)
	if err != nil {
		return err
	}
	// The spinner is a drop-in for bubbles/spinner. Here it is driven by hand,
	// because this program has no Bubble Tea runtime; Tick and Update are the
	// same calls a Bubble Tea program makes.
	spin, err := teafx.NewSpinner("dots2", teafx.WithLabel("rendering"), teafx.WithPalette("nord"))
	if err != nil {
		return err
	}

	fmt.Print("\x1b[?1049h\x1b[?25l")
	defer fmt.Print("\x1b[?1049l\x1b[?25h")

	for tick := 0; tick < frames; tick++ {
		fireBuf, err := fire.Seek(tick)
		if err != nil {
			return err
		}
		plasmaBuf, err := plasma.Seek(tick)
		if err != nil {
			return err
		}
		spin, _ = spin.Update(teafx.TickMsg{ID: spin.ID()})

		// Every layer is positioned in cells and composites with the others.
		frame := lipgloss.NewCompositor(
			lipgloss.NewLayer(titleStyle.Render("asciifx inside lipgloss")).X(1).Y(0),
			lipgloss.NewLayer(labelStyle.Render("fire")).X(1).Y(1),
			lipgloss.NewLayer(frameANSI(fireBuf)).X(1).Y(2),
			lipgloss.NewLayer(labelStyle.Render("plasma")).X(29).Y(1),
			lipgloss.NewLayer(frameANSI(plasmaBuf)).X(29).Y(2),
			lipgloss.NewLayer(labelStyle.Render("  ")+spin.View()).X(1).Y(11),
			lipgloss.NewLayer(hintStyle.Render("two effects, one spinner, one layout")).X(1).Y(13),
		).Render()

		fmt.Print("\x1b[H" + frameStyle.Render(frame))
		time.Sleep(33 * time.Millisecond)
	}
	return nil
}

// A note on the loop above: this example repaints every cell of a fixed box,
// which is simple and correct but writes far more bytes than it needs to. A
// real program should diff instead, and there are two easy ways: put the
// effect in a Bubble Tea View and let the framework's renderer diff it, or own
// the loop and use term.Renderer, which encodes only the cells that changed.

func newRun(effect string, w, h int) (*fx.Run, error) {
	spec, err := fx.Lookup(effect)
	if err != nil {
		return nil, err
	}
	return fx.NewRun(spec, fx.Options{W: w, H: h, Seed: 1})
}

// frameANSI renders a frame the way any other widget renders: a styled string.
// term.ANSI is what Model.View uses, so the CLI, a Bubble Tea View and this
// layout all show the same pixels.
func frameANSI(b *cell.Buffer) string { return term.ANSI(b, term.TrueColor) }
