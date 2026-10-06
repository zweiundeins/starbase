package web

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestArrangeSortableTree(t *testing.T) {
	const state = "earth(moon phobos) mars(deimos) jupiter(europa io) callisto ceres()"
	// Eight folders deep, the most a state may nest.
	const deep = "earth(mars(jupiter(saturn(uranus(neptune(mercury(venus()))))))) ceres() moon"
	move := func(item, from, to, before string) string {
		b, _ := json.Marshal(map[string]string{"itemId": item, "fromParent": from, "toParent": to, "before": before})
		return string(b)
	}
	for _, tc := range []struct{ state, move, want string }{
		{state, move("phobos", "earth", "mars", "deimos"), "earth(moon) mars(phobos deimos) jupiter(europa io) callisto ceres()"},
		{state, move("phobos", "earth", "mars", ""), "earth(moon) mars(deimos phobos) jupiter(europa io) callisto ceres()"},
		{state, move("callisto", "", "jupiter", ""), "earth(moon phobos) mars(deimos) jupiter(europa io callisto) ceres()"},
		{state, move("io", "jupiter", "jupiter", "europa"), "earth(moon phobos) mars(deimos) jupiter(io europa) callisto ceres()"},
		{state, move("mars", "", "", "earth"), "mars(deimos) earth(moon phobos) jupiter(europa io) callisto ceres()"},
		{state, move("moon", "earth", "ceres", ""), "earth(phobos) mars(deimos) jupiter(europa io) callisto ceres(moon)"},
		{state, move("jupiter", "", "earth", "moon"), "earth(jupiter(europa io) moon phobos) mars(deimos) callisto ceres()"},
		{deep, move("moon", "", "venus", ""), "earth(mars(jupiter(saturn(uranus(neptune(mercury(venus(moon)))))))) ceres()"},
	} {
		got, err := arrangeSortableTree(tc.state, json.RawMessage(tc.move))
		if err != nil || got != tc.want {
			t.Errorf("%s: %q %v, want %q", tc.move, got, err, tc.want)
		}
	}
	for _, tc := range []struct{ why, state, move, err string }{
		{"not where the move says it is", state, move("phobos", "mars", "mars", ""), "is not in"},
		{"not in the tree", state, move("pluto", "", "", ""), "is not in"},
		{"not a body", state, move("nowhere", "", "", ""), "is not in"},
		{"into itself", state, move("earth", "", "earth", ""), "into itself"},
		{"into its own child", "earth(jupiter(io()))", move("jupiter", "earth", "io", ""), "into itself"},
		{"a file is no folder", state, move("callisto", "", "moon", ""), "is not a folder"},
		{"a folder that isn't in the tree", state, move("callisto", "", "saturn", ""), "is not a folder"},
		{"before a node of another folder", state, move("io", "jupiter", "mars", "europa"), "is not in the list"},
		{"before itself", state, move("io", "jupiter", "jupiter", "io"), "can't move"},
		{"a ninth folder deep", deep, move("ceres", "", "venus", ""), "too deep"},
		{"a bad state", "earth(moon", move("moon", "earth", "", ""), "isn't closed"},
		{"a state that isn't bodies", "earth(nowhere)", move("earth", "", "", ""), "not a list of bodies"},
		{"a move that isn't JSON", state, "earth", "invalid character"},
	} {
		got, err := arrangeSortableTree(tc.state, json.RawMessage(tc.move))
		if err == nil || !strings.Contains(err.Error(), tc.err) {
			t.Errorf("%s: %q %v, want an error with %q", tc.why, got, err, tc.err)
		}
	}
	for _, tc := range []struct{ state, err string }{
		{"earth(moon", "isn't closed"},
		{"earth)", "closed twice"},
		{"(moon)", "needs a name"},
		{"earth(moon) earth()", "not a list of bodies"},
		{"earth(nowhere)", "not a list of bodies"},
		{"earth(mars(jupiter(saturn(uranus(neptune(mercury(venus(ceres()))))))))", "too deep"},
	} {
		if _, err := parseTree(tc.state); err == nil || !strings.Contains(err.Error(), tc.err) {
			t.Errorf("parseTree(%q): %v, want an error with %q", tc.state, err, tc.err)
		}
	}
	for _, s := range []string{state, deep} {
		if got, err := parseTree(s); err != nil || formatTree(got) != s {
			t.Errorf("round trip: %q %v, want %q", formatTree(got), err, s)
		}
	}
}
