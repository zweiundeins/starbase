package web

import (
	"encoding/json"
	"fmt"
	"html"
	"slices"
	"strings"
)

func init() {
	arrangers["kanban-board"] = arranger{arrange: arrangeKanban, render: renderKanban}
}

// A kanbanLane is a lane of the demo board. Its ID is its data-col: a move
// names the lane by id, which stays the same wherever the lane is.
type kanbanLane struct {
	ID    int
	Title string
}

// kanbanLanes are the demo board's lanes, in their order.
var kanbanLanes = []kanbanLane{{4, "To visit"}, {7, "En route"}, {9, "Visited"}}

// kanbanState reads a board: each lane's bodies, lanes separated by "|"
// ("mars venus | jupiter | earth").
func kanbanState(state string) ([][]string, error) {
	parts := strings.Split(state, "|")
	if len(parts) != len(kanbanLanes) {
		return nil, fmt.Errorf("%q is not a board of %d lanes", state, len(kanbanLanes))
	}
	if _, err := ids(strings.ReplaceAll(state, "|", " ")); err != nil {
		return nil, err
	}
	lanes := make([][]string, len(parts))
	for i, p := range parts {
		lanes[i] = strings.Fields(p)
	}
	return lanes, nil
}

// arrangeKanban applies an sb-kanban-move ({cardId, col, before}): the card
// leaves its lane and goes into the lane whose id is col, before another
// card ("": last).
func arrangeKanban(state string, move json.RawMessage) (string, error) {
	lanes, err := kanbanState(state)
	if err != nil {
		return "", err
	}
	var m struct {
		CardID string `json:"cardId"`
		Col    int    `json:"col"`
		Before string `json:"before"`
	}
	if err := json.Unmarshal(move, &m); err != nil {
		return "", err
	}
	to := slices.IndexFunc(kanbanLanes, func(l kanbanLane) bool { return l.ID == m.Col })
	if to < 0 {
		return "", fmt.Errorf("no lane %d", m.Col)
	}
	from := -1
	for i, lane := range lanes {
		if j := slices.Index(lane, m.CardID); j >= 0 {
			from = i
			lanes[i] = slices.Delete(lane, j, j+1)
		}
	}
	if from < 0 {
		return "", fmt.Errorf("no card %q", m.CardID)
	}
	// In at the end, then before its neighbour: moveBefore checks it.
	if lanes[to], err = moveBefore(append(lanes[to], m.CardID), m.CardID, m.Before); err != nil {
		return "", err
	}
	out := make([]string, len(lanes))
	for i, lane := range lanes {
		out[i] = strings.Join(lane, " ")
	}
	// Fields drops the extra space an empty lane leaves: "mars | | moon".
	return strings.Join(strings.Fields(strings.Join(out, " | ")), " "), nil
}

// renderKanban is the board's markup: the host the morph replaces, with the
// arrangement in data-state, a lane per column and a card per body. Each card
// has an id, so the morph moves it (and its focus) instead of rewriting
// another card in its place.
func renderKanban(id, state string) string {
	lanes, _ := kanbanState(state)
	esc := html.EscapeString
	var b strings.Builder
	fmt.Fprintf(&b, "<sb-kanban-board id=\"%s\" class=\"demo-kanban\" data-state=\"%s\"\n\tdata-on:sb-kanban-move=\"%s\">\n", esc(id), esc(state), arrangeOn("kanban-board"))
	for i, lane := range kanbanLanes {
		fmt.Fprintf(&b, "\t<section data-kanban-lane data-col=\"%d\" tabindex=\"-1\" aria-label=\"%s\">\n\t\t<p class=\"demo-kanban__title\">%s</p>\n\t\t<div data-kanban-lane-cards>\n", lane.ID, esc(lane.Title), esc(lane.Title))
		if i < len(lanes) {
			for _, card := range lanes[i] {
				fmt.Fprintf(&b, "\t\t\t<article id=\"%s-%s\" data-kanban-card=\"%s\" tabindex=\"0\">%s</article>\n", esc(id), esc(card), esc(card), label(card))
			}
		}
		b.WriteString("\t\t</div>\n\t</section>\n")
	}
	b.WriteString("</sb-kanban-board>")
	return b.String()
}
