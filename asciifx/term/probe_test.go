package term

import (
	"bytes"
	"context"
	"os"
	"strings"
	"testing"
	"time"
)

func TestParseSync(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want int
	}{
		{"set", "\x1b[?2026;1$y", 1},
		{"reset", "\x1b[?2026;2$y", 0},
		{"not recognised", "\x1b[?2026;0$y", -1},
		{"permanently set", "\x1b[?2026;3$y", 1},
		{"permanently reset", "\x1b[?2026;4$y", 0},
		{"other mode only", "\x1b[?2027;1$y\x1b[?62;4c", -1},
		{"da1 only", "\x1b[?62;4c", -1},
		{"silence", "", -1},
		{"garbage", "\x1b[?2026$p", -1},
		{"sync answer after other traffic", "\x1b[?25;1$y\x1b[?2026;2$y\x1b[?1;2c", 0},
	}
	for _, c := range cases {
		if got := parseSync(c.in); got != c.want {
			t.Errorf("%s: parseSync(%q) = %d, want %d", c.name, c.in, got, c.want)
		}
	}
}

func TestQuerySendsTheProbesAndStopsAtDA1(t *testing.T) {
	inR, inW, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	defer inR.Close()
	go func() {
		// A real terminal answers DECRQM first and DA1 last. Splitting the
		// writes proves the reader accumulates instead of trusting one Read.
		_, _ = inW.WriteString("\x1b[?2026;1$y")
		time.Sleep(20 * time.Millisecond)
		_, _ = inW.WriteString("\x1b[?62;4c")
		_ = inW.Close()
	}()
	var out bytes.Buffer
	got := query(int(inR.Fd()), &out, time.Second)
	if out.String() != capQueries {
		t.Errorf("sent %q, want %q", out.String(), capQueries)
	}
	if !strings.Contains(got, "\x1b[?2026;1$y") || !hasDA1(got) {
		t.Fatalf("did not collect the full reply: %q", got)
	}
	if parseSync(got) != 1 {
		t.Errorf("parseSync(%q) = %d, want 1", got, parseSync(got))
	}
}

func TestQueryGivesUpAtTheDeadline(t *testing.T) {
	inR, inW, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	defer inR.Close()
	defer inW.Close() // stays open: nothing ever arrives
	var out bytes.Buffer
	start := time.Now()
	got := query(int(inR.Fd()), &out, 60*time.Millisecond)
	elapsed := time.Since(start)
	if elapsed < 40*time.Millisecond {
		t.Fatalf("returned after %v, before the deadline", elapsed)
	}
	if elapsed > time.Second {
		t.Fatalf("returned after %v; the deadline did not bound the read", elapsed)
	}
	if got != "" {
		t.Fatalf("got %q from a silent terminal", got)
	}
}

func TestQueryKeepsAPartialReplyWithoutDA1(t *testing.T) {
	inR, inW, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	defer inR.Close()
	go func() {
		_, _ = inW.WriteString("\x1b[?2026;2$y")
	}()
	defer inW.Close()
	var out bytes.Buffer
	got := query(int(inR.Fd()), &out, 60*time.Millisecond)
	if parseSync(got) != 0 {
		t.Fatalf("parseSync(%q) = %d, want 0 (explicitly unsupported)", got, parseSync(got))
	}
}

// TestQueryStopsAtEndOfInput pins that a closed descriptor ends the read
// rather than spinning against a select that reports end of input as ready
// until the timeout runs out.
func TestQueryStopsAtEndOfInput(t *testing.T) {
	inR, inW, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	defer inR.Close()
	inW.WriteString("\x1b[?2026;2$y")
	inW.Close()
	var out bytes.Buffer
	start := time.Now()
	got := query(int(inR.Fd()), &out, 5*time.Second)
	if elapsed := time.Since(start); elapsed > time.Second {
		t.Fatalf("took %v to notice end of input; it should not wait out the timeout", elapsed)
	}
	if parseSync(got) != 0 {
		t.Fatalf("the bytes that did arrive were lost: %q", got)
	}
}

