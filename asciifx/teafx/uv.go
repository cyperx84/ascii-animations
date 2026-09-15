// This file bridges asciifx's cell buffer to Charm's ultraviolet screen, so an
// effect can be one widget inside a lipgloss v2 layout instead of having to own
// the whole terminal.
//
// The two cell types are near-isomorphic, which is the point: a bubbletea v2 or
// lipgloss v2 program already has a uv.Screen, and this is the only conversion
// needed to put 18 effects into it.
package teafx

import (
	"image/color"
	"unicode/utf8"

	uv "github.com/charmbracelet/ultraviolet"

	"github.com/cyperx84/ascii-animations/asciifx/cell"
	"github.com/cyperx84/ascii-animations/asciifx/fx"
	"github.com/cyperx84/ascii-animations/asciifx/tint"
)

// ToUV converts an asciifx cell into an ultraviolet cell. An unset colour
// becomes a nil colour, which ultraviolet renders as the terminal default —
// the same meaning it has in asciifx, so a fade to "no colour" still fades to
// the terminal background rather than to black.
func ToUV(c cell.Cell) *uv.Cell {
	u := &uv.Cell{Content: string(runeOr(c.Rune)), Width: 1}
	u.Style.Fg = colorOf(c.FG)
	u.Style.Bg = colorOf(c.BG)
	if c.Attr&cell.Bold != 0 {
		u.Style.Attrs |= uv.AttrBold
	}
	if c.Attr&cell.Dim != 0 {
		u.Style.Attrs |= uv.AttrFaint
	}
	if c.Attr&cell.Italic != 0 {
		u.Style.Attrs |= uv.AttrItalic
	}
	if c.Attr&cell.Reverse != 0 {
		u.Style.Attrs |= uv.AttrReverse
	}
	if c.Attr&cell.Underline != 0 {
		u.Style.Underline = uv.UnderlineSingle
	}
	return u
}

// FromUV converts an ultraviolet cell into an asciifx cell. Only the first
// rune of a grapheme cluster survives, because a cell buffer holds one rune
// per cell; a cluster of more than one rune, or a wide glyph, becomes '?', the
// same substitution the buffer makes when it is handed unsafe text.
func FromUV(u *uv.Cell) cell.Cell {
	if u == nil {
		return cell.Blank
	}
	c := cell.Cell{Rune: clusterRune(u.Content), FG: tintOf(u.Style.Fg), BG: tintOf(u.Style.Bg)}
	if u.Style.Attrs&uv.AttrBold != 0 {
		c.Attr |= cell.Bold
	}
	if u.Style.Attrs&uv.AttrFaint != 0 {
		c.Attr |= cell.Dim
	}
	if u.Style.Attrs&uv.AttrItalic != 0 {
		c.Attr |= cell.Italic
	}
	if u.Style.Attrs&uv.AttrReverse != 0 {
		c.Attr |= cell.Reverse
	}
	if u.Style.Underline != uv.UnderlineNone {
		c.Attr |= cell.Underline
	}
	return c
}

// UV wraps a buffer as a uv.Drawable. Canvas.Compose hands a drawable the
// canvas's own bounds, so use At to place it in one region:
//
//	canvas.Compose(teafx.At(teafx.UV(buf), uv.Rect(2, 1, 40, 10)))
func UV(b *cell.Buffer) uv.Drawable { return blit{b} }

// blit adapts a buffer to uv.Drawable.
type blit struct{ b *cell.Buffer }

func (d blit) Draw(scr uv.Screen, area uv.Rectangle) { Blit(scr, area, d.b) }

// At pins a drawable to a fixed rectangle, clipped to whatever area it is
// finally given.
//
// Canvas.Compose gives every drawable the whole canvas, so a widget meant for
// one part of a layout has no way to say where it belongs. This is that way:
//
//	canvas := lipgloss.NewCanvas(60, 20)
//	canvas.Compose(teafx.At(model, uv.Rect(2, 1, 40, 10)))
//	canvas.Compose(teafx.At(other, uv.Rect(44, 1, 14, 10)))
//
// Inside your own widget, a parent that already hands out sub-areas should
// call Model.Draw directly and needs nothing from this.
func At(d uv.Drawable, area uv.Rectangle) uv.Drawable {
	return pinned{d: d, area: area}
}

type pinned struct {
	d    uv.Drawable
	area uv.Rectangle
}

