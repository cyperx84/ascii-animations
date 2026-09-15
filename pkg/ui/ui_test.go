package ui

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/cyperx84/ascii-animations/pkg/banners"
)

// These tests pin the Bubble Tea v2 contract the showcase migrated to. They are
// deliberately not pty tests: a pty with no terminal emulator cannot answer the
// framework's capability probe, so a real terminal is not available in CI.

func key(code rune) tea.KeyPressMsg { return tea.KeyPressMsg{Code: code} }

// typed builds the message a terminal sends for a printable character.
func typed(r rune) tea.KeyPressMsg { return tea.KeyPressMsg{Code: r, Text: string(r)} }

// send feeds messages to the model and returns the last command.
func send(m Model, msgs ...tea.Msg) (Model, tea.Cmd) {
	var cmd tea.Cmd
	for _, msg := range msgs {
		var next tea.Model
		next, cmd = m.Update(msg)
		m = next.(Model)
	}
	return m, cmd
}

func sized(t *testing.T) Model {
	t.Helper()
	m, _ := send(NewModel(), tea.WindowSizeMsg{Width: 80, Height: 24})
	return m
}

// TestViewCarriesTheScreenSettings is the heart of the v2 migration: the
// alternate screen, mouse mode and window title moved from program options
// onto the View, so dropping them would silently change how the TUI runs.
func TestViewCarriesTheScreenSettings(t *testing.T) {
	v := sized(t).View()
	if !v.AltScreen {
		t.Error("View lost AltScreen; the TUI would draw over the shell scrollback")
	}
	if v.MouseMode != tea.MouseModeCellMotion {
		t.Errorf("MouseMode = %v, want MouseModeCellMotion", v.MouseMode)
	}
	if v.WindowTitle != "ASCII Animations Showcase" {
		t.Errorf("WindowTitle = %q", v.WindowTitle)
	}
	if !strings.Contains(v.Content, "ASCII Animations Showcase") {
		t.Errorf("menu is missing its heading:\n%s", v.Content)
	}
}

func TestInitHasNoCommand(t *testing.T) {
	if cmd := NewModel().Init(); cmd != nil {
		t.Error("Init should not need a command: the menu animates nothing")
	}
}

func TestMenuNavigation(t *testing.T) {
	m := sized(t)
	if m.menuCursor != 0 {
		t.Fatalf("cursor starts at %d", m.menuCursor)
	}
	// k at the top must not wrap or go negative.
	m, _ = send(m, key(tea.KeyUp))
	if m.menuCursor != 0 {
		t.Errorf("up at the top moved the cursor to %d", m.menuCursor)
	}
	for i := 0; i < len(m.categories)-1; i++ {
		m, _ = send(m, typed('j'))
	}
	if want := len(m.categories) - 1; m.menuCursor != want {
		t.Errorf("cursor = %d after walking to the end, want %d", m.menuCursor, want)
	}
	// j at the bottom must stop there.
	m, _ = send(m, typed('j'))
	if want := len(m.categories) - 1; m.menuCursor != want {
		t.Errorf("down at the bottom moved the cursor to %d", m.menuCursor)
	}
	m, _ = send(m, key(tea.KeyUp))
	if m.menuCursor != len(m.categories)-2 {
		t.Errorf("cursor = %d after one step up", m.menuCursor)
	}
}

func TestEnterOpensACategoryAndStartsTicking(t *testing.T) {
	for _, open := range []struct {
		name string
		key  tea.KeyPressMsg
	}{
		{"enter", key(tea.KeyEnter)},
		// The space bar stringifies as "space" in Bubble Tea v2, not " ", so
		// a stale " " case matches nothing and the key silently dies.
		{"space", key(tea.KeySpace)},
	} {
		t.Run(open.name, func(t *testing.T) {
			m, cmd := send(sized(t), open.key)
			if m.state != stateAnimation {
				t.Fatalf("state = %v after %s, want stateAnimation", m.state, open.name)
			}
			if cmd == nil {
				t.Fatalf("%s must return a tick command", open.name)
			}
			if m.frame != 0 || m.animCursor != 0 {
				t.Errorf("frame %d cursor %d after %s, want 0 and 0", m.frame, m.animCursor, open.name)
			}
		})
	}
	// A tick advances the frame and asks for the next one.
	m, cmd := send(sized(t), key(tea.KeyEnter))
	m, cmd = send(m, tickMsg{})
	if m.frame != 1 {
		t.Errorf("frame = %d after one tick", m.frame)
	}
	if cmd == nil {
		t.Error("a tick must schedule the next tick")
	}
	// Esc returns to the menu.
	m, _ = send(m, key(tea.KeyEsc))
	if m.state != stateMenu {
		t.Errorf("state = %v after esc", m.state)
	}
}

