package web

import (
	"cmp"
	"encoding/json"
	"fmt"
	"slices"
	"strconv"
	"strings"
)

func init() {
	arrangers["bento-workspace"] = arranger{arrange: arrangeBento, render: renderBento}
}

// The bento demo is a dashboard: its grids, in order, and its tiles, each
// holding one of Starbase's components with a fixed reading.
var (
	bentoGrids = []bentoGrid{{"deck", "Deck", 4}, {"shelf", "Shelf", 2}}
	bentoTiles = map[string]struct{ label, body string }{
		"thrust":  {"Thrust", `<sb-gauge value="72" label="Thrust" unit="%"></sb-gauge>`},
		"fuel":    {"Fuel", `<sb-meter label="Fuel" value="64" unit="%"></sb-meter>`},
		"speed":   {"Speed", `<sb-sparkline values="[12,18,15,22,30,26,34,41]" show-value unit=" km/s"></sb-sparkline>`},
		"shields": {"Shields", `<sb-meter label="Shields" value="88" unit="%"></sb-meter>`},
		"crew":    {"Crew", `<p><strong>7</strong> aboard</p>`},
	}
)

const bentoRows = 30 // five tiles of at most five rows, stacked

type bentoGrid struct {
	id, label string
	columns   int
}

// A bento arrangement is its tiles' places. Its state is the grids' ids, in
// the demo's order, each followed by its tiles as id.col.row.width.height:
// "deck thrust.1.1.2.2 shelf crew.1.1.1.1".
type bentoTile struct {
	id, grid                string
	col, row, width, height int
}

// bentoUpdate is a tile's new place, as the component reports it.
type bentoUpdate struct {
	ItemID string `json:"itemId"`
	Grid   string `json:"grid"`
	Col    int    `json:"col"`
	Row    int    `json:"row"`
	Width  int    `json:"width"`
	Height int    `json:"height"`
}

// parseBento reads a state and checks it holds every tile once, inside its
// grid, without overlaps.
func parseBento(state string) ([]bentoTile, error) {
	var tiles []bentoTile
	g := -1
	for _, w := range strings.Fields(state) {
		id, pos, ok := strings.Cut(w, ".")
		if !ok {
			if g++; g >= len(bentoGrids) || bentoGrids[g].id != w {
				return nil, fmt.Errorf("%q: the grids are deck and shelf, in that order", w)
			}
			continue
		}
		var n [4]int
		parts := strings.Split(pos, ".")
		for i := 0; i < len(parts) && i < 4; i++ {
			n[i], _ = strconv.Atoi(parts[i])
		}
		if g < 0 || len(parts) != 4 {
			return nil, fmt.Errorf("%q: a tile is id.col.row.width.height, after its grid", w)
		}
		tiles = append(tiles, bentoTile{id, bentoGrids[g].id, n[0], n[1], n[2], n[3]})
	}
	if g != len(bentoGrids)-1 {
		return nil, fmt.Errorf("%q lacks a grid", state)
	}
	return tiles, checkBento(tiles)
}

// checkBento holds every tile once, inside its grid, and no two overlap.
func checkBento(tiles []bentoTile) error {
	seen := map[string]bool{}
	for i, t := range tiles {
		_, known := bentoTiles[t.id]
		gi := slices.IndexFunc(bentoGrids, func(g bentoGrid) bool { return g.id == t.grid })
		if !known || seen[t.id] || gi < 0 {
			return fmt.Errorf("%q is no tile of the demo, or there twice", t.id)
		}
		seen[t.id] = true
		cols := bentoGrids[gi].columns
		if t.col < 1 || t.width < 1 || t.col+t.width-1 > cols || t.row < 1 || t.height < 1 || t.height > 5 || t.row+t.height-1 > bentoRows {
			return fmt.Errorf("%s is outside its grid", t.id)
		}
		for _, u := range tiles[:i] {
			if u.grid == t.grid && t.col < u.col+u.width && u.col < t.col+t.width && t.row < u.row+u.height && u.row < t.row+t.height {
				return fmt.Errorf("%s overlaps %s", t.id, u.id)
			}
		}
	}
	if len(seen) != len(bentoTiles) {
		return fmt.Errorf("a tile is missing")
	}
	return nil
}

