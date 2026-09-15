// Package ui provides the main TUI model and shared components.
package ui

import (
	"fmt"
	"math/rand"
	"strings"
	"time"

	"github.com/cyperx84/ascii-animations/pkg/banners"
	"github.com/cyperx84/ascii-animations/pkg/colors"
	"github.com/cyperx84/ascii-animations/pkg/effects"
	"github.com/cyperx84/ascii-animations/pkg/export"
	"github.com/cyperx84/ascii-animations/pkg/spinners"
	"github.com/cyperx84/ascii-animations/pkg/splash"
	"github.com/cyperx84/ascii-animations/pkg/theme"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

// Animation represents a single animation that can be previewed.
type Animation struct {
	Name        string
	Description string
	Library     string
	SourceCode  string
	Interval    time.Duration
	Frames      []string
	RenderFunc  func(w, h, frame int) string
}

// Category groups related animations.
type Category struct {
	Name       string
	Icon       string
	Animations []Animation
}

type viewState int

const (
	stateMenu viewState = iota
	stateAnimation
	stateBannerInput
)

type tickMsg time.Time

// Speed multiplier levels (index into speedMultipliers).
var speedMultipliers = []float64{0.25, 0.5, 1.0, 2.0, 4.0}
var speedLabels = []string{"0.25x", "0.5x", "1x", "2x", "4x"}

const defaultSpeedIdx = 2 // 1x

// maxBannerRunes bounds the custom banner text. It counts runes rather than
// bytes so the limit is the same however wide the characters are.
const maxBannerRunes = 20

// Model is the top-level Bubble Tea model.
type Model struct {
	state      viewState
	categories []Category
	menuCursor int
	animCursor int
	frame      int
	width      int
	height     int
	showSource bool
	exportMsg  string
	bannerText string // custom text for banner input
	speedIdx   int    // index into speedMultipliers
	rng        *rand.Rand
}

// NewModel returns a fresh Model with all animation categories loaded.
func NewModel() Model {
	return Model{
		categories: allCategories(),
		bannerText: "HELLO",
		speedIdx:   defaultSpeedIdx,
		rng:        rand.New(rand.NewSource(time.Now().UnixNano())),
	}
}

// Init starts with no command: the menu needs no ticks, and the window title
// is a property of the View in Bubble Tea v2 rather than a command.
func (m Model) Init() tea.Cmd { return nil }

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case tea.KeyPressMsg:
		m.exportMsg = ""
		return m.handleKey(msg)

	case tickMsg:
		m.frame++
		return m, m.tickCmd()
	}
	return m, nil
}

func (m Model) handleKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch m.state {
	case stateMenu:
		return m.handleMenuKey(msg)
	case stateAnimation:
		return m.handleAnimKey(msg)
	case stateBannerInput:
		return m.handleBannerInput(msg)
	}
	return m, nil
}

func (m Model) handleMenuKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "q", "ctrl+c":
		return m, tea.Quit
	case "up", "k":
		if m.menuCursor > 0 {
			m.menuCursor--
		}
	case "down", "j":
		if m.menuCursor < len(m.categories)-1 {
			m.menuCursor++
		}
	case "enter", "space":
		m.state = stateAnimation
		m.animCursor = 0
		m.frame = 0
		m.showSource = false
		m.speedIdx = defaultSpeedIdx
		return m, m.tickCmd()
	}
	return m, nil
}

