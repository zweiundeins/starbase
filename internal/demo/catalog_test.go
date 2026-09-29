package demo

import (
	"strings"
	"testing"
)

func TestCatalog(t *testing.T) {
	if StarAt(41) != StarAt(41) || StarAt(0).ID != 1 || StarAt(CatalogSize-1).ID != CatalogSize {
		t.Errorf("stars: %+v %+v", StarAt(0), StarAt(CatalogSize-1))
	}
	sum := 0
	for _, s := range spectral {
		sum += s.weight
	}
	if sum != spectralWeight {
		t.Errorf("the spectral weights add up to %d, not %d", sum, spectralWeight)
	}
	classes := map[byte]int{}
	for i := range 10000 {
		s := StarAt(i)
		if len(s.Name) < 3 || s.Constellation == "" || s.Distance < 4.2 || s.Distance > 60000 || s.Magnitude < -20 || s.Magnitude > 32 || !strings.Contains("OBAFGKM", s.Class[:1]) {
			t.Fatalf("star %d: %+v", i, s)
		}
		classes[s.Class[0]]++
	}
	if len(classes) != 7 || classes['M'] < classes['G'] {
		t.Errorf("spectral classes: %v", classes)
	}
	// The list endpoint makes thousands per request: only the name and class allocate.
	if n := testing.AllocsPerRun(100, func() { StarAt(123456) }); n > 2 {
		t.Errorf("StarAt allocates %v times", n)
	}
	if StarsVersion() != StarsVersion() {
		t.Error("the version must be stable")
	}
	shades := map[int]int{}
	for y := range 100 {
		for x := range 48 {
			shades[Pixel(x, y)]++
		}
	}
	if len(shades) != 6 || Pixel(3, 7) != Pixel(3, 7) {
		t.Errorf("pixel shades: %v", shades)
	}
}
