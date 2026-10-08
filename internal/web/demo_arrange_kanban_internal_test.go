package web

import (
	"strings"
	"testing"
)

func TestArrangeKanban(t *testing.T) {
	const board = "4: mars jupiter | 7: europa | 9: moon"
	for _, tc := range []struct {
		name, state, move, want string
	}{
		{"down in its lane", board, `{"cardId":"mars","col":4,"before":""}`, "4: jupiter mars | 7: europa | 9: moon"},
		{"up in its lane", board, `{"cardId":"jupiter","col":4,"before":"mars"}`, "4: jupiter mars | 7: europa | 9: moon"},
		{"into another lane", board, `{"cardId":"europa","col":9,"before":"moon"}`, "4: mars jupiter | 7: | 9: europa moon"},
		{"to the end of another lane", board, `{"cardId":"mars","col":7,"before":""}`, "4: jupiter | 7: europa mars | 9: moon"},
		{"into an empty lane", "4: mars | 7: | 9: moon", `{"cardId":"mars","col":7,"before":""}`, "4: | 7: mars | 9: moon"},
		{"into a lane that moved", "9: moon | 4: mars | 7:", `{"cardId":"moon","col":7,"before":""}`, "9: | 4: mars | 7: moon"},
		{"a lane to the end", board, `{"col":4,"before":""}`, "7: europa | 9: moon | 4: mars jupiter"},
		{"a lane to the front", board, `{"col":9,"before":"4"}`, "9: moon | 4: mars jupiter | 7: europa"},
		{"a lane one place right", board, `{"col":4,"before":"9"}`, "7: europa | 4: mars jupiter | 9: moon"},
		{"a lane where it is", board, `{"col":4,"before":"7"}`, board},
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
		{"two lanes", "4: mars | 7: europa", `{"cardId":"mars","col":7,"before":""}`},
		{"four lanes", "4: mars | 7: europa | 9: moon | 9:", `{"cardId":"mars","col":7,"before":""}`},
		{"a lane twice", "4: mars | 4: europa | 9: moon", `{"cardId":"mars","col":9,"before":""}`},
		{"an unknown lane", "4: mars | 5: europa | 9: moon", `{"cardId":"mars","col":9,"before":""}`},
		{"lanes without ids", "mars | europa | moon", `{"cardId":"mars","col":7,"before":""}`},
		{"an unknown body", "4: mars | 7: nibiru | 9: moon", `{"cardId":"mars","col":7,"before":""}`},
		{"a card twice", "4: mars | 7: mars | 9: moon", `{"cardId":"mars","col":7,"before":""}`},
		{"a lane's position, not its id", board, `{"cardId":"mars","col":1,"before":""}`},
		{"a card without a lane", board, `{"cardId":"mars","before":""}`},
		{"a card not on the board", board, `{"cardId":"pluto","col":7,"before":""}`},
		{"before a card in another lane", board, `{"cardId":"mars","col":7,"before":"moon"}`},
		{"before itself", board, `{"cardId":"mars","col":4,"before":"mars"}`},
		{"before an unknown card", board, `{"cardId":"mars","col":7,"before":"nibiru"}`},
		{"a lane id that isn't a number", board, `{"cardId":"mars","col":"7","before":""}`},
		{"a lane move of an unknown lane", board, `{"col":5,"before":""}`},
		{"a lane move of a removed lane", "4: mars | 7: europa | 9: moon", `{"col":3,"before":"4"}`},
		{"a lane move before a stale lane", board, `{"col":4,"before":"5"}`},
		{"a lane move before itself", board, `{"col":4,"before":"4"}`},
		{"a lane move by position", board, `{"col":0,"before":"1"}`},
		{"a lane move with a numeric before", board, `{"col":4,"before":9}`},
		{"no move", board, `null`},
	} {
		if got, err := arrangeKanban(tc.state, []byte(tc.move)); err == nil {
			t.Errorf("%s: %q, want an error", tc.name, got)
		}
	}
}

// The board's markup escapes what it writes into attributes, keeps the lanes
// in the arrangement's order, gives each lane and card an id made of the
// board's and its own and each lane a grip, and shows landing markers that a
// refusal releases.
func TestRenderKanban(t *testing.T) {
	got := renderKanban("plan", "9: moon | 4: mars | 7:")
	for _, want := range []string{
		`<sb-kanban-board id="plan" data-ignore-morph class="demo-kanban" data-kanban-landing data-kanban-landing-timeout="3000" data-state="9: moon | 4: mars | 7:"`,
		`data-on:sb-kanban-lane-move="@get('/demo/arrange/kanban-board?delay=400'`,
		`data-on:datastar-fetch="evt.detail.el === el && evt.detail.type === 'error' && el.releaseLanding()">`,
		`<section id="plan-lane-7" data-kanban-lane data-col="7" tabindex="-1" aria-label="En route">`,
		`<button type="button" class="demo-kanban__grip" data-kanban-lane-grip aria-label="Move the En route lane">`,
		`<article id="plan-mars" data-kanban-card="mars" tabindex="0">`,
		`<article id="plan-moon" data-kanban-card="moon" tabindex="0">`,
	} {
		if !strings.Contains(got, want) {
			t.Errorf("no %s in\n%s", want, got)
		}
	}
	if strings.Count(got, "<article") != 2 || strings.Index(got, `data-col="9"`) > strings.Index(got, `data-col="4"`) {
		t.Errorf("want two cards, lane 9 first:\n%s", got)
	}
}

// The gallery card's board answers moves like the demo, is one Tab stop,
// and fits the card: names without icons, and no grips.
func TestRenderKanbanCard(t *testing.T) {
	got := renderKanbanCard("card", "4: io | 7: mars moon | 9:")
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
	if n := strings.Count(got, `tabindex="0"`); n != 1 || strings.Contains(got, "grip") {
		t.Errorf("%d Tab stops, want 1 and no grips:\n%s", n, got)
	}
	if arrangers["kanban-board-card"].arrange == nil {
		t.Error("the card's moves have no arranger")
	}
}