// formatBento is the state of an arrangement: each grid's tiles in reading order.
func formatBento(tiles []bentoTile) string {
	slices.SortStableFunc(tiles, func(a, b bentoTile) int { return cmp.Or(cmp.Compare(a.row, b.row), cmp.Compare(a.col, b.col)) })
	var words []string
	for _, g := range bentoGrids {
		words = append(words, g.id)
		for _, t := range tiles {
			if t.grid == g.id {
				words = append(words, fmt.Sprintf("%s.%d.%d.%d.%d", t.id, t.col, t.row, t.width, t.height))
			}
		}
	}
	return strings.Join(words, " ")
}

// arrangeBento applies an sb-bento-move ({itemId, fromGrid, toGrid, updates})
// or an sb-bento-resize ({itemId, grid, updates}): updates are the new places
// of every tile that changed in the grid the tile lands in.
func arrangeBento(state string, move json.RawMessage) (string, error) {
	tiles, err := parseBento(state)
	if err != nil {
		return "", err
	}
	var m struct {
		ItemID   string        `json:"itemId"`
		FromGrid string        `json:"fromGrid"` // a move
		ToGrid   string        `json:"toGrid"`
		Grid     string        `json:"grid"` // a resize
		Updates  []bentoUpdate `json:"updates"`
	}
	if err := json.Unmarshal(move, &m); err != nil {
		return "", err
	}
	at := slices.IndexFunc(tiles, func(t bentoTile) bool { return t.id == m.ItemID })
	from, to := cmp.Or(m.FromGrid, m.Grid), cmp.Or(m.ToGrid, m.Grid)
	moved := slices.ContainsFunc(m.Updates, func(u bentoUpdate) bool { return u.ItemID == m.ItemID })
	if at < 0 || tiles[at].grid != from || len(m.Updates) > len(tiles) || !moved {
		return "", fmt.Errorf("not a move of %q", m.ItemID)
	}
	for _, u := range m.Updates {
		i := slices.IndexFunc(tiles, func(t bentoTile) bool { return t.id == u.ItemID })
		if i < 0 || u.Grid != to {
			return "", fmt.Errorf("an update of %q outside the grid it lands in", u.ItemID)
		}
		tiles[i] = bentoTile{u.ItemID, u.Grid, u.Col, u.Row, u.Width, u.Height}
	}
	if err := checkBento(tiles); err != nil {
		return "", err
	}
	return formatBento(tiles), nil
}

// renderBento is the dashboard's markup: the host the morph replaces, a
// grid per panel, and each tile at its place.
func renderBento(id, state string) string {
	tiles, err := parseBento(state)
	if err != nil {
		return "invalid bento state: " + err.Error()
	}
	var b strings.Builder
	fmt.Fprintf(&b, "<sb-bento-workspace id=\"%s\" class=\"demo-bento\" data-state=\"%s\"\n\tdata-on:sb-bento-move=\"%s\"\n\tdata-on:sb-bento-resize=\"%[3]s\">\n", id, state, arrangeOn("bento-workspace"))
	for _, g := range bentoGrids {
		fmt.Fprintf(&b, "\t<div class=\"demo-bento__panel\">\n\t\t<span class=\"demo-bento__label\">%s</span>\n", g.label)
		fmt.Fprintf(&b, "\t\t<div data-bento-grid=\"%s\" data-columns=\"%d\" role=\"group\" aria-label=\"%s\">\n", g.id, g.columns, g.label)
		for _, t := range tiles {
			if t.grid != g.id {
				continue
			}
			fmt.Fprintf(&b, "\t\t\t<article data-bento-item=\"%s\" data-bento-col=\"%d\" data-bento-row=\"%d\" data-bento-width=\"%d\" data-bento-height=\"%d\"\n", t.id, t.col, t.row, t.width, t.height)
			fmt.Fprintf(&b, "\t\t\t\tstyle=\"grid-column: %d / span %d; grid-row: %d / span %d\" tabindex=\"0\" aria-label=\"%s\">\n", t.col, t.width, t.row, t.height, bentoTiles[t.id].label)
			fmt.Fprintf(&b, "\t\t\t\t%s\n\t\t\t\t<span data-bento-resize aria-hidden=\"true\"></span>\n\t\t\t</article>\n", bentoTiles[t.id].body)
		}
		b.WriteString("\t\t</div>\n\t</div>\n")
	}
	b.WriteString("</sb-bento-workspace>")
	return b.String()
}
