package tint

import "fmt"

// Dither selects the ordered-dither pattern applied before palette
// quantisation.
//
// The point of dithering is not to make each cell more accurate; it is to
// make each *region* more accurate. Ordered dithering picks between the two
// nearest palette entries so that the fraction of cells taking the further
// entry matches how far the true colour sits along the line between them.
// The local average, which is what the eye integrates, lands close to the
// true colour even where the palette has no entry near it.
//
// Bayer matrices make that fraction a fixed spatial pattern, so the same
// colour at the same cell always quantises the same way and a still region
// never crawls. Error diffusion (Floyd-Steinberg and friends) is deliberately
// not offered: it pushes each cell's error into its neighbours, so the result
// depends on scan order and shifts whenever a neighbouring cell changes.
type Dither uint8

const (
	// NoDither picks the single nearest palette entry per cell.
	NoDither Dither = iota
	// Bayer4 uses a 4x4 ordered matrix (16 thresholds), the cheapest option.
	Bayer4
	// Bayer8 uses an 8x8 ordered matrix (64 thresholds); the default.
	Bayer8
)

func (d Dither) String() string {
	switch d {
	case Bayer4:
		return "bayer4"
	case Bayer8:
		return "bayer8"
	}
	return "none"
}

// DitherNames lists the patterns in increasing cost order.
func DitherNames() []string { return []string{"none", "bayer4", "bayer8"} }

// ParseDither accepts none, bayer4 or bayer8.
func ParseDither(s string) (Dither, error) {
	switch s {
	case "", "none", "off", "no", "0", "false":
		return NoDither, nil
	case "bayer", "bayer4", "bayer-4", "4":
		return Bayer4, nil
	case "bayer8", "bayer-8", "8":
		return Bayer8, nil
	}
	return NoDither, fmt.Errorf("unknown dither %q: use none, bayer4 or bayer8", s)
}

// bayer4 is the classic 4x4 ordered-dither matrix.
var bayer4 = [4][4]uint8{
	{0, 8, 2, 10},
	{12, 4, 14, 6},
	{3, 11, 1, 9},
	{15, 7, 13, 5},
}

// bayer8 is the classic 8x8 ordered-dither matrix.
var bayer8 = [8][8]uint8{
	{0, 32, 8, 40, 2, 34, 10, 42},
	{48, 16, 56, 24, 50, 18, 58, 26},
	{12, 44, 4, 36, 14, 46, 6, 38},
	{60, 28, 52, 20, 62, 30, 54, 22},
	{3, 35, 11, 43, 1, 33, 9, 41},
	{51, 19, 59, 27, 49, 17, 57, 25},
	{15, 47, 7, 39, 13, 45, 5, 37},
	{63, 31, 55, 23, 61, 29, 53, 21},
}

// bucket is the dither matrix cell for (x,y): 0-15 for Bayer4, 0-63 for
// Bayer8. It doubles as the per-position cache key component.
func (d Dither) bucket(x, y int) int {
	switch d {
	case Bayer4:
		return int(bayer4[mod(y, 4)][mod(x, 4)])
	case Bayer8:
		return int(bayer8[mod(y, 8)][mod(x, 8)])
	}
	return 0
}

// threshold returns the dither threshold for cell (x,y) in (0,1). A cell
// takes the further palette entry when the threshold is below the true
// colour's position between the two entries.
func (d Dither) threshold(x, y int) float64 {
	n := 16.0
	if d == Bayer8 {
		n = 64.0
	}
	return (float64(d.bucket(x, y)) + 0.5) / n
}

// mod is a modulo that stays non-negative for negative coordinates.
func mod(v, n int) int {
	r := v % n
	if r < 0 {
		r += n
	}
	return r
}
