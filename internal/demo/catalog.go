package demo

import (
	"fmt"
	"math"
	"slices"
)

// CatalogSize is the number of stars in the catalog: each is computed from
// its index, so a list of a million needs no storage.
const CatalogSize = 1_000_000

// CatalogStar is one line of the star catalog.
type CatalogStar struct {
	Name          string // "SB 0042137"
	Class         string // spectral class, e.g. "G2V"
	Constellation string
	Magnitude     float64 // apparent brightness: lower is brighter
	Distance      int     // light years
}

var constellations = func() []string {
	var out []string
	for _, s := range brightStars {
		if !slices.Contains(out, s[1]) {
			out = append(out, s[1])
		}
	}
	return out
}()

// mix is splitmix64: well spread bits from an index.
func mix(x uint64) uint64 {
	x += 0x9E3779B97F4A7C15
	x = (x ^ x>>30) * 0xBF58476D1CE4E5B9
	x = (x ^ x>>27) * 0x94D049BB133111EB
	return x ^ x>>31
}

// unit is a number in [0, 1) from h's bits starting at shift.
func unit(h uint64, shift uint) float64 { return float64(h>>shift&0xFFFF) / 0x10000 }

// Star is the catalog's star i (0 ≤ i < CatalogSize), the same on every call.
func Star(i int) CatalogStar {
	h := mix(uint64(i))
	// Spectral classes roughly as common as among nearby stars: mostly K and M.
	const classes = "OBBBAAAAAAFFFFFFFFFFGGGGGGGGGGGGGGKKKKKKKKKKKKKKKKKKKKKKKKKKMMMMMMMMMMMMMMMMMMMMMMMMMMMMMMMMMMMMMMMM"
	lum := "V"
	if u := unit(h, 16); u > 0.97 {
		lum = "I"
	} else if u > 0.82 {
		lum = "III"
	}
	return CatalogStar{
		Name:          fmt.Sprintf("SB %07d", i+1),
		Class:         fmt.Sprintf("%c%d%s", classes[h%uint64(len(classes))], h>>8%10, lum),
		Constellation: constellations[h>>12%uint64(len(constellations))],
		Magnitude:     math.Round((2+11*math.Pow(unit(h, 32), 0.4))*100) / 100,
		Distance:      int(4 * math.Pow(5000, unit(h, 48))),
	}
}

// Pixel is the shade (0 to 5) of the pixel at x, y of an endless nebula:
// drifting bands of gas, and here and there a star.
func Pixel(x, y int) int {
	h := mix(uint64(y)<<32 | uint64(uint32(x)))
	if h%89 == 0 {
		return 5
	}
	fx, fy := float64(x), float64(y)
	v := math.Sin(fx/7+fy/19) + math.Sin(fy/11-fx/13) + math.Sin((fx+fy)/29) // -3 to 3
	v += (unit(h, 16) - 0.5) * 0.9                                           // dither
	return max(0, min(4, int((v+3)/6*5)))
}
