package splash

import (
	"testing"
)

func TestTypingFrames(t *testing.T) {
	frames := TypingFrames("Hello")
	if len(frames) == 0 {
		t.Error("expected non-empty frames")
	}
	// first frame should be cursor only
	if frames[0] != "█" {
		t.Errorf("first frame should be cursor, got %q", frames[0])
	}
}

func TestTypingFramesEmpty(t *testing.T) {
	frames := TypingFrames("")
	if len(frames) == 0 {
		t.Error("expected frames even for empty input")
	}
}

func TestExpandBorderFrames(t *testing.T) {
	frames := ExpandBorderFrames()
	if len(frames) == 0 {
		t.Error("expected non-empty frames")
	}
	if len(frames) < 5 {
		t.Errorf("expected at least 5 frames, got %d", len(frames))
	}
}

func TestFadeInFrames(t *testing.T) {
	frames := FadeInFrames()
	if len(frames) == 0 {
		t.Error("expected non-empty frames")
	}
	if len(frames) < 4 {
		t.Errorf("expected at least 4 frames, got %d", len(frames))
	}
}

func TestSpinnerSplashFrames(t *testing.T) {
	frames := SpinnerSplashFrames()
	if len(frames) == 0 {
		t.Error("expected non-empty frames")
	}
	if len(frames) < 10 {
		t.Errorf("expected at least 10 frames, got %d", len(frames))
	}
}

func TestGlitchFrames(t *testing.T) {
	frames := GlitchFrames()
	if len(frames) == 0 {
		t.Error("expected non-empty frames")
	}
	// last few frames should be clean text
	last := frames[len(frames)-1]
	if last != "ASCII ANIMATIONS" {
		t.Errorf("last frame should be clean text, got %q", last)
	}
}

func TestScanLineFrames(t *testing.T) {
	frames := ScanLineFrames()
	if len(frames) == 0 {
		t.Error("expected non-empty frames")
	}
	if len(frames) < 10 {
		t.Errorf("expected at least 10 frames, got %d", len(frames))
	}
}

func TestAllSplashFramesNonEmpty(t *testing.T) {
	allFrameSets := []struct {
		name   string
		frames []string
	}{
		{"Typing", TypingFrames("Test")},
		{"ExpandBorder", ExpandBorderFrames()},
		{"FadeIn", FadeInFrames()},
		{"SpinnerSplash", SpinnerSplashFrames()},
		{"Glitch", GlitchFrames()},
		{"ScanLine", ScanLineFrames()},
	}
	for _, fs := range allFrameSets {
		t.Run(fs.name, func(t *testing.T) {
			for i, f := range fs.frames {
				if f == "" {
					t.Errorf("frame %d is empty", i)
				}
			}
		})
	}
}
