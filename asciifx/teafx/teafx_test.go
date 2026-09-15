package teafx

import (
	"testing"

	tea "charm.land/bubbletea/v2"

	_ "github.com/cyperx84/ascii-animations/asciifx/effects"
	"github.com/cyperx84/ascii-animations/asciifx/fx"
	"github.com/cyperx84/ascii-animations/asciifx/tint"
)

func TestTicksAdvanceUntilDone(t *testing.T) {
	m, err := New("reveal", fx.Options{W: 12, H: 3, Seed: 1, Content: fx.Text("HELLO", tint.None)})
	if err != nil {
		t.Fatal(err)
	}
	if m.Init() == nil {
		t.Fatal("Init should start ticking")
	}
	first := m.View()
	frames := m.Run().Frames()
	if frames < 3 {
		t.Fatalf("reveal has %d frames", frames)
	}
	changed := false
	var cmd tea.Cmd
	for i := 1; i < frames; i++ {
		if m.Done() {
			t.Fatalf("done early at tick %d of %d", i, frames)
		}
		m, cmd = m.Update(TickMsg{ID: m.ID()})
		if m.View() != first {
			changed = true
		}
		if i < frames-1 && cmd == nil {
			t.Fatalf("tick %d returned no follow-up tick", i)
		}
	}
	if !changed {
		t.Fatal("View never changed")
	}
	if !m.Done() {
		t.Fatalf("not done after %d frames (tick %d)", frames, m.Run().Tick())
	}
	if cmd != nil {
		t.Fatal("finished effect should stop ticking")
	}
	last := m.View()
	m, cmd = m.Update(TickMsg{ID: m.ID()})
	if cmd != nil || m.View() != last {
		t.Fatal("ticks after Done should be no-ops")
	}

	// Output matches the headless renderer for the same options.
	want, _, err := fx.Render("reveal", fx.Options{W: 12, H: 3, Seed: 1, Content: fx.Text("HELLO", tint.None)}, frames-1)
	if err != nil {
		t.Fatal(err)
	}
	if got := m.buf.Plain(); got != want.Plain() {
		t.Fatalf("final frame differs from fx.Render:\n%s\nvs\n%s", got, want.Plain())
	}
}

func TestInstancesIgnoreForeignTicks(t *testing.T) {
	a, err := New("fire", fx.Options{W: 10, H: 4})
	if err != nil {
		t.Fatal(err)
	}
	b, err := New("fire", fx.Options{W: 10, H: 4})
	if err != nil {
		t.Fatal(err)
	}
	if a.ID() == b.ID() {
		t.Fatal("instances share an id")
	}
	before := a.Run().Tick()
	a, cmd := a.Update(TickMsg{ID: b.ID()})
	if cmd != nil || a.Run().Tick() != before {
		t.Fatal("foreign tick advanced the model")
	}
	a, _ = a.Update(TickMsg{ID: a.ID()})
	if a.Run().Tick() != before+1 {
		t.Fatal("own tick did not advance")
	}
}

func TestRestartDropsStaleTicksAndLoop(t *testing.T) {
	m, err := New("reveal", fx.Options{W: 8, H: 3, Content: fx.Text("HI", tint.None), Params: map[string]string{"duration": "0.1"}})
	if err != nil {
		t.Fatal(err)
	}
	// Advance once so the model is on a tag a restart can supersede; a tag of 0
	// is a wildcard and would be accepted whenever it arrived.
	m, _ = m.Update(TickMsg{ID: m.ID()})
	stale := TickMsg{ID: m.ID(), tag: m.tag}
	m.Restart()
	m, cmd := m.Update(stale)
	if cmd != nil || m.Run().Tick() != 0 {
		t.Fatal("stale tick from before Restart was applied")
	}
	// Drive the live chain by always presenting its current tag.
	m.Loop = true
	for i := 0; i < m.Run().Frames()+2; i++ {
		m, cmd = m.Update(TickMsg{ID: m.ID(), tag: m.tag})
		if cmd == nil {
			t.Fatal("looping model stopped ticking")
		}
		if m.Done() {
			t.Fatal("looping model reported done")
		}
	}
}