// TestQueryNeedsNoFileDeadline is the bug this replaced. os.File read
// deadlines are not available for terminal stdio -- the runtime poller only
// takes descriptors it opened non-blocking -- so a probe built on
// SetReadDeadline gave up before reading a byte. query must not depend on it.
//
// A pipe is pollable, which is exactly why the old tests passed while the
// real terminal did not; asserting on os.Stdin is what tells the two apart.
func TestQueryNeedsNoFileDeadline(t *testing.T) {
	if !canWaitReadable {
		t.Skip("no readiness check on this platform")
	}
	if err := os.Stdin.SetReadDeadline(time.Now().Add(time.Millisecond)); err == nil {
		_ = os.Stdin.SetReadDeadline(time.Time{})
		t.Skip("stdin here does take a deadline; the interesting case is a terminal")
	}
	// The same descriptor that cannot take a deadline can still be waited on.
	if _, err := waitReadable(int(os.Stdin.Fd()), time.Millisecond); err != nil {
		t.Fatalf("waitReadable on a descriptor with no deadline support: %v", err)
	}
}

// TestProbeRawReadsTheAnswer covers probeRaw itself, not just query: the
// reply has to travel all the way into a Caps.
func TestProbeRawReadsTheAnswer(t *testing.T) {
	for _, c := range []struct {
		name      string
		reply     string
		syncKnown bool
		noSync    bool
	}{
		{"supported", "\x1b[?2026;1$y\x1b[?62;1;c", true, false},
		{"unsupported", "\x1b[?2026;2$y\x1b[?62;1;c", true, true},
		{"mode not recognised", "\x1b[?2026;0$y\x1b[?62;1;c", false, false},
		{"da1 only", "\x1b[?62;1;c", false, false},
		{"silence", "", false, false},
		{"decrqm with no da1 behind it", "\x1b[?2026;2$y", true, true},
	} {
		t.Run(c.name, func(t *testing.T) {
			inR, inW, err := os.Pipe()
			if err != nil {
				t.Fatal(err)
			}
			defer inR.Close()
			if c.reply != "" {
				inW.WriteString(c.reply)
			}
			inW.Close()
			out, err := os.CreateTemp(t.TempDir(), "probe")
			if err != nil {
				t.Fatal(err)
			}
			defer out.Close()

			got := probeRaw(inR, out, 60*time.Millisecond)
			if got.SyncKnown != c.syncKnown || got.NoSync != c.noSync {
				t.Fatalf("SyncKnown=%v NoSync=%v, want %v/%v", got.SyncKnown, got.NoSync, c.syncKnown, c.noSync)
			}
			if !got.Animate {
				t.Error("probeRaw must not turn animation off")
			}
			b, _ := os.ReadFile(out.Name())
			if string(b) != capQueries {
				t.Errorf("wrote %q to the terminal, want the probes", b)
			}
		})
	}
}

func TestDa1EndIgnoresOtherCSISequences(t *testing.T) {
	if da1End("\x1b[?2026;1$y") != -1 {
		t.Fatal("a DECRPM reply is not a DA1 reply")
	}
	if da1End("\x1b[?1;2c") != 7 {
		t.Fatalf("da1End(%q) = %d, want 7", "\x1b[?1;2c", da1End("\x1b[?1;2c"))
	}
	if da1End("\x1b[?62;4;6;22c tail") != 13 {
		t.Fatalf("da1End with several attributes = %d, want 13", da1End("\x1b[?62;4;6;22c tail"))
	}
}

