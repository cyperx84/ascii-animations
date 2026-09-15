// Package spinner is a drop-in replacement for charm.land/bubbles/v2/spinner.
//
// Migrating is an import change and nothing else:
//
//	-import "charm.land/bubbles/v2/spinner"
//	+import "github.com/cyperx84/ascii-animations/asciifx/spinner"
//
// Everything else — New, WithSpinner, WithStyle, the twelve predefined
// spinners, TickMsg, Update, View, Tick, ID, and the tag and ID filtering that
// stops a duplicated Tick from doubling the spin rate — has the same shape and
// the same behaviour, so existing code compiles and runs unchanged. The tests
// here are written in upstream's idiom for that reason: they are the
// compatibility claim, checked by the compiler.
//
//	// unchanged from upstream
//	s := spinner.New(spinner.WithSpinner(spinner.MiniDot))
//	func (m model) Init() tea.Cmd              { return m.s.Tick }
//	case spinner.TickMsg: m.s, cmd = m.s.Update(msg); return m, cmd
//	view := m.s.View() + " Loading"
//
// On top of that surface there are three opt-in extras for the rendering this
// project adds — WithLabel, WithPalette, WithShimmer — and this project's frame
// sets as Named and as package vars. Nothing in the upstream surface changes
// because of them, and all three are off unless asked for.
package spinner

import (
	"image/color"
	"math"
	"strings"
	"sync/atomic"
	"time"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/cyperx84/ascii-animations/asciifx/spinner/styles"
	"github.com/cyperx84/ascii-animations/asciifx/tint"
)

// nextID hands out instance ids, starting at 1 so a zero-value Model is
// recognisably different from one built by New.
var nextID atomic.Int64

// Spinner is a set of frames used in animating the spinner.
//
// FPS is a per-frame duration, not a rate; the name is upstream's.
type Spinner struct {
	Frames []string
	FPS    time.Duration
}

// The twelve spinners upstream defines, with upstream's exact frames and
// intervals, kept so that code referring to them by name still compiles and
// still lays out the same width.
//
// Globe, Moon and Monkey are emoji upstream, whose width varies between
// terminals — this project refuses to emit those anywhere else. They stay
// because dropping them would break the compatibility this package exists for.
// The Named frame sets are the single-width alternative.
var (
	Line = Spinner{
		Frames: []string{"|", "/", "-", "\\"},
		FPS:    time.Second / 10,
	}
	Dot = Spinner{
		// Trailing spaces are deliberate: they give the glyph a fixed width so
		// a label after it does not jitter.
		Frames: []string{"⣾ ", "⣽ ", "⣻ ", "⢿ ", "⡿ ", "⣟ ", "⣯ ", "⣷ "},
		FPS:    time.Second / 10,
	}
	MiniDot = Spinner{
		Frames: []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"},
		FPS:    time.Second / 12,
	}
	Jump = Spinner{
		Frames: []string{"⢄", "⢂", "⢁", "⡁", "⡈", "⡐", "⡠"},
		FPS:    time.Second / 10,
	}
	Pulse = Spinner{
		Frames: []string{"█", "▓", "▒", "░"},
		FPS:    time.Second / 8,
	}
	Points = Spinner{
		Frames: []string{"∙∙∙", "●∙∙", "∙●∙", "∙∙●"},
		FPS:    time.Second / 7,
	}
	Globe = Spinner{
		Frames: []string{"🌍", "🌎", "🌏"},
		FPS:    time.Second / 4,
	}
	Moon = Spinner{
		Frames: []string{"🌑", "🌒", "🌓", "🌔", "🌕", "🌖", "🌗", "🌘"},
		FPS:    time.Second / 8,
	}
	Monkey = Spinner{
		Frames: []string{"🙈", "🙉", "🙊"},
		FPS:    time.Second / 3,
	}
	Meter = Spinner{
		Frames: []string{
			"▱▱▱",
			"▰▱▱",
			"▰▰▱",
			"▰▰▰",
			"▰▰▱",
			"▰▱▱",
			"▱▱▱",
		},
		FPS: time.Second / 7,
	}
	Hamburger = Spinner{
		Frames: []string{"☱", "☲", "☴", "☲"},
		FPS:    time.Second / 3,
	}
	Ellipsis = Spinner{
		// The empty first frame is upstream's, and is why View can legitimately
		// render an empty string.
		Frames: []string{"", ".", "..", "..."},
		FPS:    time.Second / 3,
	}
)