func (m Model) handleAnimKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	cat := m.categories[m.menuCursor]
	switch msg.String() {
	case "q", "esc":
		m.state = stateMenu
		m.showSource = false
		return m, nil
	case "ctrl+c":
		return m, tea.Quit
	case "left", "h":
		if m.animCursor > 0 {
			m.animCursor--
			m.frame = 0
		}
	case "right", "l":
		if m.animCursor < len(cat.Animations)-1 {
			m.animCursor++
			m.frame = 0
		}
	case "s":
		m.showSource = !m.showSource
	case "e":
		anim := cat.Animations[m.animCursor]
		var frames []string
		if len(anim.Frames) > 0 {
			frames = anim.Frames
		} else if anim.RenderFunc != nil {
			for i := 0; i < 20; i++ {
				frames = append(frames, anim.RenderFunc(60, 20, i))
			}
		}
		path, err := export.Generate(anim.Name, frames, anim.Interval, "exported")
		if err != nil {
			m.exportMsg = "Export failed: " + err.Error()
		} else {
			m.exportMsg = "Exported to " + path
		}
	case "t":
		// custom text input for banners
		if cat.Name == "Text Banners" {
			m.state = stateBannerInput
			m.bannerText = ""
			return m, nil
		}
	case "+", "=":
		if m.speedIdx < len(speedMultipliers)-1 {
			m.speedIdx++
		}
	case "-", "_":
		if m.speedIdx > 0 {
			m.speedIdx--
		}
	case "r":
		// random animation
		if len(cat.Animations) > 1 {
			m.animCursor = m.rng.Intn(len(cat.Animations))
			m.frame = 0
		}
	}
	return m, nil
}

func (m Model) handleBannerInput(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "ctrl+c":
		return m, tea.Quit
	case "esc":
		m.state = stateAnimation
		return m, m.tickCmd()
	case "enter":
		if m.bannerText != "" {
			m.rebuildBannerCategory()
		}
		m.state = stateAnimation
		m.animCursor = 0
		m.frame = 0
		return m, m.tickCmd()
	case "backspace":
		// Slice runes, not bytes: a multi-byte rune removed by byte would leave
		// an invalid UTF-8 prefix behind.
		if r := []rune(m.bannerText); len(r) > 0 {
			m.bannerText = string(r[:len(r)-1])
		}
	default:
		// Key.Text is the literal text a key produced, so this accepts shifted
		// and pasted characters that String would spell out. Only runes some
		// font can actually draw are accepted, because Render substitutes a
		// blank for anything else: taking them would look like nothing
		// happened while still putting the rune in the string.
		if r := []rune(msg.Text); len(r) == 1 && banners.Renderable(r[0]) && len([]rune(m.bannerText)) < maxBannerRunes {
			m.bannerText += string(r[0])
		}
	}
	return m, nil
}

func (m *Model) rebuildBannerCategory() {
	var anims []Animation
	for _, f := range banners.AllFonts() {
		rendered := banners.Render(m.bannerText, f.Chars)
		anims = append(anims, Animation{
			Name:        f.Name + ": " + strings.ToUpper(m.bannerText),
			Description: f.Name + " font rendering of \"" + m.bannerText + "\"",
			Library:     "figlet (300+ fonts) · toilet (color) · go-figure (Go)",
			SourceCode:  fmt.Sprintf("import \"github.com/common-nighthawk/go-figure\"\n\nfig := figure.NewFigure(\"%s\", \"%s\", true)\nfig.Print()", m.bannerText, strings.ToLower(f.Name)),
			Interval:    500 * time.Millisecond,
			Frames:      []string{rendered},
		})
	}
	// find and replace banner category
	for i, cat := range m.categories {
		if cat.Name == "Text Banners" {
			m.categories[i].Animations = anims
			break
		}
	}
}

