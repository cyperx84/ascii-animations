package lint_test

import (
	"strings"
	"testing"

	"github.com/cyperx84/ascii-animations/asciifx/fx"
	"github.com/cyperx84/ascii-animations/asciifx/lint"
)

// The test a consumer writes: whatever I generate, it still redraws the same
// in every terminal.
func TestBannersPassTheirOwnLinter(t *testing.T) {
	for _, font := range fx.BannerFonts() {
		art, err := fx.Banner("ASCIIFX 2026", font)
		if err != nil {
			t.Fatalf("%s: %v", font, err)
		}
		for _, is := range lint.String(art) {
			t.Errorf("%s:%d:%d: %s", font, is.Line, is.Col, is.Message)
		}
	}
}

func TestStringFindsWhatTheCLIFinds(t *testing.T) {
	// One wide rune and one short line: the two failures art hits most.
	issues := lint.String("ab\u6f22\nxy")
	kinds := map[string]bool{}
	for _, is := range issues {
		kinds[is.Kind] = true
	}
	for _, want := range []string{"wide", "ragged"} {
		if !kinds[want] {
			t.Errorf("no %q issue in %+v", want, issues)
		}
	}
}

func TestCleanArtHasNoIssues(t *testing.T) {
	if got := lint.String("+--+\n|  |\n+--+"); len(got) != 0 {
		t.Fatalf("clean art flagged: %+v", got)
	}
}

func TestStringsChecksASetAsOneAnimation(t *testing.T) {
	// Frames of different widths are the classic spinner bug: the shorter
	// frame leaves the last cell of the longer one on screen.
	issues := lint.Strings("dots", []string{"..", "..."})
	found := false
	for _, is := range issues {
		if is.Kind == "frame-width" && is.Set == "dots" {
			found = true
		}
	}
	if !found {
		t.Fatalf("frames of different widths not reported: %+v", issues)
	}
}

func TestSplitReadsSpinnerJSONAndTextFrames(t *testing.T) {
	frames, err := lint.Split([]byte(`{"interval":80,"frames":["-","\\","|","/"]}`), "---")
	if err != nil {
		t.Fatal(err)
	}
	if len(frames) != 4 {
		t.Fatalf("got %d frames, want 4", len(frames))
	}
	frames, err = lint.Split([]byte("aa\nbb\n---\ncc\ndd\n"), "---")
	if err != nil {
		t.Fatal(err)
	}
	if len(frames) != 2 {
		t.Fatalf("got %d text frames, want 2", len(frames))
	}
	// Base points back into the file, so an issue in frame 1 is reportable
	// at the line it actually occupies.
	if frames[1].Base != 4 {
		t.Fatalf("second frame starts at file line %d, want 4", frames[1].Base)
	}
}

func TestSplitRejectsJSONWithoutFrames(t *testing.T) {
	_, err := lint.Split([]byte(`{"nope":1}`), "---")
	if err == nil || !strings.Contains(err.Error(), "cli-spinners") {
		t.Fatalf("got %v, want the cli-spinners shape error", err)
	}
}

func TestWidthCountsTheColumnsATerminalWouldUse(t *testing.T) {
	cases := map[string]int{"abc": 3, "\u6f22": 2, "\t": 8, "a\tb": 9}
	for in, want := range cases {
		if got := lint.Width(in); got != want {
			t.Errorf("Width(%q) = %d, want %d", in, got, want)
		}
	}
}
