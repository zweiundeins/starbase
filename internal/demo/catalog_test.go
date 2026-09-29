package demo

import "testing"

func TestCatalog(t *testing.T) {
	if Star(41) != Star(41) || Star(0).Name != "SB 0000001" || Star(CatalogSize-1).Name != "SB 1000000" {
		t.Errorf("stars: %+v %+v", Star(0), Star(CatalogSize-1))
	}
	classes := map[byte]int{}
	for i := range 10000 {
		s := Star(i)
		if s.Constellation == "" || s.Magnitude < 2 || s.Magnitude > 13 || s.Distance < 4 || s.Distance > 20000 {
			t.Fatalf("star %d: %+v", i, s)
		}
		classes[s.Class[0]]++
	}
	if classes['M'] < classes['G'] || classes['O'] == 0 {
		t.Errorf("spectral classes: %v", classes)
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