// TestEveryDocumentedMenuKeyDoesSomething guards the class of bug the space key
// was: a key name that no longer matches what the key stringifies to in the
// framework's current major version. The strings here are the ones v2 actually
// produces, checked against KeyPressMsg.String rather than assumed.
func TestEveryDocumentedMenuKeyDoesSomething(t *testing.T) {
	keys := []struct {
		msg  tea.KeyPressMsg
		want string
	}{
		{key(tea.KeyUp), "up"},
		{key(tea.KeyDown), "down"},
		{key(tea.KeyEnter), "enter"},
		{key(tea.KeySpace), "space"},
		{typed('j'), "j"},
		{typed('k'), "k"},
		{typed('q'), "q"},
	}
	for _, c := range keys {
		if got := c.msg.String(); got != c.want {
			t.Errorf("key stringifies as %q, but the handlers match %q", got, c.want)
		}
	}
	// Space is the odd one out, and the reason " " must not appear in a
	// handler: a terminal sends it with Text " ", and String() still reports
	// "space" because the framework refuses to stringify a bare space. Both
	// shapes a message can arrive in must match the same case.
	for _, space := range []tea.KeyPressMsg{
		{Code: tea.KeySpace},
		{Code: tea.KeySpace, Text: " "},
	} {
		if got := space.String(); got != "space" {
			t.Errorf("space with Text %q stringifies as %q, want space", space.Text, got)
		}
	}
}

func TestQuitKeys(t *testing.T) {
	// Commands are functions, so the quit is identified by the message it
	// produces rather than by comparing the functions themselves.
	quit := func(cmd tea.Cmd) bool {
		if cmd == nil {
			return false
		}
		_, isQuit := cmd().(tea.QuitMsg)
		return isQuit
	}
	if _, cmd := send(sized(t), typed('q')); !quit(cmd) {
		t.Error("q in the menu should quit")
	}
	if _, cmd := send(sized(t), tea.KeyPressMsg{Code: 'c', Mod: tea.ModCtrl}); !quit(cmd) {
		t.Error("ctrl+c in the menu should quit")
	}
	// Inside a category, q goes back and only ctrl+c quits.
	m, cmd := send(sized(t), key(tea.KeyEnter))
	if quit(cmd) {
		t.Fatal("enter should not quit")
	}
	m, cmd = send(m, typed('q'))
	if quit(cmd) || m.state != stateMenu {
		t.Errorf("q inside a category should go back, state = %v", m.state)
	}
	if _, cmd := send(m, tea.KeyPressMsg{Code: 'c', Mod: tea.ModCtrl}); !quit(cmd) {
		t.Error("ctrl+c inside a category should quit")
	}
}

// TestBannerInputUsesKeyText covers the one place the migration could break
// silently: v2 exposes the literal typed text on the key message, and reading
// the wrong field would make the text field appear to do nothing.
func TestBannerInputUsesKeyText(t *testing.T) {
	m := sized(t)
	// Walk to the Text Banners category.
	for m.categories[m.menuCursor].Name != "Text Banners" {
		m, _ = send(m, typed('j'))
	}
	m, _ = send(m, key(tea.KeyEnter))
	m, _ = send(m, typed('t'))
	if m.state != stateBannerInput {
		t.Fatalf("state = %v after t on banners, want stateBannerInput", m.state)
	}
	if m.bannerText != "" {
		t.Fatalf("typing mode started with %q in the field", m.bannerText)
	}
	m, _ = send(m, typed('O'), typed('K'))
	if m.bannerText != "OK" {
		t.Fatalf("banner text = %q, want OK", m.bannerText)
	}
	// Backspace edits it.
	m, _ = send(m, key(tea.KeyBackspace))
	if m.bannerText != "O" {
		t.Fatalf("banner text = %q after backspace", m.bannerText)
	}
	m, _ = send(m, typed('K'))
	m, cmd := send(m, key(tea.KeyEnter))
	if m.state != stateAnimation {
		t.Fatalf("state = %v after confirming the text", m.state)
	}
	if cmd == nil {
		t.Error("confirming should restart the tick")
	}
	if m.bannerText != "OK" {
		t.Fatalf("confirmed text is %q", m.bannerText)
	}
	// The category was rebuilt for the new text: 12 fonts, one text each.
	for _, cat := range m.categories {
		if cat.Name != "Text Banners" {
			continue
		}
		if len(cat.Animations) != len(banners.AllFonts()) {
			t.Errorf("Text Banners has %d animations, want %d", len(cat.Animations), len(banners.AllFonts()))
		}
		for _, a := range cat.Animations {
			if !strings.Contains(a.Name, "OK") {
				t.Errorf("animation %q does not name the new text", a.Name)
			}
		}
	}
}

