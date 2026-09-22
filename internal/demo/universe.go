// Package demo is the example dataset behind the demo endpoints
// (/demo/data/…): a small universe with a hierarchy (galaxies, groups,
// systems, planets, moons) and enough bodies to search. Components' docs and
// playgrounds use it for lazy trees, autocomplete, tables and the like.
package demo

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"
	"unicode"

	"golang.org/x/text/runes"
	"golang.org/x/text/transform"
	"golang.org/x/text/unicode/norm"
)

// Body is one thing in the universe.
type Body struct {
	ID     string // letters and digits only: it becomes a signal key
	Parent string // "" for the top level
	Kind   string // galaxy, group, constellation, system, star, planet, dwarf, moon
	Name   string
	Detail string // one line, e.g. "Star in Orion"
	Order  int    // position among its siblings
}

// Icons per kind, for trees and lists.
var Icons = map[string]string{
	"galaxy": "🌌", "group": "🗂️", "constellation": "✨", "system": "☀️",
	"star": "⭐", "planet": "🪐", "dwarf": "🪨", "moon": "🌑",
}

// Fold is the searchable form of a text: lower case, without accents.
func Fold(s string) string {
	out, _, _ := transform.String(transform.Chain(norm.NFD, runes.Remove(runes.In(unicode.Mn)), norm.NFC), s)
	return strings.ToLower(out)
}

// ID makes an id from a name: its letters and digits, lower case.
func ID(name string) string {
	var b strings.Builder
	for _, r := range Fold(name) {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			b.WriteRune(r)
		}
	}
	return b.String()
}

// Bright stars and their constellations.
var brightStars = [][2]string{
	{"Sirius", "Canis Major"}, {"Canopus", "Carina"}, {"Arcturus", "Boötes"}, {"Vega", "Lyra"},
	{"Capella", "Auriga"}, {"Rigel", "Orion"}, {"Procyon", "Canis Minor"}, {"Achernar", "Eridanus"},
	{"Betelgeuse", "Orion"}, {"Hadar", "Centaurus"}, {"Altair", "Aquila"}, {"Acrux", "Crux"},
	{"Aldebaran", "Taurus"}, {"Antares", "Scorpius"}, {"Spica", "Virgo"}, {"Pollux", "Gemini"},
	{"Fomalhaut", "Piscis Austrinus"}, {"Deneb", "Cygnus"}, {"Mimosa", "Crux"}, {"Regulus", "Leo"},
	{"Adhara", "Canis Major"}, {"Castor", "Gemini"}, {"Shaula", "Scorpius"}, {"Gacrux", "Crux"},
	{"Bellatrix", "Orion"}, {"Elnath", "Taurus"}, {"Miaplacidus", "Carina"}, {"Alnilam", "Orion"},
	{"Alnair", "Grus"}, {"Alnitak", "Orion"}, {"Alioth", "Ursa Major"}, {"Dubhe", "Ursa Major"},
	{"Mirfak", "Perseus"}, {"Wezen", "Canis Major"}, {"Sargas", "Scorpius"}, {"Kaus Australis", "Sagittarius"},
	{"Avior", "Carina"}, {"Alkaid", "Ursa Major"}, {"Menkalinan", "Auriga"}, {"Atria", "Triangulum Australe"},
	{"Alhena", "Gemini"}, {"Peacock", "Pavo"}, {"Mirzam", "Canis Major"}, {"Alphard", "Hydra"},
	{"Polaris", "Ursa Minor"}, {"Hamal", "Aries"}, {"Algieba", "Leo"}, {"Diphda", "Cetus"},
	{"Nunki", "Sagittarius"}, {"Menkent", "Centaurus"}, {"Mirach", "Andromeda"}, {"Alpheratz", "Andromeda"},
	{"Rasalhague", "Ophiuchus"}, {"Kochab", "Ursa Minor"}, {"Saiph", "Orion"}, {"Denebola", "Leo"},
	{"Algol", "Perseus"}, {"Alphecca", "Corona Borealis"}, {"Mintaka", "Orion"}, {"Sadr", "Cygnus"},
	{"Eltanin", "Draco"}, {"Schedar", "Cassiopeia"}, {"Naos", "Puppis"}, {"Almach", "Andromeda"},
	{"Caph", "Cassiopeia"}, {"Izar", "Boötes"}, {"Merak", "Ursa Major"}, {"Ankaa", "Phoenix"},
	{"Enif", "Pegasus"}, {"Scheat", "Pegasus"}, {"Sabik", "Ophiuchus"}, {"Phecda", "Ursa Major"},
	{"Markab", "Pegasus"}, {"Zosma", "Leo"}, {"Arneb", "Lepus"}, {"Unukalhai", "Serpens"},
	{"Tarazed", "Aquila"},
}

