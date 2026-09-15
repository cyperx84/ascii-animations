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
	stale := TickMsg{ID: m.ID()}
	m.Restart()
	m, cmd := m.Update(stale)
	if cmd != nil || m.Run().Tick() != 0 {
		t.Fatal("stale tick from before Restart was applied")
	}
	fresh := TickMsg{ID: m.ID(), gen: m.gen}
	m.Loop = true
	for i := 0; i < m.Run().Frames()+2; i++ {
		m, cmd = m.Update(fresh)
		if cmd == nil {
			t.Fatal("looping model stopped ticking")
		}
		if m.Done() {
			t.Fatal("looping model reported done")
		}
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
