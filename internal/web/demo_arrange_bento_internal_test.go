package web

import (
	"encoding/json"
	"strings"
	"testing"
)

const bentoStart = "deck thrust.1.1.2.2 fuel.3.1.2.1 speed.3.2.2.1 shelf shields.1.1.2.1 crew.1.2.1.1"

func TestArrangeBento(t *testing.T) {
	for _, tc := range []struct{ name, move, want string }{
		{"a move down pushes the tile below", `{"itemId":"fuel","fromGrid":"deck","toGrid":"deck","updates":[{"itemId":"fuel","grid":"deck","col":3,"row":2,"width":2,"height":1},{"itemId":"speed","grid":"deck","col":3,"row":3,"width":2,"height":1}]}`,
			"deck thrust.1.1.2.2 fuel.3.2.2.1 speed.3.3.2.1 shelf shields.1.1.2.1 crew.1.2.1.1"},
		{"a resize", `{"itemId":"shields","grid":"shelf","updates":[{"itemId":"shields","grid":"shelf","col":1,"row":1,"width":2,"height":2},{"itemId":"crew","grid":"shelf","col":1,"row":3,"width":1,"height":1}]}`,
			"deck thrust.1.1.2.2 fuel.3.1.2.1 speed.3.2.2.1 shelf shields.1.1.2.2 crew.1.3.1.1"},
		{"to the other grid", `{"itemId":"speed","fromGrid":"deck","toGrid":"shelf","updates":[{"itemId":"speed","grid":"shelf","col":1,"row":1,"width":2,"height":1},{"itemId":"shields","grid":"shelf","col":1,"row":2,"width":2,"height":1},{"itemId":"crew","grid":"shelf","col":1,"row":3,"width":1,"height":1}]}`,
			"deck thrust.1.1.2.2 fuel.3.1.2.1 shelf speed.1.1.2.1 shields.1.2.2.1 crew.1.3.1.1"},
		{"to the last row", `{"itemId":"crew","fromGrid":"shelf","toGrid":"shelf","updates":[{"itemId":"crew","grid":"shelf","col":2,"row":30,"width":1,"height":1}]}`,
			"deck thrust.1.1.2.2 fuel.3.1.2.1 speed.3.2.2.1 shelf shields.1.1.2.1 crew.2.30.1.1"},
	} {
		got, err := bentoDashboard.arrange(bentoStart, json.RawMessage(tc.move))
		if err != nil || got != tc.want {
			t.Errorf("%s: %q %v, want %q", tc.name, got, err, tc.want)
		}
	}
	const huge = "9223372036854775807"
	for name, move := range map[string]string{
		"an overlap":          `{"itemId":"fuel","fromGrid":"deck","toGrid":"deck","updates":[{"itemId":"fuel","grid":"deck","col":3,"row":2,"width":2,"height":1}]}`,
		"outside the grid":    `{"itemId":"fuel","fromGrid":"deck","toGrid":"deck","updates":[{"itemId":"fuel","grid":"deck","col":4,"row":1,"width":2,"height":1}]}`,
		"column 0":            `{"itemId":"crew","fromGrid":"shelf","toGrid":"shelf","updates":[{"itemId":"crew","grid":"shelf","col":0,"row":3,"width":1,"height":1}]}`,
		"no width":            `{"itemId":"crew","grid":"shelf","updates":[{"itemId":"crew","grid":"shelf","col":1,"row":2,"width":0,"height":1}]}`,
		"too tall":            `{"itemId":"crew","grid":"shelf","updates":[{"itemId":"crew","grid":"shelf","col":1,"row":2,"width":1,"height":6}]}`,
		"past the row cap":    `{"itemId":"crew","fromGrid":"shelf","toGrid":"shelf","updates":[{"itemId":"crew","grid":"shelf","col":1,"row":31,"width":1,"height":1}]}`,
		"taller than the cap": `{"itemId":"crew","grid":"shelf","updates":[{"itemId":"crew","grid":"shelf","col":1,"row":28,"width":1,"height":4}]}`,
		"a huge column":       `{"itemId":"fuel","fromGrid":"deck","toGrid":"deck","updates":[{"itemId":"fuel","grid":"deck","col":` + huge + `,"row":1,"width":2,"height":1}]}`,
		"a huge width":        `{"itemId":"crew","grid":"shelf","updates":[{"itemId":"crew","grid":"shelf","col":2,"row":2,"width":` + huge + `,"height":1}]}`,
		"a huge row":          `{"itemId":"thrust","fromGrid":"deck","toGrid":"deck","updates":[{"itemId":"thrust","grid":"deck","col":1,"row":` + huge + `,"width":2,"height":2}]}`,
		"a huge height":       `{"itemId":"crew","grid":"shelf","updates":[{"itemId":"crew","grid":"shelf","col":1,"row":2,"width":1,"height":` + huge + `}]}`,
		"an unknown tile":     `{"itemId":"warp","fromGrid":"deck","toGrid":"deck","updates":[{"itemId":"warp","grid":"deck","col":1,"row":4,"width":1,"height":1}]}`,
		"an unknown update":   `{"itemId":"fuel","fromGrid":"deck","toGrid":"deck","updates":[{"itemId":"fuel","grid":"deck","col":3,"row":4,"width":2,"height":1},{"itemId":"warp","grid":"deck","col":1,"row":5,"width":1,"height":1}]}`,
		"an unknown grid":     `{"itemId":"crew","fromGrid":"shelf","toGrid":"hold","updates":[{"itemId":"crew","grid":"hold","col":1,"row":1,"width":1,"height":1}]}`,
		"the wrong grid":      `{"itemId":"crew","fromGrid":"deck","toGrid":"deck","updates":[{"itemId":"crew","grid":"deck","col":1,"row":4,"width":1,"height":1}]}`,
		"an update elsewhere": `{"itemId":"fuel","fromGrid":"deck","toGrid":"deck","updates":[{"itemId":"fuel","grid":"deck","col":3,"row":4,"width":2,"height":1},{"itemId":"crew","grid":"shelf","col":1,"row":4,"width":1,"height":1}]}`,
		"no update of itself": `{"itemId":"fuel","fromGrid":"deck","toGrid":"deck","updates":[{"itemId":"speed","grid":"deck","col":3,"row":4,"width":2,"height":1}]}`,
		"no updates":          `{"itemId":"fuel","fromGrid":"deck","toGrid":"deck","updates":[]}`,
		"not json":            `"fuel"`,
	} {
		if got, err := bentoDashboard.arrange(bentoStart, json.RawMessage(move)); err == nil {
			t.Errorf("%s: accepted, %q", name, got)
		}
	}
	for _, bad := range []string{
		"", "deck", "shelf deck", "deck thrust.1.1.2.2 shelf",
		"deck thrust.1.1.2 fuel.3.1.2.1 speed.3.2.2.1 shelf shields.1.1.2.1 crew.1.2.1.1",
		bentoStart + " crew.2.2.1.1",
		strings.Replace(bentoStart, "crew.1.2.1.1", "crew.1.2.1.1."+huge, 1),
		strings.Replace(bentoStart, "fuel.3.1.2.1", "fuel."+huge+".1.2.1", 1),
		strings.Replace(bentoStart, "thrust.1.1.2.2", "thrust.1."+huge+".2.2", 1),
		strings.Replace(bentoStart, "crew.1.2.1.1", "crew.2.2.99999999999999999999.1", 1),
		strings.Replace(bentoStart, "crew.1.2.1.1", "warp.1.2.1.1", 1),
		strings.Replace(bentoStart, " shelf ", " hold ", 1),
		"thrust.1.1.2.2 deck fuel.3.1.2.1 speed.3.2.2.1 shelf shields.1.1.2.1 crew.1.2.1.1",
	} {
		if _, err := bentoDashboard.parse(bad); err == nil {
			t.Errorf("%q: accepted", bad)
		}
	}
}

