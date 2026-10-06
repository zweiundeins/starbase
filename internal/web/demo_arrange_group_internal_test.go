package web

import (
	"encoding/json"
	"testing"
)

func TestArrangeDragGroup(t *testing.T) {
	const state = "planets=earth,ceres,mars dwarfs=pluto,venus"
	for _, tc := range []struct{ move, want string }{
		{`{"itemId":"ceres","fromList":"planets","toList":"dwarfs","before":"pluto"}`, "planets=earth,mars dwarfs=ceres,pluto,venus"},
		{`{"itemId":"venus","fromList":"dwarfs","toList":"planets","before":""}`, "planets=earth,ceres,mars,venus dwarfs=pluto"},
		{`{"itemId":"mars","fromList":"planets","toList":"planets","before":"earth"}`, "planets=mars,earth,ceres dwarfs=pluto,venus"},
		{`{"itemId":"pluto","fromList":"dwarfs","toList":"planets","before":"earth"}`, "planets=pluto,earth,ceres,mars dwarfs=venus"},
	} {
		got, err := arrangeDragGroup(state, json.RawMessage(tc.move))
		if err != nil || got != tc.want {
			t.Errorf("%s: %q %v, want %q", tc.move, got, err, tc.want)
		}
	}
	if got, err := arrangeDragGroup("planets=earth dwarfs=", json.RawMessage(`{"itemId":"earth","fromList":"planets","toList":"dwarfs","before":""}`)); err != nil || got != "planets= dwarfs=earth" {
		t.Errorf("into an empty list: %q %v", got, err)
	}
	for _, bad := range []struct{ state, move string }{
		{"planets=earth,nowhere", `{"itemId":"earth","fromList":"planets","toList":"planets","before":""}`},
		{"moons=earth", `{"itemId":"earth","fromList":"moons","toList":"moons","before":""}`},
		{"planets=earth dwarfs=earth", `{"itemId":"earth","fromList":"planets","toList":"dwarfs","before":""}`},
		{state, `{"itemId":"pluto","fromList":"planets","toList":"dwarfs","before":""}`},
		{state, `{"itemId":"earth","fromList":"planets","toList":"moons","before":""}`},
		{state, `{"itemId":"earth","fromList":"planets","toList":"dwarfs","before":"mars"}`},
	} {
		if got, err := arrangeDragGroup(bad.state, json.RawMessage(bad.move)); err == nil {
			t.Errorf("%s with %s: %q, want an error", bad.state, bad.move, got)
		}
	}
}
