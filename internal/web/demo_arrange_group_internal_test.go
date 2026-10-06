package web

import (
	"encoding/json"
	"testing"
)

func TestArrangeDragGroup(t *testing.T) {
	const state = "planets=earth,ceres,mars dwarfs=pluto,venus"
	for _, tc := range []struct{ state, move, want string }{
		{state, `{"itemId":"ceres","fromList":"planets","toList":"dwarfs","before":"pluto"}`, "planets=earth,mars dwarfs=ceres,pluto,venus"},
		{state, `{"itemId":"venus","fromList":"dwarfs","toList":"planets","before":""}`, "planets=earth,ceres,mars,venus dwarfs=pluto"},
		{state, `{"itemId":"mars","fromList":"planets","toList":"planets","before":"earth"}`, "planets=mars,earth,ceres dwarfs=pluto,venus"},
		{state, `{"itemId":"pluto","fromList":"dwarfs","toList":"planets","before":"earth"}`, "planets=pluto,earth,ceres,mars dwarfs=venus"},
		{"planets=earth dwarfs=", `{"itemId":"earth","fromList":"planets","toList":"dwarfs","before":""}`, "planets= dwarfs=earth"},
		{"planets= dwarfs=pluto", `{"itemId":"pluto","fromList":"dwarfs","toList":"planets","before":""}`, "planets=pluto dwarfs="},
	} {
		got, err := arrangeDragGroup(tc.state, json.RawMessage(tc.move))
		if err != nil || got != tc.want {
			t.Errorf("%s with %s: %q %v, want %q", tc.state, tc.move, got, err, tc.want)
		}
	}
	for _, bad := range []struct{ why, state, move string }{
		{"an unknown body", "planets=earth,nowhere", `{"itemId":"earth","fromList":"planets","toList":"planets","before":""}`},
		{"an unknown list", "moons=earth", `{"itemId":"earth","fromList":"moons","toList":"moons","before":""}`},
		{"a body in two lists", "planets=earth dwarfs=earth", `{"itemId":"earth","fromList":"planets","toList":"dwarfs","before":""}`},
		{"a list twice", "planets=earth planets=mars", `{"itemId":"earth","fromList":"planets","toList":"planets","before":"mars"}`},
		{"a list without =", "planets dwarfs=pluto", `{"itemId":"pluto","fromList":"dwarfs","toList":"planets","before":""}`},
		{"no lists", "", `{"itemId":"earth","fromList":"planets","toList":"planets","before":""}`},
		{"an empty id", "planets=earth,,mars dwarfs=pluto", `{"itemId":"earth","fromList":"planets","toList":"dwarfs","before":""}`},
		{"a trailing comma", "planets=earth, dwarfs=pluto", `{"itemId":"earth","fromList":"planets","toList":"dwarfs","before":""}`},
		{"a leading comma", "planets=,earth dwarfs=pluto", `{"itemId":"earth","fromList":"planets","toList":"dwarfs","before":""}`},
		{"an item from another list", state, `{"itemId":"pluto","fromList":"planets","toList":"dwarfs","before":""}`},
		{"an unknown item", state, `{"itemId":"io","fromList":"planets","toList":"dwarfs","before":""}`},
		{"no item", state, `{"itemId":"","fromList":"planets","toList":"dwarfs","before":""}`},
		{"an unknown source list", state, `{"itemId":"earth","fromList":"moons","toList":"dwarfs","before":""}`},
		{"an unknown target list", state, `{"itemId":"earth","fromList":"planets","toList":"moons","before":""}`},
		{"before an item of the other list", state, `{"itemId":"earth","fromList":"planets","toList":"dwarfs","before":"mars"}`},
		{"before an unknown item", state, `{"itemId":"earth","fromList":"planets","toList":"planets","before":"io"}`},
		{"before itself", state, `{"itemId":"earth","fromList":"planets","toList":"planets","before":"earth"}`},
		{"a move that isn't JSON", state, `{"itemId":`},
	} {
		if got, err := arrangeDragGroup(bad.state, json.RawMessage(bad.move)); err == nil {
			t.Errorf("%s (%s with %s): %q, want an error", bad.why, bad.state, bad.move, got)
		}
	}
}
