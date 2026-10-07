package web

import (
	"encoding/json"
	"fmt"
	"html"
	"slices"
	"strconv"
	"strings"
)

func init() {
	arrangers["kanban-board"] = arranger{arrange: arrangeKanban, render: renderKanban}
	arrangers["kanban-board-card"] = arranger{arrange: arrangeKanban, render: renderKanbanCard}
}

// A kanbanLane is a lane of the demo board. Its ID is its data-col: a move
// names the lane by id, which stays the same wherever the lane is.
type kanbanLane struct {
	ID    int
	Title string
}

// kanbanLanes are the demo board's lanes.
var kanbanLanes = []kanbanLane{{4, "To visit"}, {7, "En route"}, {9, "Visited"}}

// A kanbanColumn is a lane in a board's arrangement: its id and its cards.
type kanbanColumn struct {
	ID    int
	Cards []string
}

// kanbanState reads a board: its lanes in their order, each its id and its
// bodies ("4: mars venus | 9: | 7: earth"), every lane of kanbanLanes once.
func kanbanState(state string) ([]kanbanColumn, error) {
	parts := strings.Split(state, "|")
	if len(parts) != len(kanbanLanes) {
		return nil, fmt.Errorf("%q is not a board of %d lanes", state, len(kanbanLanes))
	}
	cols := make([]kanbanColumn, 0, len(parts))
	var cards []string
	for _, p := range parts {
		id, list, _ := strings.Cut(p, ":")
		n, err := strconv.Atoi(strings.TrimSpace(id))
		if err != nil || kanbanTitle(n) == "" || slices.ContainsFunc(cols, func(c kanbanColumn) bool { return c.ID == n }) {
			return nil, fmt.Errorf("%q is not a lane of the board", p)
		}
		cols = append(cols, kanbanColumn{n, strings.Fields(list)})
		cards = append(cards, strings.Fields(list)...)
	}
	if _, err := ids(strings.Join(cards, " ")); err != nil {
		return nil, err
	}
	return cols, nil
}

// kanbanTitle is the title of the lane with that id, "" for none.
func kanbanTitle(id int) string {
	if i := slices.IndexFunc(kanbanLanes, func(l kanbanLane) bool { return l.ID == id }); i >= 0 {
		return kanbanLanes[i].Title
	}
	return ""
}

// kanbanFormat writes a board as kanbanState reads it.
func kanbanFormat(cols []kanbanColumn) string {
	out := make([]string, len(cols))
	for i, c := range cols {
		out[i] = strings.TrimSpace(strconv.Itoa(c.ID) + ": " + strings.Join(c.Cards, " "))
	}
	return strings.Join(out, " | ")
}

// arrangeKanban applies an sb-kanban-move ({cardId, col, before}): the card
// leaves its lane and goes into the lane whose id is col, before another
// card ("": last). A move without a card moves a lane (arrangeKanbanLane).
func arrangeKanban(state string, move json.RawMessage) (string, error) {
	cols, err := kanbanState(state)
	if err != nil {
		return "", err
	}
	var m struct {
		CardID *string `json:"cardId"`
		Col    int     `json:"col"`
		Before string  `json:"before"`
	}
	if err := json.Unmarshal(move, &m); err != nil {
		return "", err
	}
	if m.CardID == nil {
		return arrangeKanbanLane(cols, m.Col, m.Before)
	}
	to := slices.IndexFunc(cols, func(c kanbanColumn) bool { return c.ID == m.Col })
	if to < 0 {
		return "", fmt.Errorf("no lane %d", m.Col)
	}
	from := -1
	for i, c := range cols {
		if j := slices.Index(c.Cards, *m.CardID); j >= 0 {
			from = i
			cols[i].Cards = slices.Delete(c.Cards, j, j+1)
		}
	}
	if from < 0 {
		return "", fmt.Errorf("no card %q", *m.CardID)
	}
	// In at the end, then before its neighbour: moveBefore checks it.
	if cols[to].Cards, err = moveBefore(append(cols[to].Cards, *m.CardID), *m.CardID, m.Before); err != nil {
		return "", err
	}
	return kanbanFormat(cols), nil
}