func (p pinned) Draw(scr uv.Screen, area uv.Rectangle) {
	if r := p.area.Intersect(area); !r.Empty() {
		p.d.Draw(scr, r)
	}
}

// Draw paints the model's current frame into area, which makes a Model a
// uv.Drawable and therefore usable in any ultraviolet or lipgloss v2 layout.
//
// To place an effect in one region of a layout, pin it with At, because
// Canvas.Compose hands every drawable the whole canvas:
//
//	canvas := lipgloss.NewCanvas(60, 20)
//	canvas.Compose(teafx.At(model, uv.Rect(2, 1, 40, 10)))
//	fmt.Print(canvas.Render())
//
// A parent widget that hands its children sub-areas can call Draw directly
// with that sub-area, and the frame is clipped to it either way. For a plain
// string layout with no uv screen in sight, Model.View is the simpler path:
//
//	fmt.Print(lipgloss.NewCompositor(
//		lipgloss.NewLayer(model.View()).X(2).Y(1),
//	).Render())
func (m Model) Draw(scr uv.Screen, area uv.Rectangle) { Blit(scr, area, m.buf) }

// Blit paints b with its top-left at area's origin, clipped to area. It is the
// single place that knows how an asciifx buffer becomes a Charm screen, so a
// Model, a Canvas and a bare Buffer all render identically.
func Blit(scr uv.Screen, area uv.Rectangle, b *cell.Buffer) {
	if scr == nil || b == nil {
		return
	}
	w, h := min(b.W, area.Dx()), min(b.H, area.Dy())
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			scr.SetCell(area.Min.X+x, area.Min.Y+y, ToUV(b.Cells[y*b.W+x]))
		}
	}
}

// Snapshot copies area of scr into b, resizing b to the area. Cells outside
// the screen become blanks.
//
// Use it to animate content that is already on screen: capture a region, hand
// it to a transition, and the effect transforms your layout instead of
// replacing it. For that, Content is the one-liner.
func Snapshot(b *cell.Buffer, scr uv.Screen, area uv.Rectangle) {
	w, h := max(area.Dx(), 0), max(area.Dy(), 0)
	b.Resize(w, h)
	if scr == nil {
		return
	}
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			b.Set(x, y, FromUV(scr.CellAt(area.Min.X+x, area.Min.Y+y)))
		}
	}
}

// Content returns fx.Content that seeds a transition with what is already on
// screen in area. Pass it as fx.Options.Content:
//
//	run, err := fx.NewRun(spec, fx.Options{
//		W: area.Dx(), H: area.Dy(),
//		Content: teafx.Content(screen, area),
//	})
//
// The snapshot is taken once, when Content is called, so it captures the frame
// you are about to animate.
func Content(scr uv.Screen, area uv.Rectangle) fx.Content {
	b := cell.New(area.Dx(), area.Dy())
	Snapshot(b, scr, area)
	return func(dst *cell.Buffer) { dst.CopyFrom(b) }
}

// colorOf maps an asciifx colour to a charm colour, keeping "unset" distinct
// from black.
func colorOf(c tint.Color) color.Color {
	if !c.Valid {
		return nil
	}
	return color.RGBA{R: c.R, G: c.G, B: c.B, A: 0xFF}
}

// tintOf maps a charm colour back to an asciifx colour. Fully transparent is
// treated as unset, which is how ultraviolet spells "terminal default".
func tintOf(c color.Color) tint.Color {
	if c == nil {
		return tint.None
	}
	r, g, b, a := c.RGBA()
	if a == 0 {
		return tint.None
	}
	return tint.RGB(uint8(r>>8), uint8(g>>8), uint8(b>>8))
}

// runeOr turns a buffer's zero rune into a space, matching cell.runeOf.
func runeOr(r rune) rune {
	if r == 0 {
		return ' '
	}
	return r
}

// clusterRune reduces a grapheme cluster to the one rune a cell buffer can
// hold, using the same "unsafe becomes ?" rule as cell.Buffer.WriteString.
//
// A cluster of more than one rune becomes '?'. Decoding only the first rune
// would silently drop the rest, so "e" followed by a combining acute would
// arrive as a bare "e" — a different character than the one on screen.
func clusterRune(content string) rune {
	if content == "" {
		return ' '
	}
	r, size := utf8.DecodeRuneInString(content)
	if r == utf8.RuneError || size != len(content) || cell.Width(r) != 1 {
		return '?'
	}
	return r
}
