package demo

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"math"
)

// CatalogSize is the number of stars in the catalog: each is computed from
// its index, so a list of a million needs no storage.
const CatalogSize = 1_000_000

// StarCount is how many of them SeedStars stores, for demos that sort: enough
// for a table that only works when the server sends the rows in view.
const StarCount = 100_000

// Star is a made-up star of the catalog.
type Star struct {
	ID            int     `json:"id"` // its index + 1
	Name          string  `json:"name"`
	Class         string  `json:"class"` // spectral class, e.g. "G2V"
	Temp          int     `json:"temp"`  // surface temperature in kelvin: the order of the classes
	Constellation string  `json:"constellation"`
	Distance      float64 `json:"distance"`  // light years
	Magnitude     float64 `json:"magnitude"` // apparent: how bright it looks from Earth, lower is brighter
	Planets       int     `json:"planets"`
}

// Constellations are the 88 of the IAU.
var Constellations = []string{
	"Andromeda", "Antlia", "Apus", "Aquarius", "Aquila", "Ara", "Aries", "Auriga",
	"Boötes", "Caelum", "Camelopardalis", "Cancer", "Canes Venatici", "Canis Major", "Canis Minor", "Capricornus",
	"Carina", "Cassiopeia", "Centaurus", "Cepheus", "Cetus", "Chamaeleon", "Circinus", "Columba",
	"Coma Berenices", "Corona Australis", "Corona Borealis", "Corvus", "Crater", "Crux", "Cygnus", "Delphinus",
	"Dorado", "Draco", "Equuleus", "Eridanus", "Fornax", "Gemini", "Grus", "Hercules",
	"Horologium", "Hydra", "Hydrus", "Indus", "Lacerta", "Leo", "Leo Minor", "Lepus",
	"Libra", "Lupus", "Lynx", "Lyra", "Mensa", "Microscopium", "Monoceros", "Musca",
	"Norma", "Octans", "Ophiuchus", "Orion", "Pavo", "Pegasus", "Perseus", "Phoenix",
	"Pictor", "Pisces", "Piscis Austrinus", "Puppis", "Pyxis", "Reticulum", "Sagitta", "Sagittarius",
	"Scorpius", "Sculptor", "Scutum", "Serpens", "Sextans", "Taurus", "Telescopium", "Triangulum",
	"Triangulum Australe", "Tucana", "Ursa Major", "Ursa Minor", "Vela", "Virgo", "Volans", "Vulpecula",
}

// Spectral types, hottest first: how common each is here (in the sky nearly
// every star is an M), and the temperature and absolute magnitude of a
// main-sequence star from subclass 0 to 9.
var spectral = []struct {
	letter        byte
	weight        int
	hot, cool     int
	bright, faint float64
}{
	{'O', 1, 50000, 30000, -5.5, -4},
	{'B', 5, 30000, 10000, -4, -0.5},
	{'A', 9, 10000, 7500, 0.5, 2.5},
	{'F', 13, 7500, 6000, 2.5, 4},
	{'G', 18, 6000, 5200, 4, 5.5},
	{'K', 22, 5200, 3700, 5.5, 8},
	{'M', 32, 3700, 2400, 8, 15},
}

const spectralWeight = 100 // the weights' sum

var (
	onsets  = []string{"Al", "Ar", "Be", "Ca", "Cor", "Da", "El", "Fen", "Ga", "Hal", "Ix", "Ka", "Ker", "Lu", "Mi", "Mor", "Ne", "Or", "Pha", "Qua", "Ra", "Sa", "Sol", "Ta", "Thu", "Ul", "Ve", "Vor", "Xe", "Za"}
	middles = []string{"ba", "de", "gi", "la", "lo", "mi", "na", "ni", "ra", "ri", "ro", "sa", "ta", "te", "va", "ze"}
	codas   = []string{"n", "r", "s", "x", "ra", "rin", "lis", "mar", "nox", "phon", "tar", "tis", "vane", "dor"}
)

// mix is splitmix64: well spread bits from an index.
func mix(x uint64) uint64 {
	x += 0x9E3779B97F4A7C15
	x = (x ^ x>>30) * 0xBF58476D1CE4E5B9
	x = (x ^ x>>27) * 0x94D049BB133111EB
	return x ^ x>>31
}

// unit is a number in [0, 1) from h's bits starting at shift.
func unit(h uint64, shift uint) float64 { return float64(h>>shift&0xFFFF) / 0x10000 }

// stream is a splitmix64 sequence: one star's random numbers.
type stream uint64

func (s *stream) next() uint64 {
	v := mix(uint64(*s))
	*s += 0x9E3779B97F4A7C15
	return v
}

func (s *stream) float() float64            { return float64(s.next()>>11) / (1 << 53) }
func (s *stream) pick(list []string) string { return list[s.next()%uint64(len(list))] }

// StarAt is the catalog's star i (0 ≤ i < CatalogSize), the same on every
// call: its numbers come from a stream seeded by its index. It allocates only
// its name and class.
func StarAt(i int) Star {
	s := stream(mix(uint64(i)))
	var buf [24]byte
	name := append(buf[:0], s.pick(onsets)...)
	if s.next()%2 == 0 {
		name = append(name, s.pick(middles)...)
	}
	name = append(name, s.pick(codas)...)

	w := int(s.next() % spectralWeight)
	t := spectral[0]
	for _, c := range spectral {
		if w < c.weight {
			t = c
			break
		}
		w -= c.weight
	}
	sub := int(s.next() % 10)
	absMag := t.bright + (t.faint-t.bright)*(float64(sub)+s.float())/10
	// Most are dwarfs (V); subgiants, giants and supergiants are rarer and brighter.
	lum := "V"
	switch r := s.next() % 100; {
	case r < 3:
		lum, absMag = "I", absMag-7
	case r < 15:
		lum, absMag = "III", absMag-3.5
	case r < 22:
		lum, absMag = "IV", absMag-1
	}
	var cls [5]byte
	class := append(append(cls[:0], t.letter, '0'+byte(sub)), lum...)
	// From 4.2 light years (Proxima Centauri) to 60,000, even on a log scale.
	dist := 4.2 * math.Pow(60000/4.2, s.float())
	mag := absMag + 5*math.Log10(dist/3.2616/10)
	planets := 0
	if s.next()%10 < 6 {
		planets = 1 + int(s.next()%4) + int(s.next()%3)*int(s.next()%3)
	}
	return Star{
		ID:            i + 1,
		Name:          string(name),
		Class:         string(class),
		Temp:          t.hot - (t.hot-t.cool)*sub/10,
		Constellation: s.pick(Constellations),
		Distance:      math.Round(dist*100) / 100,
		Magnitude:     math.Round(mag*100) / 100,
		Planets:       planets,
	}
}

// StarsVersion identifies the stored stars (the first StarCount); the seed
// replaces them when it changes.
func StarsVersion() string {
	h := sha256.New()
	for i := range StarCount {
		s := StarAt(i)
		fmt.Fprintf(h, "%d|%s|%s|%d|%s|%g|%g|%d\n", s.ID, s.Name, s.Class, s.Temp, s.Constellation, s.Distance, s.Magnitude, s.Planets)
	}
	return hex.EncodeToString(h.Sum(nil))[:12]
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
