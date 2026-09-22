package web

import (
	"net/http"
	"strings"
	"time"
	"unicode"

	"github.com/starfederation/datastar-go/datastar"
	"golang.org/x/text/runes"
	"golang.org/x/text/transform"
	"golang.org/x/text/unicode/norm"
)

// demoStars is sb-select's server-side autocomplete demo: bright stars and
// their constellations.
var demoStars = [][2]string{
	{"Sirius", "Canis Major"}, {"Canopus", "Carina"}, {"Rigil Kentaurus", "Centaurus"}, {"Arcturus", "Boötes"},
	{"Vega", "Lyra"}, {"Capella", "Auriga"}, {"Rigel", "Orion"}, {"Procyon", "Canis Minor"},
	{"Achernar", "Eridanus"}, {"Betelgeuse", "Orion"}, {"Hadar", "Centaurus"}, {"Altair", "Aquila"},
	{"Acrux", "Crux"}, {"Aldebaran", "Taurus"}, {"Antares", "Scorpius"}, {"Spica", "Virgo"},
	{"Pollux", "Gemini"}, {"Fomalhaut", "Piscis Austrinus"}, {"Deneb", "Cygnus"}, {"Mimosa", "Crux"},
	{"Regulus", "Leo"}, {"Adhara", "Canis Major"}, {"Castor", "Gemini"}, {"Shaula", "Scorpius"},
	{"Gacrux", "Crux"}, {"Bellatrix", "Orion"}, {"Elnath", "Taurus"}, {"Miaplacidus", "Carina"},
	{"Alnilam", "Orion"}, {"Alnair", "Grus"}, {"Alnitak", "Orion"}, {"Alioth", "Ursa Major"},
	{"Dubhe", "Ursa Major"}, {"Mirfak", "Perseus"}, {"Wezen", "Canis Major"}, {"Sargas", "Scorpius"},
	{"Kaus Australis", "Sagittarius"}, {"Avior", "Carina"}, {"Alkaid", "Ursa Major"}, {"Menkalinan", "Auriga"},
	{"Atria", "Triangulum Australe"}, {"Alhena", "Gemini"}, {"Peacock", "Pavo"}, {"Alsephina", "Vela"},
	{"Mirzam", "Canis Major"}, {"Alphard", "Hydra"}, {"Polaris", "Ursa Minor"}, {"Hamal", "Aries"},
	{"Algieba", "Leo"}, {"Diphda", "Cetus"}, {"Nunki", "Sagittarius"}, {"Menkent", "Centaurus"},
	{"Mirach", "Andromeda"}, {"Alpheratz", "Andromeda"}, {"Rasalhague", "Ophiuchus"}, {"Kochab", "Ursa Minor"},
	{"Saiph", "Orion"}, {"Denebola", "Leo"}, {"Algol", "Perseus"}, {"Tiaki", "Grus"},
	{"Muhlifain", "Centaurus"}, {"Aspidiske", "Carina"}, {"Suhail", "Vela"}, {"Alphecca", "Corona Borealis"},
	{"Mintaka", "Orion"}, {"Sadr", "Cygnus"}, {"Eltanin", "Draco"}, {"Schedar", "Cassiopeia"},
	{"Naos", "Puppis"}, {"Almach", "Andromeda"}, {"Caph", "Cassiopeia"}, {"Izar", "Boötes"},
	{"Dschubba", "Scorpius"}, {"Larawag", "Scorpius"}, {"Merak", "Ursa Major"}, {"Ankaa", "Phoenix"},
	{"Girtab", "Scorpius"}, {"Enif", "Pegasus"}, {"Scheat", "Pegasus"}, {"Sabik", "Ophiuchus"},
	{"Phecda", "Ursa Major"}, {"Aludra", "Canis Major"}, {"Markeb", "Vela"}, {"Navi", "Cassiopeia"},
	{"Markab", "Pegasus"}, {"Aljanah", "Cygnus"}, {"Acrab", "Scorpius"}, {"Zosma", "Leo"},
	{"Arneb", "Lepus"}, {"Unukalhai", "Serpens"}, {"Tarazed", "Aquila"}, {"Proxima Centauri", "Centaurus"},
}

var foldText = transform.Chain(norm.NFD, runes.Remove(runes.In(unicode.Mn)), norm.NFC)

func fold(s string) string {
	out, _, _ := transform.String(foldText, s)
	return strings.ToLower(out)
}

// demoSearch answers sb-select's sb-search: up to 8 stars whose name or
// constellation matches ?q= (names that start with it first), as a patch of
// the _found signal. A short pause makes the loading state visible.
func (s *Server) demoSearch(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*") // public; used from the playground sandbox
	q := fold(strings.TrimSpace(r.URL.Query().Get("q")))
	var prefix, contains []map[string]any
	for _, st := range demoStars {
		name, con := fold(st[0]), fold(st[1])
		o := map[string]any{"value": st[0], "label": st[0], "description": st[1]}
		switch {
		case q == "":
		case strings.HasPrefix(name, q):
			prefix = append(prefix, o)
		case strings.Contains(name, q) || strings.Contains(con, q):
			contains = append(contains, o)
		}
	}
	found := append(prefix, contains...)
	if len(found) > 8 {
		found = found[:8]
	}
	if found == nil {
		found = []map[string]any{}
	}
	select {
	case <-time.After(200 * time.Millisecond):
	case <-r.Context().Done():
		return
	}
	datastar.NewSSE(w, r).MarshalAndPatchSignals(map[string]any{"_found": found})
}
