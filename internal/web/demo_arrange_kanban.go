package web

import (
	"encoding/json"
	"fmt"
	"strings"
)

func init() {
	arrangers["kanban-board"] = arranger{arrange: arrangeKanban, render: renderKanban}
}

// kanbanLanes are the demo board's lanes; data-col is the index.
var kanbanLanes = []string{"To visit", "En route", "Visited"}

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
// leaves its lane and goes into lane col, before another card ("": last).
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
	if m.Col < 0 || m.Col >= len(lanes) {
		return "", fmt.Errorf("no lane %d", m.Col)
	}
	from := -1
	for i, lane := range lanes {
		for j, id := range lane {
			if id == m.CardID {
				from = i
				lanes[i] = append(lane[:j:j], lane[j+1:]...)
				break
			}
		}
	}
	if from < 0 {
		return "", fmt.Errorf("no card %q", m.CardID)
	}
	// In at the end, then before its neighbour: moveBefore checks it.
	if lanes[m.Col], err = moveBefore(append(lanes[m.Col], m.CardID), m.CardID, m.Before); err != nil {
		return "", err
	}
	out := make([]string, len(lanes))
	for i, lane := range lanes {
		out[i] = strings.Join(lane, " ")
	}
	return strings.Join(out, " | "), nil
}

// renderKanban is the board's markup: the host the morph replaces, with the
// arrangement in data-state, a lane per column and a card per body.
func renderKanban(id, state string) string {
	lanes, _ := kanbanState(state)
	var b strings.Builder
	fmt.Fprintf(&b, "<sb-kanban-board id=\"%s\" class=\"demo-kanban\" data-state=\"%s\"\n\tdata-on:sb-kanban-move=\"%s\">\n", id, state, arrangeOn("kanban-board"))
	for i, title := range kanbanLanes {
		fmt.Fprintf(&b, "\t<section data-kanban-lane data-col=\"%d\" aria-label=\"%s\">\n\t\t<p class=\"demo-kanban__title\">%s</p>\n\t\t<div data-kanban-lane-cards>\n", i, title, title)
		if i < len(lanes) {
			for _, card := range lanes[i] {
				fmt.Fprintf(&b, "\t\t\t<article data-kanban-card=\"%s\" tabindex=\"0\">%s</article>\n", card, label(card))
			}
		}
		b.WriteString("\t\t</div>\n\t</section>\n")
	}
	b.WriteString("</sb-kanban-board>")
	return b.String()
}
