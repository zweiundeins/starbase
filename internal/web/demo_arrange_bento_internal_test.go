package web

import (
	"encoding/json"
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
	} {
		got, err := arrangeBento(bentoStart, json.RawMessage(tc.move))
		if err != nil || got != tc.want {
			t.Errorf("%s: %q %v, want %q", tc.name, got, err, tc.want)
		}
	}
	for name, move := range map[string]string{
		"an overlap":          `{"itemId":"fuel","fromGrid":"deck","toGrid":"deck","updates":[{"itemId":"fuel","grid":"deck","col":3,"row":2,"width":2,"height":1}]}`,
		"outside the grid":    `{"itemId":"fuel","fromGrid":"deck","toGrid":"deck","updates":[{"itemId":"fuel","grid":"deck","col":4,"row":1,"width":2,"height":1}]}`,
		"too tall":            `{"itemId":"crew","grid":"shelf","updates":[{"itemId":"crew","grid":"shelf","col":1,"row":2,"width":1,"height":6}]}`,
		"an unknown tile":     `{"itemId":"warp","fromGrid":"deck","toGrid":"deck","updates":[{"itemId":"warp","grid":"deck","col":1,"row":4,"width":1,"height":1}]}`,
		"the wrong grid":      `{"itemId":"crew","fromGrid":"deck","toGrid":"deck","updates":[{"itemId":"crew","grid":"deck","col":1,"row":4,"width":1,"height":1}]}`,
		"an update elsewhere": `{"itemId":"fuel","fromGrid":"deck","toGrid":"deck","updates":[{"itemId":"fuel","grid":"deck","col":3,"row":4,"width":2,"height":1},{"itemId":"crew","grid":"shelf","col":1,"row":4,"width":1,"height":1}]}`,
		"no update of itself": `{"itemId":"fuel","fromGrid":"deck","toGrid":"deck","updates":[{"itemId":"speed","grid":"deck","col":3,"row":4,"width":2,"height":1}]}`,
	} {
		if got, err := arrangeBento(bentoStart, json.RawMessage(move)); err == nil {
			t.Errorf("%s: accepted, %q", name, got)
		}
	}
	for _, bad := range []string{"", "deck", "shelf deck", "deck thrust.1.1.2.2 shelf", "deck thrust.1.1.2 fuel.3.1.2.1 speed.3.2.2.1 shelf shields.1.1.2.1 crew.1.2.1.1", bentoStart + " crew.2.2.1.1"} {
		if _, err := parseBento(bad); err == nil {
			t.Errorf("%q: accepted", bad)
		}
	}
}

