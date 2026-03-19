package colors

import (
	"strings"
	"testing"
)

func TestRender16(t *testing.T) {
	result := Render16()
	if result == "" {
		t.Error("expected non-empty output")
	}
	if !strings.Contains(result, "ANSI") {
		t.Error("expected header text")
	}
}

func TestRender256(t *testing.T) {
	result := Render256()
	if result == "" {
		t.Error("expected non-empty output")
	}
	if !strings.Contains(result, "256") {
		t.Error("expected header text")
	}
}

func TestRenderTruecolorDimensions(t *testing.T) {
	if RenderTruecolor(2, 2, 0) != "" {
		t.Error("expected empty for too-small dimensions")
	}
	result := RenderTruecolor(40, 20, 0)
	if result == "" {
		t.Error("expected non-empty result")
	}
}

func TestRenderTruecolorAnimation(t *testing.T) {
	f0 := RenderTruecolor(40, 20, 0)
	f5 := RenderTruecolor(40, 20, 5)
	if f0 == f5 {
		t.Error("different frames should produce different output")
	}
}

func TestHsvToRGB(t *testing.T) {
	tests := []struct {
		h, s, v    float64
		r, g, b    int
	}{
		{0, 1, 1, 255, 0, 0},       // red
		{120, 1, 1, 0, 255, 0},     // green
		{240, 1, 1, 0, 0, 255},     // blue
		{0, 0, 0, 0, 0, 0},         // black
		{0, 0, 1, 255, 255, 255},   // white
	}
	for _, tt := range tests {
		r, g, b := hsvToRGB(tt.h, tt.s, tt.v)
		if r != tt.r || g != tt.g || b != tt.b {
			t.Errorf("hsvToRGB(%v,%v,%v) = (%d,%d,%d), want (%d,%d,%d)",
				tt.h, tt.s, tt.v, r, g, b, tt.r, tt.g, tt.b)
		}
	}
}
