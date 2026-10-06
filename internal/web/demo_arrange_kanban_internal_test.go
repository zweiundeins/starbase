package web

import (
	"strings"
	"testing"
)

func TestArrangeKanban(t *testing.T) {
	const board = "mars jupiter | europa | moon"
	for _, tc := range []struct {
		name, state, move, want string
	}{
		{"down in its lane", board, `{"cardId":"mars","col":4,"before":""}`, "jupiter mars | europa | moon"},
		{"up in its lane", board, `{"cardId":"jupiter","col":4,"before":"mars"}`, "jupiter mars | europa | moon"},
		{"into another lane", board, `{"cardId":"europa","col":9,"before":"moon"}`, "mars jupiter | | europa moon"},
		{"to the end of another lane", board, `{"cardId":"mars","col":7,"before":""}`, "jupiter | europa mars | moon"},
		{"into an empty lane", "mars | | moon", `{"cardId":"mars","col":7,"before":""}`, "| mars | moon"},
	} {
		got, err := arrangeKanban(tc.state, []byte(tc.move))
		if err != nil || got != tc.want {
			t.Errorf("%s: %q, %v; want %q", tc.name, got, err, tc.want)
		}
		if _, err := kanbanState(got); err != nil {
			t.Errorf("%s: the answer %q doesn't read back: %v", tc.name, got, err)
		}
	}
	for _, tc := range []struct{ name, state, move string }{
		{"two lanes", "mars | europa", `{"cardId":"mars","col":7,"before":""}`},
		{"four lanes", "mars | europa | moon |", `{"cardId":"mars","col":7,"before":""}`},
		{"an unknown body", "mars | nibiru | moon", `{"cardId":"mars","col":7,"before":""}`},
		{"a card twice", "mars | mars | moon", `{"cardId":"mars","col":7,"before":""}`},
		{"a lane's position, not its id", board, `{"cardId":"mars","col":1,"before":""}`},
		{"no lane", board, `{"cardId":"mars","before":""}`},
		{"a card not on the board", board, `{"cardId":"pluto","col":7,"before":""}`},
		{"before a card in another lane", board, `{"cardId":"mars","col":7,"before":"moon"}`},
		{"before itself", board, `{"cardId":"mars","col":4,"before":"mars"}`},
		{"before an unknown card", board, `{"cardId":"mars","col":7,"before":"nibiru"}`},
		{"a lane id that isn't a number", board, `{"cardId":"mars","col":"7","before":""}`},
		{"no move", board, `null`},
	} {
		if got, err := arrangeKanban(tc.state, []byte(tc.move)); err == nil {
			t.Errorf("%s: %q, want an error", tc.name, got)
		}
	}
}

// The board's markup escapes what it writes into attributes, and gives each
// card an id made of the board's and the card's.
func TestRenderKanban(t *testing.T) {
	got := renderKanban("plan", "mars | | moon")
	for _, want := range []string{
		`<sb-kanban-board id="plan" class="demo-kanban" data-state="mars | | moon"`,
		`<section data-kanban-lane data-col="7" tabindex="-1" aria-label="En route">`,
		`<article id="plan-mars" data-kanban-card="mars" tabindex="0">`,
		`<article id="plan-moon" data-kanban-card="moon" tabindex="0">`,
	} {
		if !strings.Contains(got, want) {
			t.Errorf("no %s in\n%s", want, got)
		}
	}
	if strings.Count(got, "<article") != 2 {
		t.Errorf("want two cards:\n%s", got)
	}
}

// The gallery card's board answers moves like the demo, is one Tab stop,
// and fits the card: names without icons.
func TestRenderKanbanCard(t *testing.T) {
	got := renderKanbanCard("card", "io | mars moon |")
	for _, want := range []string{
		`data-on:sb-kanban-move="@get('/demo/arrange/kanban-board-card'`,
		`<article id="card-io" data-kanban-card="io" tabindex="0">Io</article>`,
		`<article id="card-mars" data-kanban-card="mars" tabindex="-1">Mars</article>`,
		`<section data-kanban-lane data-col="9" tabindex="-1" aria-label="Visited"><div data-kanban-lane-cards></div></section>`,
	} {
		if !strings.Contains(got, want) {
			t.Errorf("no %s in\n%s", want, got)
		}
	}
	if n := strings.Count(got, `tabindex="0"`); n != 1 {
		t.Errorf("%d Tab stops, want 1:\n%s", n, got)
	}
	if arrangers["kanban-board-card"].arrange == nil {
		t.Error("the card's moves have no arranger")
	}
}
