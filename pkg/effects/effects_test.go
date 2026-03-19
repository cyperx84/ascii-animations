package effects

import (
	"testing"
)

func TestRenderMatrixDimensions(t *testing.T) {
	tests := []struct {
		name   string
		w, h   int
		empty  bool
	}{
		{"zero", 0, 0, true},
		{"too small", 1, 1, true},
		{"minimal", 2, 2, false},
		{"normal", 40, 20, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := RenderMatrix(tt.w, tt.h, 0)
			if tt.empty && result != "" {
				t.Error("expected empty result")
			}
			if !tt.empty && result == "" {
				t.Error("expected non-empty result")
			}
		})
	}
}

func TestRenderFireDimensions(t *testing.T) {
	if RenderFire(0, 0, 0) != "" {
		t.Error("expected empty for zero dimensions")
	}
	if RenderFire(20, 10, 5) == "" {
		t.Error("expected non-empty result")
	}
}

func TestRenderRainDimensions(t *testing.T) {
	if RenderRain(1, 1, 0) != "" {
		t.Error("expected empty for too-small dimensions")
	}
	if RenderRain(30, 15, 3) == "" {
		t.Error("expected non-empty result")
	}
}

func TestRenderStarfieldDimensions(t *testing.T) {
	if RenderStarfield(2, 2, 0) != "" {
		t.Error("expected empty for too-small dimensions")
	}
	if RenderStarfield(40, 20, 10) == "" {
		t.Error("expected non-empty result")
	}
}

func TestRenderSnowDimensions(t *testing.T) {
	if RenderSnow(1, 1, 0) != "" {
		t.Error("expected empty for too-small dimensions")
	}
	if RenderSnow(30, 15, 5) == "" {
		t.Error("expected non-empty result")
	}
}

func TestRenderDNADimensions(t *testing.T) {
	if RenderDNA(5, 2, 0) != "" {
		t.Error("expected empty for too-small dimensions")
	}
	if RenderDNA(40, 20, 5) == "" {
		t.Error("expected non-empty result")
	}
}

func TestRenderWaveDimensions(t *testing.T) {
	if RenderWave(2, 2, 0) != "" {
		t.Error("expected empty for too-small dimensions")
	}
	if RenderWave(40, 20, 5) == "" {
		t.Error("expected non-empty result")
	}
}

func TestRenderPlasmaDimensions(t *testing.T) {
	if RenderPlasma(2, 2, 0) != "" {
		t.Error("expected empty for too-small dimensions")
	}
	if RenderPlasma(30, 15, 5) == "" {
		t.Error("expected non-empty result")
	}
}

func TestEffectsMultipleFrames(t *testing.T) {
	// ensure effects produce different output across frames
	renderers := []struct {
		name string
		fn   func(w, h, frame int) string
	}{
		{"Matrix", RenderMatrix},
		{"Fire", RenderFire},
		{"Starfield", RenderStarfield},
		{"Snow", RenderSnow},
		{"DNA", RenderDNA},
		{"Wave", RenderWave},
		{"Plasma", RenderPlasma},
	}
	for _, r := range renderers {
		t.Run(r.name, func(t *testing.T) {
			f0 := r.fn(40, 20, 0)
			f5 := r.fn(40, 20, 5)
			if f0 == f5 {
				t.Error("frames 0 and 5 should differ")
			}
		})
	}
}
