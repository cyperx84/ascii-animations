package term

import (
	"bytes"
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
	got := query(inR, &out, time.Second, inR.SetReadDeadline)
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
	got := query(inR, &out, 60*time.Millisecond, inR.SetReadDeadline)
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
	got := query(inR, &out, 60*time.Millisecond, inR.SetReadDeadline)
	if parseSync(got) != 0 {
		t.Fatalf("parseSync(%q) = %d, want 0 (explicitly unsupported)", got, parseSync(got))
	}
}

func TestQueryWithoutDeadlineSupportSkips(t *testing.T) {
	// A descriptor that cannot take a deadline must not be read at all,
	// because the read could block forever.
	inR, inW, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	defer inR.Close()
	defer inW.Close()
	var out bytes.Buffer
	fail := func(time.Time) error { return os.ErrNoDeadline }
	if got := query(inR, &out, time.Second, fail); got != "" {
		t.Fatalf("read %q from a descriptor with no deadline support", got)
	}
	if out.String() != capQueries {
		t.Fatal("the probe should still have been written")
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