// This project's frame sets, from the same table `asciifx info spinner`
// reports. They are single-width in every terminal and carry no emoji. Line and
// Pulse are absent because upstream already defines those names; Named("line")
// and Named("pulse") reach them.
var (
	Dots         = mustNamed("dots")
	Dots2        = mustNamed("dots2")
	Pipe         = mustNamed("pipe")
	Arc          = mustNamed("arc")
	Circle       = mustNamed("circle")
	Bounce       = mustNamed("bounce")
	Bar          = mustNamed("bar")
	Blocks       = mustNamed("blocks")
	BrailleSnake = mustNamed("braille-snake")
	Arrow        = mustNamed("arrow")
	Toggle       = mustNamed("toggle")
	Star         = mustNamed("star")
)

// Named returns one of this project's frame sets by the name `asciifx info
// spinner` reports, as a plain Spinner usable anywhere an upstream one is. The
// second result is false for an unknown name, so a typo fails where it is
// written instead of silently spinning something else.
func Named(name string) (Spinner, bool) {
	s, ok := styles.Lookup(name)
	if !ok {
		return Spinner{}, false
	}
	return Spinner{Frames: s.Frames, FPS: s.Interval}, true
}

// StyleNames lists the names Named accepts, sorted.
func StyleNames() []string { return styles.Names() }

// mustNamed is Named for the package vars, which are known good at init.
func mustNamed(name string) Spinner {
	s, ok := Named(name)
	if !ok {
		panic("asciifx: unknown spinner style " + name)
	}
	return s
}

// Model contains the state for the spinner. Use New to create new models rather
// than using Model as a struct literal.
type Model struct {
	// Spinner settings to use. See type Spinner.
	Spinner Spinner

	// Style sets the styling for the spinner. Most of the time you'll just
	// want foreground and background coloring, and potentially some padding.
	Style lipgloss.Style

	frame int
	id    int
	tag   int

	// The extras. All inert unless an Option set them, so a Model built by
	// upstream's code renders exactly as it used to.
	label    string
	hasLabel bool
	palette  tint.Gradient
	shimmer  float64
}

// ID returns the spinner's unique ID.
func (m Model) ID() int { return m.id }

// New returns a model with default values: the Line spinner at ten frames a
// second, unstyled, as upstream.
//
// It panics on an invalid WithPalette argument. New cannot return an error
// without breaking the signature this package exists to match, and a bad
// palette name is a mistake in the program rather than in user input, so it
// fails at startup with a message naming the problem.
func New(opts ...Option) Model {
	m := Model{
		Spinner: Line,
		id:      int(nextID.Add(1)),
	}
	for _, opt := range opts {
		opt(&m)
	}
	return m
}

// TickMsg indicates that the timer has ticked and we should render a frame.
//
// tag is this package's tick chain. It is unexported so it cannot be forged:
// it is what makes a duplicated Tick collapse into one chain instead of
// doubling the spin rate forever.
type TickMsg struct {
	Time time.Time

	tag int

	// ID is the ID of the spinner this message belongs to. A zero ID is
	// accepted by every spinner, which is what lets a message be built by hand
	// in a test or routed by hand in a program.
	ID int
}

// Update is the Tea update function.
func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	switch msg := msg.(type) {
	case TickMsg:
		// If an ID is set, and the ID doesn't belong to this spinner, reject
		// the message.
		if msg.ID > 0 && msg.ID != m.id {
			return m, nil
		}
		// If a tag is set and is not the tag of the chain we are on, this tick
		// belongs to a chain that has been superseded. Dropping it is what
		// collapses duplicate chains rather than running at their sum.
		if msg.tag > 0 && msg.tag != m.tag {
			return m, nil
		}
		if len(m.Spinner.Frames) > 0 {
			m.frame = (m.frame + 1) % len(m.Spinner.Frames)
		}
		m.tag++
		return m, m.tick(m.id, m.tag)
	}
	return m, nil
}

// View renders the model's view.
//
// The out-of-range case is upstream's, including that it fires for an empty
// frame list, so that a program which golden-tests this output keeps passing.
func (m Model) View() string {
	if m.frame >= len(m.Spinner.Frames) {
		return "(error)"
	}
	style := m.Style
	// One turn of the palette per turn of the frames.
	if c := m.glyphColour(); c.Valid {
		style = style.Foreground(rgba(c))
	}
	out := style.Render(m.Spinner.Frames[m.frame])
	if m.hasLabel && m.label != "" {
		out += " " + m.labelView()
	}
	return out
}

