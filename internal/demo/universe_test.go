package demo

import "testing"

func TestUniverse(t *testing.T) {
	bodies := Universe()
	ids := map[string]bool{}
	for _, b := range bodies {
		if b.ID == "" || ids[b.ID] {
			t.Errorf("id %q is empty or repeated (%s)", b.ID, b.Name)
		}
		if b.Parent != "" && !ids[b.Parent] {
			t.Errorf("%s comes before its parent %s", b.ID, b.Parent)
		}
		if _, ok := Icons[b.Kind]; !ok {
			t.Errorf("%s: unknown kind %q", b.ID, b.Kind)
		}
		ids[b.ID] = true
	}
	if len(bodies) < 100 || !ids["jupiter"] || !ids["europa"] || !ids["orion"] || !ids["betelgeuse"] || !ids["trappist1h"] {
		t.Errorf("%d bodies, missing expected ones", len(bodies))
	}
	if Version() != Version() {
		t.Error("the version must be stable")
	}
	if ID("Barnard's Star") != "barnardsstar" || Fold("Boötes") != "bootes" {
		t.Error("ID/Fold")
	}
}