// TestMergeProbeOnlyEverDisables pins the merge rule Play applies to a probe
// result: the probe may disable synchronized output, never re-enable it.
func TestMergeProbeOnlyEverDisables(t *testing.T) {
	answered := func(noSync bool) Caps { return Caps{Animate: true, SyncKnown: true, NoSync: noSync} }
	for _, c := range []struct {
		name          string
		start, got    Caps
		noSync, known bool
	}{
		{"caller disabled it, terminal says supported",
			Caps{NoSync: true}, answered(false), true, true},
		{"caller disabled it, terminal says unsupported",
			Caps{NoSync: true}, answered(true), true, true},
		{"default, terminal says unsupported",
			Caps{}, answered(true), true, true},
		{"default, terminal says supported",
			Caps{}, answered(false), false, true},
		{"caller disabled it, terminal never answered",
			Caps{NoSync: true}, Caps{Animate: true}, true, false},
		{"default, terminal never answered",
			Caps{}, Caps{Animate: true}, false, false},
	} {
		t.Run(c.name, func(t *testing.T) {
			caps := c.start
			caps.mergeProbe(c.got)
			if caps.NoSync != c.noSync {
				t.Errorf("NoSync = %v, want %v", caps.NoSync, c.noSync)
			}
			if caps.SyncKnown != c.known {
				t.Errorf("SyncKnown = %v, want %v", caps.SyncKnown, c.known)
			}
		})
	}
}

// TestASCIIFXSyncStopsThePlayProbe pins the precedence above the merge rule:
// an explicit ASCIIFX_SYNC is recorded by Detect, and Play does not probe at
// all when it was given, in either direction.
func TestASCIIFXSyncStopsThePlayProbe(t *testing.T) {
	for _, c := range []struct {
		value  string
		noSync bool
	}{{"0", true}, {"1", false}, {"false", true}, {"", false}} {
		caps := detect(true, env("ASCIIFX_SYNC", c.value, "TERM", "xterm-256color"))
		if caps.NoSync != c.noSync {
			t.Errorf("ASCIIFX_SYNC=%q: NoSync = %v, want %v", c.value, caps.NoSync, c.noSync)
		}
		// syncSet is what Play consults before probing at all; an unset
		// variable leaves it false so the probe may still run.
		if want := c.value != ""; caps.syncSet != want {
			t.Errorf("ASCIIFX_SYNC=%q: syncSet = %v, want %v", c.value, caps.syncSet, want)
		}
		// Even reached, the merge could not undo an explicit disable.
		caps.mergeProbe(Caps{Animate: true, SyncKnown: true, NoSync: false})
		if caps.NoSync != c.noSync {
			t.Errorf("ASCIIFX_SYNC=%q: a supported answer changed NoSync to %v", c.value, caps.NoSync)
		}
	}
}

// TestPlayProbeKeepsACallerDisable is finding 5 end to end, through Play's own
// probe path. It needs a terminal on stdin that answers, so it is skipped by a
// normal `go test` run and exercised by the pty harness in the scratchpad:
//
//	go test -c -o /tmp/term.test ./asciifx/term
//	python3 ptyreply.py out.txt '\x1b[?2026;1$y\x1b[?62;1;c' \
//	  /tmp/term.test -test.run TestPlayProbeKeepsACallerDisable -test.v
func TestPlayProbeKeepsACallerDisable(t *testing.T) {
	if !isTerminal(os.Stdin) {
		t.Skip("needs a terminal on stdin that answers DECRQM; run under the pty harness")
	}
	f, err := os.CreateTemp(t.TempDir(), "probe")
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	err = Play(context.Background(), digitRun(t, 20, 0.2), PlayOptions{
		Out:   f,
		Caps:  Caps{Profile: TrueColor, Animate: true, NoSync: true},
		Probe: true,
		Limit: 300 * time.Millisecond,
	})
	if err != nil {
		t.Fatal(err)
	}
	b, _ := os.ReadFile(f.Name())
	if strings.Contains(string(b), syncStart) {
		t.Fatalf("the caller set NoSync, the terminal said mode 2026 was supported, and Play emitted %q anyway", syncStart)
	}
}