func (m Model) tickCmd() tea.Cmd {
	if m.state != stateAnimation {
		return nil
	}
	cat := m.categories[m.menuCursor]
	if m.animCursor >= len(cat.Animations) {
		return nil
	}
	anim := cat.Animations[m.animCursor]
	interval := anim.Interval
	if interval == 0 {
		interval = 100 * time.Millisecond
	}
	// apply speed multiplier (higher multiplier = faster = shorter interval)
	adjusted := time.Duration(float64(interval) / speedMultipliers[m.speedIdx])
	return tea.Tick(adjusted, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}

// View returns the styled frame plus the screen mode it needs. Bubble Tea v2
// moved the alternate screen, the mouse mode and the window title from program
// options onto the View, so the model carries them itself.
func (m Model) View() tea.View {
	v := tea.NewView(m.body())
	v.AltScreen = true
	v.MouseMode = tea.MouseModeCellMotion
	v.WindowTitle = "ASCII Animations Showcase"
	return v
}

// body is the frame without the v2 screen settings, so it can be tested and
// composed on its own.
func (m Model) body() string {
	switch m.state {
	case stateMenu:
		return m.menuView()
	case stateAnimation:
		return m.animView()
	case stateBannerInput:
		return m.bannerInputView()
	}
	return ""
}

func (m Model) menuView() string {
	var sb strings.Builder

	title := theme.Title.Render("  ASCII Animations Showcase")
	sb.WriteString(title)
	sb.WriteString("\n\n")

	for i, cat := range m.categories {
		cursor := "  "
		style := theme.MenuItem
		if i == m.menuCursor {
			cursor = "▸ "
			style = theme.SelectedItem
		}
		count := len(cat.Animations)
		line := fmt.Sprintf("%s%s %s (%d)", cursor, cat.Icon, cat.Name, count)
		sb.WriteString(style.Render(line))
		sb.WriteString("\n")
	}

	sb.WriteString("\n")
	sb.WriteString(theme.Footer.Render("  Powered by research from cyperx84/openclaw"))
	sb.WriteString("\n\n")
	sb.WriteString(theme.Help.Render("  j/k ↑↓ navigate · enter select · q quit"))

	return sb.String()
}

func (m Model) animView() string {
	cat := m.categories[m.menuCursor]
	if m.animCursor >= len(cat.Animations) {
		return ""
	}
	anim := cat.Animations[m.animCursor]

	var sb strings.Builder

	// header with animation info
	header := fmt.Sprintf("  %s %s  (%d/%d)", cat.Icon, cat.Name, m.animCursor+1, len(cat.Animations))
	sb.WriteString(theme.Header.Render(header))
	sb.WriteString("\n")

	// info panel: name, description, library, speed
	infoLine := theme.AnimName.Render("  "+anim.Name) + "  " + theme.AnimDesc.Render(anim.Description)
	sb.WriteString(infoLine)
	sb.WriteString("\n")
	sb.WriteString(theme.Lib.Render("  Lib: " + anim.Library))
	sb.WriteString("  ")
	sb.WriteString(theme.Help.Render("Speed: " + speedLabels[m.speedIdx]))
	sb.WriteString("\n\n")

	// animation preview area
	previewW := m.width - 6
	if previewW < 20 {
		previewW = 20
	}
	previewH := m.height - 14
	if previewH < 5 {
		previewH = 5
	}

	var content string
	if anim.RenderFunc != nil {
		content = anim.RenderFunc(previewW-4, previewH-2, m.frame)
	} else if len(anim.Frames) > 0 {
		idx := m.frame % len(anim.Frames)
		content = anim.Frames[idx]
	}

	// center spinners in the preview area
	if cat.Name == "Spinners" && len(content) < 20 {
		padV := previewH / 2
		padH := previewW / 2
		var centered strings.Builder
		for i := 0; i < padV; i++ {
			centered.WriteByte('\n')
		}
		centered.WriteString(strings.Repeat(" ", padH))
		centered.WriteString(content)
		content = centered.String()
	}

	preview := theme.Preview.Width(previewW).Height(previewH).Render(content)
	sb.WriteString(preview)
	sb.WriteString("\n")

	// source code toggle
	if m.showSource {
		src := theme.Source.Width(previewW).Render(anim.SourceCode)
		sb.WriteString(src)
		sb.WriteString("\n")
	}

	// export message
	if m.exportMsg != "" {
		sb.WriteString(theme.ExportMsg.Render("  " + m.exportMsg))
		sb.WriteString("\n")
	}

	// footer help
	nav := "  h/l ←→ cycle"
	if len(cat.Animations) > 1 {
		nav += fmt.Sprintf(" (%d animations)", len(cat.Animations))
	}
	nav += " · r random · +/- speed · s source · e export"
	if cat.Name == "Text Banners" {
		nav += " · t custom text"
	}
	nav += " · q/esc back"
	sb.WriteString(theme.Help.Render(nav))

	return lipgloss.NewStyle().MaxWidth(m.width).MaxHeight(m.height).Render(sb.String())
}

func (m Model) bannerInputView() string {
	var sb strings.Builder
	sb.WriteString(theme.Title.Render("  Custom Banner Text"))
	sb.WriteString("\n\n")
	sb.WriteString(theme.InputStyle.Render("  Type your text: "))
	sb.WriteString(theme.AnimName.Render(m.bannerText))
	sb.WriteString(theme.InputStyle.Render("█"))
	sb.WriteString("\n\n")

	// live preview
	if m.bannerText != "" {
		fonts := banners.AllFonts()
		if len(fonts) > 0 {
			preview := banners.Render(m.bannerText, fonts[0].Chars)
			sb.WriteString(theme.Preview.Render(preview))
		}
	}

	sb.WriteString("\n\n")
	sb.WriteString(theme.Help.Render("  enter confirm · esc cancel"))
	return sb.String()
}

// allCategories builds all animation categories.
func allCategories() []Category {
	return []Category{
		spinnerCategory(),
		effectsCategory(),
		bannerCategory(),
		splashCategory(),
		colorCategory(),
		exportCategory(),
	}
}

func spinnerCategory() Category {
	var anims []Animation
	for _, s := range spinners.All() {
		anims = append(anims, Animation{
			Name:        s.Name,
			Description: s.Desc + " [" + s.Category + "]",
			Library:     "briandowns/spinner (90+ presets) · cli-spinners JSON format",
			SourceCode:  spinners.SourceSnippet,
			Interval:    spinners.Interval,
			Frames:      s.Frames,
		})
	}
	return Category{Name: "Spinners", Icon: "🎯", Animations: anims}
}

func effectsCategory() Category {
	return Category{
		Name: "Full-Screen Effects",
		Icon: "🔥",
		Animations: []Animation{
			{
				Name: "Matrix Rain", Description: "Digital rain inspired by The Matrix",
				Library: "Custom · Bubble Tea tick model", SourceCode: effects.SourceMatrix,
				Interval: 60 * time.Millisecond, RenderFunc: effects.RenderMatrix,
			},
			{
				Name: "Fire", Description: "Rising flame simulation",
				Library: "Custom · Bubble Tea tick model", SourceCode: effects.SourceFire,
				Interval: 70 * time.Millisecond, RenderFunc: effects.RenderFire,
			},
			{
				Name: "Rain", Description: "Falling rain with splash effects",
				Library: "Custom · Bubble Tea tick model", SourceCode: effects.SourceRain,
				Interval: 60 * time.Millisecond, RenderFunc: effects.RenderRain,
			},
			{
				Name: "Starfield", Description: "Warp-speed starfield zooming outward",
				Library: "Custom · Bubble Tea tick model", SourceCode: effects.SourceStarfield,
				Interval: 50 * time.Millisecond, RenderFunc: effects.RenderStarfield,
			},
			{
				Name: "Snow", Description: "Falling snowflakes with wind drift and accumulation",
				Library: "Custom · Bubble Tea tick model", SourceCode: effects.SourceSnow,
				Interval: 80 * time.Millisecond, RenderFunc: effects.RenderSnow,
			},
			{
				Name: "DNA Helix", Description: "Rotating double helix with base pairs",
				Library: "Custom · Bubble Tea tick model", SourceCode: effects.SourceDNA,
				Interval: 60 * time.Millisecond, RenderFunc: effects.RenderDNA,
			},
			{
				Name: "Wave", Description: "Colorful sine waves scrolling across the screen",
				Library: "Custom · Bubble Tea tick model", SourceCode: effects.SourceWave,
				Interval: 50 * time.Millisecond, RenderFunc: effects.RenderWave,
			},
			{
				Name: "Plasma", Description: "Organic plasma patterns with color cycling",
				Library: "Custom · Bubble Tea tick model", SourceCode: effects.SourcePlasma,
				Interval: 60 * time.Millisecond, RenderFunc: effects.RenderPlasma,
			},
		},
	}
}

func bannerCategory() Category {
	sampleTexts := []string{"HELLO", "ASCII", "GO TUI", "CHARM", "CYBER"}
	var anims []Animation
	for _, f := range banners.AllFonts() {
		for _, text := range sampleTexts {
			rendered := banners.Render(text, f.Chars)
			anims = append(anims, Animation{
				Name:        f.Name + ": " + text,
				Description: f.Name + " font rendering of \"" + text + "\"",
				Library:     "figlet (300+ fonts) · toilet (color) · go-figure (Go)",
				SourceCode:  fmt.Sprintf("%s\n\nfig := figure.NewFigure(\"%s\", \"%s\", true)\nfig.Print()", banners.SourceSnippet, text, strings.ToLower(f.Name)),
				Interval:    500 * time.Millisecond,
				Frames:      []string{rendered},
			})
		}
	}
	return Category{Name: "Text Banners", Icon: "📝", Animations: anims}
}

func splashCategory() Category {
	return Category{
		Name: "Splash Screens",
		Icon: "🎨",
		Animations: []Animation{
			{
				Name: "Typing Effect", Description: "Text appears character by character with blinking cursor",
				Library: "Custom · Bubble Tea tick model", SourceCode: splash.SourceTyping,
				Interval: 60 * time.Millisecond, Frames: splash.TypingFrames("ASCII Animations Showcase"),
			},
			{
				Name: "Expanding Border", Description: "Logo box appears line by line",
				Library: "Lip Gloss borders · Custom animation", SourceCode: splash.SourceExpand,
				Interval: 150 * time.Millisecond, Frames: splash.ExpandBorderFrames(),
			},
			{
				Name: "Fade In", Description: "Logo fades in through density characters",
				Library: "Custom · Block characters ░▒▓█", SourceCode: splash.SourceFade,
				Interval: 200 * time.Millisecond, Frames: splash.FadeInFrames(),
			},
			{
				Name: "Spinner Splash", Description: "Loading spinner transitions to logo reveal",
				Library: "Custom · Braille + box drawing", SourceCode: splash.SourceSpinner,
				Interval: 80 * time.Millisecond, Frames: splash.SpinnerSplashFrames(),
			},
			{
				Name: "Glitch Reveal", Description: "Text un-corrupts from random glitch characters",
				Library: "Custom · Character substitution", SourceCode: splash.SourceGlitch,
				Interval: 60 * time.Millisecond, Frames: splash.GlitchFrames(),
			},
			{
				Name: "Scan Line", Description: "Bright scan line sweeps top to bottom revealing content",
				Library: "Custom · ANSI reverse video", SourceCode: splash.SourceScanLine,
				Interval: 100 * time.Millisecond, Frames: splash.ScanLineFrames(),
			},
		},
	}
}

func colorCategory() Category {
	return Category{
		Name: "Color Showcase",
		Icon: "🌈",
		Animations: []Animation{
			{
				Name: "16 Colors", Description: "Standard ANSI 16-color palette",
				Library: "ANSI escape codes · lipgloss.Color(\"1\"-\"15\")", SourceCode: colors.Source16,
				Interval: time.Second, Frames: []string{colors.Render16()},
			},
			{
				Name: "256 Colors", Description: "Full 256-color terminal palette",
				Library: "ANSI 256 · lipgloss.Color(\"0\"-\"255\")", SourceCode: colors.Source256,
				Interval: time.Second, Frames: []string{colors.Render256()},
			},
			{
				Name: "Truecolor", Description: "24-bit RGB color gradient",
				Library: "ANSI truecolor · lipgloss.Color(\"#rrggbb\")", SourceCode: colors.SourceTrue,
				Interval: 100 * time.Millisecond, RenderFunc: colors.RenderTruecolor,
			},
		},
	}
}

func exportCategory() Category {
	return Category{
		Name: "Export",
		Icon: "📦",
		Animations: []Animation{
			{
				Name:        "Export Demo",
				Description: "Press 'e' on any animation to export as standalone Go code",
				Library:     "Built-in exporter · Generates minimal Bubble Tea programs",
				SourceCode:  "// Navigate to any animation and press 'e' to export\n// The exported file will be a complete, runnable Go program\n// that reproduces the selected animation.",
				Interval:    time.Second,
				Frames: []string{
					"  📦 Export any animation as standalone Go code\n\n" +
						"  How to use:\n" +
						"  1. Navigate to any animation (Spinners, Effects, etc.)\n" +
						"  2. Press 'e' to export the current animation\n" +
						"  3. Find the exported .go file in the 'exported/' directory\n" +
						"  4. Run it with: go run exported/<name>.go\n\n" +
						"  The exported code is a complete, standalone program\n" +
						"  using only the standard library — no dependencies needed.",
				},
			},
		},
	}
}
