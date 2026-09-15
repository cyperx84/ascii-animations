package cell

import "testing"

func TestWidthClassification(t *testing.T) {
	for r, want := range map[rune]int{
		'a': 1, '█': 1, '▀': 1, '⠂': 1, '─': 1, '╳': 1, '🯀': 1,
		'·': -1, '×': -1, '←': -1, '★': -1, '😀': -1,
		'漢': 2, '\t': 0,
	} {
		if got := Width(r); got != want {
			t.Errorf("Width(%q U+%04X) = %d, want %d", r, r, got, want)
		}
	}
}
