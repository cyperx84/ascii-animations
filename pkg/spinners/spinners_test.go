package spinners

import (
	"testing"
)

func TestAllReturnsSpinners(t *testing.T) {
	all := All()
	if len(all) < 25 {
		t.Errorf("expected at least 25 spinners, got %d", len(all))
	}
}

func TestSpinnersHaveValidData(t *testing.T) {
	for _, s := range All() {
		t.Run(s.Name, func(t *testing.T) {
			if s.Name == "" {
				t.Error("spinner has empty name")
			}
			if s.Desc == "" {
				t.Errorf("spinner %q has empty description", s.Name)
			}
			if s.Category == "" {
				t.Errorf("spinner %q has empty category", s.Name)
			}
			if len(s.Frames) == 0 {
				t.Errorf("spinner %q has no frames", s.Name)
			}
			for i, f := range s.Frames {
				if f == "" {
					t.Errorf("spinner %q has empty frame at index %d", s.Name, i)
				}
			}
		})
	}
}

func TestUniqueNames(t *testing.T) {
	seen := make(map[string]bool)
	for _, s := range All() {
		if seen[s.Name] {
			t.Errorf("duplicate spinner name: %q", s.Name)
		}
		seen[s.Name] = true
	}
}

func TestCategories(t *testing.T) {
	cats := Categories()
	if len(cats) == 0 {
		t.Error("no categories returned")
	}
	// every spinner should have a valid category
	catSet := make(map[string]bool)
	for _, c := range cats {
		catSet[c] = true
	}
	for _, s := range All() {
		if !catSet[s.Category] {
			t.Errorf("spinner %q has category %q not in Categories()", s.Name, s.Category)
		}
	}
}

func TestInterval(t *testing.T) {
	if Interval <= 0 {
		t.Error("interval should be positive")
	}
}