// labelView renders the label, with a highlight crossing it when the shimmer is
// on. With no shimmer the whole label is one Render call, so the common case
// emits one style; a shimmer varies per rune by construction, so it cannot be
// batched, which is part of why it is opt-in.
func (m Model) labelView() string {
	if m.shimmer <= 0 {
		return m.Style.Render(m.label)
	}
	runes := []rune(m.label)
	n := len(runes)
	// The band crosses the label, then rests, taking shimmer frames per
	// crossing: the same shape the spinner effect uses, with frames for time.
	span := float64(n + 6)
	period := m.shimmer * 1.5
	center := -3 + math.Mod(float64(m.frame), period)/period*span

	var sb strings.Builder
	var run []rune
	var prev tint.Color
	started := false
	flush := func() {
		if len(run) == 0 {
			return
		}
		style := m.Style
		if prev.Valid {
			style = style.Foreground(rgba(prev))
		}
		sb.WriteString(style.Render(string(run)))
		run = run[:0]
	}
	for i, r := range runes {
		base := sample(m.palette, float64(i)/float64(max(n, 1)))
		d := float64(i) - center
		k := math.Exp(-d * d / 4.5)
		col := tint.Lerp(base, tint.RGB(255, 255, 255), 0.45*k)
		if !started || col != prev {
			flush()
			prev, started = col, true
		}
		run = append(run, r)
	}
	flush()
	return sb.String()
}

// Tick is the command used to advance the spinner one frame. Use this command
// to effectively start the spinner.
//
// It returns a message rather than a command, as upstream does, so that
// `m.s.Tick` is itself usable as a tea.Cmd.
func (m Model) Tick() tea.Msg {
	return TickMsg{
		// The time at which the tick occurred.
		Time: time.Now(),

		// The ID of the spinner that this message belongs to. This can be
		// helpful when routing messages, however bear in mind that spinners
		// will ignore messages that don't contain ID by default.
		ID: m.id,

		tag: m.tag,
	}
}

// tick schedules the next tick of this chain.
func (m Model) tick(id, tag int) tea.Cmd {
	fps := m.Spinner.FPS
	if fps <= 0 {
		// A zero or negative interval would make tea.Tick fire in a hot loop.
		fps = time.Second / 10
	}
	return tea.Tick(fps, func(t time.Time) tea.Msg {
		return TickMsg{Time: t, ID: id, tag: tag}
	})
}

// Option is used to set options in New.
type Option func(*Model)

// WithSpinner sets the frame set.
func WithSpinner(s Spinner) Option {
	return func(m *Model) { m.Spinner = s }
}

// WithStyle sets the style, applied to the frame and to the label.
func WithStyle(s lipgloss.Style) Option {
	return func(m *Model) { m.Style = s }
}

// WithLabel sets text drawn after the frame. Upstream callers concatenate a
// label themselves, so this exists for the shimmering one; it is off by default
// because a library should not put words on screen uninvited.
func WithLabel(text string) Option {
	return func(m *Model) { m.label, m.hasLabel = text, true }
}

// WithPalette cycles the glyph, and the label highlight, through a palette given
// as a name or as comma-separated hex stops: WithPalette("nord") or
// WithPalette("#ff0000,#0000ff").
func WithPalette(name string) Option {
	return func(m *Model) {
		g, err := tint.ParsePalette(name)
		if err != nil {
			panic("asciifx: spinner palette: " + err.Error())
		}
		m.palette = g
	}
}

// WithShimmer sets how many frames the highlight takes to cross the label, or 0
// to leave it off, which is the default.
//
// The highlight advances one step per tick, so it is only as smooth as the frame
// rate: a shimmer wants a Spinner.FPS near 30ms, while the glyph alone is happy
// with the 70-250ms the frame sets use. That trade is why it is opt-in, and why
// the default keeps the cheap rate.
//
// It uses the palette from WithPalette, and gives the glyph that palette to
// cycle through as the spinner effect does. With no palette it falls back to the
// effect's default so the label still shimmers rather than silently doing
// nothing. Ordering against WithPalette does not matter.
func WithShimmer(frames float64) Option {
	return func(m *Model) {
		m.shimmer = frames
		if frames > 0 && len(m.palette) == 0 {
			m.palette = tint.Palettes[defaultShimmerPalette]
		}
	}
}

// defaultShimmerPalette is the palette the spinner effect uses by default, so
// the extras here look like `asciifx play spinner` does.
const defaultShimmerPalette = "catppuccin"

// glyphColour is the palette colour for the current frame, or an unset colour
// when no palette was chosen, in which case the frame keeps the user's style.
func (m Model) glyphColour() tint.Color {
	if len(m.palette) == 0 || len(m.Spinner.Frames) == 0 {
		return tint.None
	}
	return m.palette.Cyclic(float64(m.frame) / float64(len(m.Spinner.Frames)))
}

// sample reads a palette, tolerating an empty gradient.
func sample(g tint.Gradient, t float64) tint.Color {
	if len(g) == 0 {
		return tint.None
	}
	return g.Cyclic(t)
}

// rgba converts one of this project's colours for lipgloss, which takes
// image/color. An unset colour is nil: it means "the terminal's own colour",
// which image/color has no way to express, so callers must not pass it on.
func rgba(c tint.Color) color.Color {
	if !c.Valid {
		return nil
	}
	return color.RGBA{R: c.R, G: c.G, B: c.B, A: 0xFF}
}