func TestBannerInputCancelKeepsTheOldText(t *testing.T) {
	m := sized(t)
	for m.categories[m.menuCursor].Name != "Text Banners" {
		m, _ = send(m, typed('j'))
	}
	m, _ = send(m, key(tea.KeyEnter), typed('t'), typed('X'))
	if m.bannerText != "X" {
		t.Fatalf("field holds %q", m.bannerText)
	}
	m, _ = send(m, key(tea.KeyEsc))
	if m.state != stateAnimation {
		t.Fatalf("state = %v after esc", m.state)
	}
	if m.bannerText != "X" {
		t.Fatalf("esc should leave the field alone, got %q", m.bannerText)
	}
	// Re-entering typing mode starts fresh, as before the migration.
	m, _ = send(m, typed('t'))
	if m.bannerText != "" {
		t.Fatalf("re-entering typing mode kept %q", m.bannerText)
	}
}

func TestAnimationViewRespectsTheWindowSize(t *testing.T) {
	m := sized(t)
	m, _ = send(m, key(tea.KeyEnter)) // Spinners
	small, _ := send(m, tea.WindowSizeMsg{Width: 40, Height: 14})
	wide, _ := send(m, tea.WindowSizeMsg{Width: 120, Height: 40})
	if a, b := small.animView(), wide.animView(); len(a) >= len(b) {
		t.Errorf("a wider window did not produce a wider view (%d vs %d bytes)", len(a), len(b))
	}
	// A window too small to hold the preview must not panic or produce nothing.
	tiny, _ := send(m, tea.WindowSizeMsg{Width: 10, Height: 6})
	if tiny.animView() == "" {
		t.Error("a tiny window produced an empty view")
	}
}

func TestRandomStaysInRange(t *testing.T) {
	// NewModel seeds from the clock, so this checks the bound, not a value.
	m := sized(t)
	m, _ = send(m, key(tea.KeyEnter))
	for i := 0; i < 50; i++ {
		m, _ = send(m, typed('r'))
		if n := len(m.categories[m.menuCursor].Animations); m.animCursor >= n {
			t.Fatalf("random picked %d of %d", m.animCursor, n)
		}
	}
}

func TestSpeedBoundsAndCycleKeys(t *testing.T) {
	m := sized(t)
	m, _ = send(m, key(tea.KeyEnter))
	if m.speedIdx != defaultSpeedIdx {
		t.Fatalf("speed starts at %d", m.speedIdx)
	}
	for i := 0; i < 10; i++ {
		m, _ = send(m, typed('+'))
	}
	if m.speedIdx != len(speedMultipliers)-1 {
		t.Errorf("speed climbed past the top: %d", m.speedIdx)
	}
	for i := 0; i < 10; i++ {
		m, _ = send(m, typed('-'))
	}
	if m.speedIdx != 0 {
		t.Errorf("speed fell below the bottom: %d", m.speedIdx)
	}
	// Every speed level must still schedule a tick; a division that produced a
	// zero interval would make tea.Tick fire in a hot loop.
	for m.speedIdx = 0; m.speedIdx < len(speedMultipliers); m.speedIdx++ {
		if m.tickCmd() == nil {
			t.Errorf("speed %s scheduled no tick", speedLabels[m.speedIdx])
		}
	}
	m.speedIdx = defaultSpeedIdx
	m, _ = send(m, key(tea.KeyRight))
	if m.animCursor != 1 {
		t.Errorf("right moved to %d", m.animCursor)
	}
	m, _ = send(m, key(tea.KeyLeft))
	if m.animCursor != 0 {
		t.Errorf("left moved to %d", m.animCursor)
	}
}

func TestSourceToggle(t *testing.T) {
	m := sized(t)
	m, _ = send(m, key(tea.KeyEnter))
	before := m.animView()
	m, _ = send(m, typed('s'))
	if !m.showSource {
		t.Fatal("s did not turn the source panel on")
	}
	after := m.animView()
	if after == before {
		t.Error("the source panel changed nothing in the view")
	}
	m, _ = send(m, typed('s'))
	if m.showSource {
		t.Error("s did not turn the source panel off")
	}
}