// The gallery card has its own tiles and fewer rows.
func TestArrangeBentoCard(t *testing.T) {
	const start = "deck thrust.1.1.2.2 fuel.3.1.1.1 crew.3.2.1.1"
	got, err := bentoCard.arrange(start, json.RawMessage(`{"itemId":"crew","fromGrid":"deck","toGrid":"deck","updates":[{"itemId":"crew","grid":"deck","col":1,"row":3,"width":1,"height":1}]}`))
	if want := "deck thrust.1.1.2.2 fuel.3.1.1.1 crew.1.3.1.1"; err != nil || got != want {
		t.Errorf("a move: %q %v, want %q", got, err, want)
	}
	for name, move := range map[string]string{
		"past the row cap": `{"itemId":"crew","fromGrid":"deck","toGrid":"deck","updates":[{"itemId":"crew","grid":"deck","col":1,"row":7,"width":1,"height":1}]}`,
		"a dashboard tile": `{"itemId":"speed","fromGrid":"deck","toGrid":"deck","updates":[{"itemId":"speed","grid":"deck","col":1,"row":3,"width":1,"height":1}]}`,
		"the shelf":        `{"itemId":"crew","fromGrid":"deck","toGrid":"shelf","updates":[{"itemId":"crew","grid":"shelf","col":1,"row":1,"width":1,"height":1}]}`,
	} {
		if got, err := bentoCard.arrange(start, json.RawMessage(move)); err == nil {
			t.Errorf("%s: accepted, %q", name, got)
		}
	}
	if _, err := bentoCard.parse(bentoStart); err == nil {
		t.Error("the card took the dashboard's state")
	}
}