// TestDuplicateTicksDoNotDoubleTheRate is the bug this guards. Tick called
// twice from the same state used to start two chains that both stayed live,
// because the tag they carried never changed, so the effect advanced twice per
// interval for as long as it ran.
func TestDuplicateTicksDoNotDoubleTheRate(t *testing.T) {
	m, err := New("reveal", fx.Options{
		W: 8, H: 3, Seed: 1, Content: fx.Text("HI", tint.None),
		Params: map[string]string{"duration": "0.4"},
	})
	if err != nil {
		t.Fatal(err)
	}

	// Two chains started from the same state, which is what a parent does when
	// it returns Tick from Init and again from a restart path.
	msg1, ok := m.Init()().(TickMsg)
	if !ok {
		t.Fatal("Init did not produce a TickMsg")
	}
	msg2, ok := m.Init()().(TickMsg)
	if !ok {
		t.Fatal("a second Tick did not produce a TickMsg")
	}

	m, cmdA := m.Update(msg1)
	m, cmdB := m.Update(msg2)
	if cmdA == nil || cmdB == nil {
		t.Fatal("an accepted tick must schedule its follow-up")
	}

	// Both carried the same tag, so both were accepted; from here the two
	// follow-ups carry different tags and the older chain is dead. That is the
	// whole fix: one extra frame slips through, not a permanently doubled rate.
	live := m.tag
	stale := live - 1
	if stale == 0 {
		t.Fatalf("expected a superseded chain, tag is %d", live)
	}

	before := m.Run().Tick()
	if got, cmd := m.Update(TickMsg{ID: m.ID(), tag: stale}); cmd != nil || got.Run().Tick() != before {
		t.Fatal("the superseded chain is still live, so the rate stays doubled")
	}
	got, cmd := m.Update(TickMsg{ID: m.ID(), tag: live})
	if cmd == nil {
		t.Fatal("the live chain stopped ticking")
	}
	if advanced := got.Run().Tick() - before; advanced != 1 {
		t.Fatalf("one live tick advanced %d frames, want 1", advanced)
	}
}

// TestTickStampsTheCurrentTag goes through the real command path, so a bug in
// tick itself — stamping a constant, or the wrong field — is caught. The
// hand-built messages above cannot see that, because they choose their own tags.
func TestTickStampsTheCurrentTag(t *testing.T) {
	m, err := New("fire", fx.Options{W: 8, H: 3})
	if err != nil {
		t.Fatal(err)
	}
	// Two chains started from the same state.
	c1, c2 := m.Init(), m.Init()
	m, cmdA := m.Update(c1().(TickMsg))
	m, cmdB := m.Update(c2().(TickMsg))
	if cmdA == nil || cmdB == nil {
		t.Fatal("an accepted tick must schedule its follow-up")
	}

	before := m.Run().Tick()
	if _, cmd := m.Update(cmdA().(TickMsg)); cmd != nil {
		t.Error("the superseded chain survived a real round trip, so the rate stays doubled")
	}
	got, cmd := m.Update(cmdB().(TickMsg))
	if cmd == nil {
		t.Fatal("the live chain stopped ticking")
	}
	if advanced := got.Run().Tick() - before; advanced != 1 {
		t.Errorf("the live chain advanced %d frames, want 1", advanced)
	}
}

// TestZeroTagIsAWildcard pins the affordance that makes hand-built messages
// usable in tests and in code that routes ticks itself, and that Restart relies
// on to supersede a live chain rather than deadlock it.
func TestZeroTagIsAWildcard(t *testing.T) {
	m, err := New("fire", fx.Options{W: 8, H: 3})
	if err != nil {
		t.Fatal(err)
	}
	for i := 1; i <= 3; i++ {
		var cmd tea.Cmd
		m, cmd = m.Update(TickMsg{ID: m.ID()})
		if cmd == nil {
			t.Fatalf("tick %d was dropped", i)
		}
		if m.Run().Tick() != i {
			t.Fatalf("after %d wildcard ticks the frame is %d", i, m.Run().Tick())
		}
	}
	// A stale non-zero tag is still dropped even after wildcards.
	before := m.Run().Tick()
	if got, cmd := m.Update(TickMsg{ID: m.ID(), tag: m.tag - 1}); cmd != nil || got.Run().Tick() != before {
		t.Fatal("a stale tag was accepted")
	}
}

func TestFitResizes(t *testing.T) {
	m, err := New("fire", fx.Options{W: 10, H: 4})
	if err != nil {
		t.Fatal(err)
	}
	m, _ = m.Update(tea.WindowSizeMsg{Width: 30, Height: 6})
	if w, h := m.Run().Size(); w != 10 || h != 4 {
		t.Fatal("resized without Fit")
	}
	m.Fit = true
	m, _ = m.Update(tea.WindowSizeMsg{Width: 30, Height: 6})
	if w, h := m.Run().Size(); w != 30 || h != 6 {
		t.Fatalf("Fit did not resize: %dx%d", w, h)
	}
	if m.buf.W != 30 {
		t.Fatal("view buffer not updated after resize")
	}
}
