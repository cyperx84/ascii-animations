package term

import (
	"context"
	"errors"
	"math/rand/v2"
	"os"
	"regexp"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/cyperx84/ascii-animations/asciifx/fx"
	"github.com/cyperx84/ascii-animations/asciifx/tint"
)

// digits draws the tick number, so the byte stream shows which frames were
// actually put on screen.
type digits struct{ d float64 }

func (e digits) Duration() float64 { return e.d }
func (e digits) Step(f *fx.Frame) {
	f.Buf.WriteString(0, 0, strconv.Itoa(f.Tick%10), tint.RGB(200, 200, 200))
}

func digitRun(t *testing.T, fps int, dur float64) *fx.Run {
	t.Helper()
	spec := &fx.Spec{
		Name: "digits", Kind: fx.Transition, FPS: fps, DefW: 1, DefH: 1,
		New: func(fx.Values, int, int, *rand.Rand) (fx.Effect, error) { return digits{dur}, nil },
	}
	r, err := fx.NewRun(spec, fx.Options{})
	if err != nil {
		t.Fatal(err)
	}
	return r
}

func playToFile(t *testing.T, r *fx.Run, o PlayOptions) (string, time.Duration, error) {
	t.Helper()
	f, err := os.CreateTemp(t.TempDir(), "play")
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	o.Out = f
	o.Caps = Caps{Profile: TrueColor, Animate: true}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	start := time.Now()
	err = Play(ctx, r, o)
	elapsed := time.Since(start)
	if errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("Play did not finish on its own within 3s")
	}
	b, _ := os.ReadFile(f.Name())
	return string(b), elapsed, err
}

var escapes = regexp.MustCompile(`\x1b\[[0-9;?]*[A-Za-z]|\r`)

func TestLoopHonoursLimit(t *testing.T) {
	r := digitRun(t, 20, 0.1) // 3 frames per cycle, 0.1 s
	f, err := os.CreateTemp(t.TempDir(), "limit")
	if err != nil {
		t.Fatal(err)
	}
	done := make(chan error, 1)
	go func() {
		done <- Play(context.Background(), r, PlayOptions{
			Out: f, Caps: Caps{Profile: TrueColor, Animate: true},
			Loop: true, Limit: 400 * time.Millisecond,
		})
	}()
	select {
	case err := <-done:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("looping playback ignored Limit")
	}
}

func TestLoopDrawsFrameZeroAtBoundary(t *testing.T) {
	r := digitRun(t, 20, 0.1)
	out, _, err := playToFile(t, r, PlayOptions{Loop: true, Limit: 700 * time.Millisecond})
	if err != nil {
		t.Fatal(err)
	}
	shown := escapes.ReplaceAllString(out, "")
	if !strings.Contains(shown, "0120") {
		t.Fatalf("frame 0 not drawn when the loop restarted; frames shown: %q", shown)
	}
	if strings.Contains(shown, "21") {
		t.Fatalf("last frame stayed up into the next cycle; frames shown: %q", shown)
	}
}

func TestPlayRestoresTerminal(t *testing.T) {
	out, _, err := playToFile(t, digitRun(t, 30, 0.1), PlayOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(out, "\x1b[?25l\x1b[?1049h") {
		t.Fatalf("setup sequence missing: %q", out[:min(len(out), 20)])
	}
	if !strings.HasSuffix(out, "\x1b[?1049l\x1b[?25h") {
		t.Fatalf("terminal not restored: %q", out[max(0, len(out)-30):])
	}
	inline, _, _ := playToFile(t, digitRun(t, 30, 0.1), PlayOptions{Inline: true})
	if strings.Contains(inline, "1049") || !strings.HasSuffix(inline, "\x1b[?25h") {
		t.Fatalf("inline mode touched alt screen or left cursor hidden: %q", inline)
	}
}

func TestStaticWhenNotAnimating(t *testing.T) {
	f, _ := os.CreateTemp(t.TempDir(), "static")
	defer f.Close()
	r := digitRun(t, 20, 0.1)
	if err := Play(context.Background(), r, PlayOptions{Out: f, Caps: Caps{Animate: false}}); err != nil {
		t.Fatal(err)
	}
	b, _ := os.ReadFile(f.Name())
	if string(b) != "2\n" {
		t.Fatalf("static output should be the final frame only, got %q", b)
	}
}

func TestWatchKeysStopsOnCancelWithoutStealingInput(t *testing.T) {
	if !canWatchKeys {
		t.Skip("no key watcher on this platform")
	}
	rd, wr, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	defer rd.Close()
	defer wr.Close()
	ctx, cancel := context.WithCancel(context.Background())
	keys := make(chan struct{}, 1)
	done := watchKeys(ctx, int(rd.Fd()), keys)

	wr.Write([]byte("x")) // not a quit key: watcher must keep going
	time.Sleep(60 * time.Millisecond)
	select {
	case <-done:
		t.Fatal("watcher exited on a non-quit key")
	case <-keys:
		t.Fatal("non-quit key signalled quit")
	default:
	}

	cancel()
	select {
	case <-done:
	case <-time.After(500 * time.Millisecond):
		t.Fatal("watcher did not exit after cancel")
	}

	// Input arriving after the watcher stopped belongs to the host.
	wr.Write([]byte("hello"))
	buf := make([]byte, 16)
	n, _ := rd.Read(buf)
	if string(buf[:n]) != "hello" {
		t.Fatalf("host read %q after watcher exit", buf[:n])
	}
}

func TestWatchKeysSignalsQuit(t *testing.T) {
	if !canWatchKeys {
		t.Skip("no key watcher on this platform")
	}
	rd, wr, _ := os.Pipe()
	defer rd.Close()
	defer wr.Close()
	keys := make(chan struct{}, 1)
	done := watchKeys(context.Background(), int(rd.Fd()), keys)
	wr.Write([]byte("abq"))
	select {
	case <-keys:
	case <-time.After(500 * time.Millisecond):
		t.Fatal("q did not signal")
	}
	<-done
}