// arrangeKanbanLane applies an sb-kanban-lane-move ({col, before}): the lane
// whose id is col goes before the lane whose id is before ("": last). An id
// the board doesn't have is refused, since the page that sent it is out of
// date: it never lands the lane next to the wrong neighbour.
func arrangeKanbanLane(cols []kanbanColumn, col int, before string) (string, error) {
	order := make([]string, len(cols))
	for i, c := range cols {
		order[i] = strconv.Itoa(c.ID)
	}
	order, err := moveBefore(order, strconv.Itoa(col), before)
	if err != nil {
		return "", fmt.Errorf("no lane move %d before %q: %w", col, before, err)
	}
	moved := make([]kanbanColumn, len(order))
	for i, id := range order {
		moved[i] = cols[slices.IndexFunc(cols, func(c kanbanColumn) bool { return strconv.Itoa(c.ID) == id })]
	}
	return kanbanFormat(moved), nil
}

// renderKanban is the board's markup: the host the morph replaces, with the
// arrangement in data-state, a lane per column and a card per body. Lanes
// and cards have ids, so the morph moves them (and their focus) instead of
// rewriting one in another's place, and each lane has a grip to drag it by.
func renderKanban(id, state string) string {
	cols, _ := kanbanState(state)
	esc := html.EscapeString
	var b strings.Builder
	on := arrangeOn("kanban-board")
	fmt.Fprintf(&b, "<sb-kanban-board id=\"%s\" data-ignore-morph class=\"demo-kanban\" data-state=\"%s\"\n\tdata-on:sb-kanban-move=\"%s\"\n\tdata-on:sb-kanban-lane-move=\"%s\">\n", esc(id), esc(state), on, on)
	for _, c := range cols {
		title := esc(kanbanTitle(c.ID))
		fmt.Fprintf(&b, "\t<section id=\"%s-lane-%d\" data-kanban-lane data-col=\"%d\" tabindex=\"-1\" aria-label=\"%s\">\n", esc(id), c.ID, c.ID, title)
		fmt.Fprintf(&b, "\t\t<div class=\"demo-kanban__head\">\n\t\t\t<button type=\"button\" class=\"demo-kanban__grip\" data-kanban-lane-grip aria-label=\"Move the %s lane\">↔</button>\n\t\t\t<p class=\"demo-kanban__title\">%s</p>\n\t\t</div>\n\t\t<div data-kanban-lane-cards>\n", title, title)
		for _, card := range c.Cards {
			fmt.Fprintf(&b, "\t\t\t<article id=\"%s-%s\" data-kanban-card=\"%s\" tabindex=\"0\">%s</article>\n", esc(id), esc(card), esc(card), label(card))
		}
		b.WriteString("\t\t</div>\n\t</section>\n")
	}
	b.WriteString("</sb-kanban-board>")
	return b.String()
}

// renderKanbanCard is the gallery card's board: the same lanes and moves in
// markup small enough for the card, with names only. Its first card is the
// card's one Tab stop, which the arrows take along to the others.
func renderKanbanCard(id, state string) string {
	cols, _ := kanbanState(state)
	esc := html.EscapeString
	var b strings.Builder
	fmt.Fprintf(&b, "<sb-kanban-board id=\"%s\" data-ignore-morph class=\"demo-kanban-card\" data-state=\"%s\" data-on:sb-kanban-move=\"%s\">\n", esc(id), esc(state), arrangeOn("kanban-board-card"))
	tab := "0"
	for _, c := range cols {
		fmt.Fprintf(&b, "<section data-kanban-lane data-col=\"%d\" tabindex=\"-1\" aria-label=\"%s\"><div data-kanban-lane-cards>", c.ID, esc(kanbanTitle(c.ID)))
		for _, card := range c.Cards {
			fmt.Fprintf(&b, "<article id=\"%s-%s\" data-kanban-card=\"%s\" tabindex=\"%s\">%s</article>", esc(id), esc(card), esc(card), tab, esc(bodies()[card].Name))
			tab = "-1"
		}
		b.WriteString("</div></section>\n")
	}
	b.WriteString("</sb-kanban-board>")
	return b.String()
}