// Universe returns every body, parents before their children.
func Universe() []Body {
	var out []Body
	order := map[string]int{}
	taken := map[string]bool{}
	add := func(parent, kind, name, detail string) string {
		id := ID(name)
		if taken[id] { // e.g. Andromeda, the galaxy and the constellation
			id += kind
		}
		taken[id] = true
		out = append(out, Body{ID: id, Parent: parent, Kind: kind, Name: name, Detail: detail, Order: order[parent]})
		order[parent]++
		return id
	}

	mw := add("", "galaxy", "Milky Way", "Our galaxy, a barred spiral")
	andromeda := add("", "galaxy", "Andromeda", "The nearest large spiral galaxy")
	add("", "galaxy", "Triangulum", "The third-largest galaxy of the Local Group")
	add("", "galaxy", "Large Magellanic Cloud", "A satellite galaxy of the Milky Way")
	add("", "galaxy", "Small Magellanic Cloud", "A satellite galaxy of the Milky Way")
	add(andromeda, "galaxy", "Messier 32", "A dwarf galaxy orbiting Andromeda")
	add(andromeda, "galaxy", "Messier 110", "A dwarf galaxy orbiting Andromeda")

	// The Solar System.
	sol := add(mw, "system", "Solar System", "The Sun and everything that orbits it")
	add(sol, "star", "Sun", "The star of the Solar System")
	moons := map[string][]string{
		"Earth":   {"Moon"},
		"Mars":    {"Phobos", "Deimos"},
		"Jupiter": {"Io", "Europa", "Ganymede", "Callisto"},
		"Saturn":  {"Titan", "Enceladus", "Rhea", "Mimas", "Iapetus"},
		"Uranus":  {"Titania", "Oberon", "Miranda"},
		"Neptune": {"Triton"},
		"Pluto":   {"Charon"},
	}
	for _, p := range []struct{ name, kind string }{
		{"Mercury", "planet"}, {"Venus", "planet"}, {"Earth", "planet"}, {"Mars", "planet"},
		{"Ceres", "dwarf"}, {"Jupiter", "planet"}, {"Saturn", "planet"}, {"Uranus", "planet"},
		{"Neptune", "planet"}, {"Pluto", "dwarf"},
	} {
		detail := "Planet of the Solar System"
		if p.kind == "dwarf" {
			detail = "Dwarf planet of the Solar System"
		}
		id := add(sol, p.kind, p.name, detail)
		for _, m := range moons[p.name] {
			add(id, "moon", m, "Moon of "+p.name)
		}
	}

	// Nearby systems.
	near := add(mw, "group", "Nearby systems", "Stars within a few dozen light years")
	ac := add(near, "system", "Alpha Centauri", "The closest star system to the Sun")
	add(ac, "star", "Rigil Kentaurus", "Star in Centaurus")
	add(ac, "star", "Toliman", "Star in Centaurus")
	prox := add(ac, "star", "Proxima Centauri", "The closest star to the Sun")
	add(prox, "planet", "Proxima b", "Exoplanet of Proxima Centauri")
	add(prox, "planet", "Proxima d", "Exoplanet of Proxima Centauri")
	add(near, "star", "Barnard's Star", "A red dwarf in Ophiuchus")
	tr := add(near, "system", "TRAPPIST-1", "A red dwarf with seven known planets")
	for _, l := range "bcdefgh" {
		add(tr, "planet", "TRAPPIST-1"+string(l), "Exoplanet of TRAPPIST-1")
	}

	// Bright stars, by constellation (skipping ids already used).
	cons := add(mw, "group", "Constellations", "Bright stars, grouped by constellation")
	conID := map[string]string{}
	for _, s := range brightStars {
		if taken[ID(s[0])] {
			continue
		}
		c, ok := conID[s[1]]
		if !ok {
			c = add(cons, "constellation", s[1], "Constellation")
			conID[s[1]] = c
		}
		add(c, "star", s[0], "Star in "+s[1])
	}
	return out
}

// Version identifies the dataset; the seed replaces the stored one when it changes.
func Version() string {
	h := sha256.New()
	for _, b := range Universe() {
		fmt.Fprintf(h, "%s|%s|%s|%s|%s|%d\n", b.ID, b.Parent, b.Kind, b.Name, b.Detail, b.Order)
	}
	return hex.EncodeToString(h.Sum(nil))[:12]
}
