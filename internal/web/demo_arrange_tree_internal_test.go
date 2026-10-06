package web

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestArrangeSortableTree(t *testing.T) {
	const state = "earth(moon phobos) mars(deimos) jupiter(europa io) callisto ceres()"
	move := func(item, from, to, before string) string {
		b, _ := json.Marshal(map[string]string{"itemId": item, "fromParent": from, "toParent": to, "before": before})
		return string(b)
	}
	for _, tc := range []struct{ move, want string }{
		{move("phobos", "earth", "mars", "deimos"), "earth(moon) mars(phobos deimos) jupiter(europa io) callisto ceres()"},
		{move("phobos", "earth", "mars", ""), "earth(moon) mars(deimos phobos) jupiter(europa io) callisto ceres()"},
		{move("callisto", "", "jupiter", ""), "earth(moon phobos) mars(deimos) jupiter(europa io callisto) ceres()"},
		{move("io", "jupiter", "jupiter", "europa"), "earth(moon phobos) mars(deimos) jupiter(io europa) callisto ceres()"},
		{move("mars", "", "", "earth"), "mars(deimos) earth(moon phobos) jupiter(europa io) callisto ceres()"},
		{move("moon", "earth", "ceres", ""), "earth(phobos) mars(deimos) jupiter(europa io) callisto ceres(moon)"},
		{move("jupiter", "", "earth", "moon"), "earth(jupiter(europa io) moon phobos) mars(deimos) callisto ceres()"},
	} {
		got, err := arrangeSortableTree(state, json.RawMessage(tc.move))
		if err != nil || got != tc.want {
			t.Errorf("%s: %q %v, want %q", tc.move, got, err, tc.want)
		}
	}
	for _, bad := range []string{
		move("phobos", "mars", "mars", ""),      // not where the move says it is
		move("earth", "", "earth", ""),          // into itself
		move("callisto", "", "moon", ""),        // a file is no folder
		move("io", "jupiter", "mars", "europa"), // europa isn't in mars
		move("pluto", "", "", ""),               // not in the tree
	} {
		if got, err := arrangeSortableTree(state, json.RawMessage(bad)); err == nil {
			t.Errorf("%s: %q, want an error", bad, got)
		}
	}
	jupiterIntoIo := move("jupiter", "earth", "io", "")
	if _, err := arrangeSortableTree("earth(jupiter(io()))", json.RawMessage(jupiterIntoIo)); err == nil || !strings.Contains(err.Error(), "into itself") {
		t.Errorf("a folder into its own child: %v", err)
	}
	for _, bad := range []string{"earth(moon", "earth)", "(moon)", "earth(moon) earth()", "earth(nowhere)", "a(b(c(d(e(f(g(h(i(j()))))))))) "} {
		if _, err := parseTree(bad); err == nil {
			t.Errorf("parseTree(%q): no error", bad)
		}
	}
	if got, _ := parseTree(state); formatTree(got) != state {
		t.Errorf("round trip: %q", formatTree(got))
	}
}
