package web_test

import (
	"strings"
	"testing"
)

func TestDemoSearch(t *testing.T) {
	ts, c := newServer(t)
	_, body := get(t, c, ts.URL+"/demo/search?q=cru")
	if !strings.Contains(body, "datastar-patch-signals") || !strings.Contains(body, `"label":"Acrux"`) || !strings.Contains(body, `"description":"Crux"`) {
		t.Errorf("q=cru:\n%s", body)
	}
	// Accents fold: "bootes" finds stars in Boötes; names that start with the query come first.
	if _, body := get(t, c, ts.URL+"/demo/search?q=bootes"); !strings.Contains(body, `"Arcturus"`) {
		t.Errorf("q=bootes:\n%s", body)
	}
	if _, body := get(t, c, ts.URL+"/demo/search?q=al"); strings.Index(body, `"Aldebaran"`) > strings.Index(body, `"Rigil Kentaurus"`) && strings.Contains(body, `"Rigil Kentaurus"`) {
		t.Errorf("prefix matches should come first:\n%s", body)
	}
	if _, body := get(t, c, ts.URL+"/demo/search?q=zzz"); !strings.Contains(body, `"_found":[]`) {
		t.Errorf("no match:\n%s", body)
	}
}
