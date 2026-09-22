package web_test

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"
)

func TestDemoDataEndpoints(t *testing.T) {
	ts, c := newServer(t)
	// A Datastar request: a patch into the named signal, keyed by parent.
	_, body := get(t, c, ts.URL+"/demo/data/children?parent=mars&into=_sky")
	if !strings.Contains(body, "datastar-patch-signals") || !strings.Contains(body, `"_sky":{"mars":[`) || !strings.Contains(body, `"label":"Phobos"`) {
		t.Errorf("children:\n%s", body)
	}
	// Top level: a plain list, lazy where there are children.
	_, body = get(t, c, ts.URL+"/demo/data/children")
	if !strings.Contains(body, `"_tree":[{"id":"milkyway"`) || !strings.Contains(body, `"lazy":true`) {
		t.Errorf("top level:\n%.300s", body)
	}
	// JSON on request.
	req, _ := http.NewRequest("GET", ts.URL+"/demo/data/search?q=orion", nil)
	req.Header.Set("Accept", "application/json")
	res, err := c.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	var items []struct{ Value, Label, Kind string }
	json.NewDecoder(res.Body).Decode(&items)
	res.Body.Close()
	if len(items) == 0 || items[0].Label != "Orion" {
		t.Errorf("search json = %+v", items)
	}
	// Datastar's requests accept JSON as well, and must still get a patch.
	req, _ = http.NewRequest("GET", ts.URL+"/demo/data/search?q=orion", nil)
	req.Header.Set("Accept", "text/event-stream, application/json")
	req.Header.Set("Datastar-Request", "true")
	res, err = c.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	b, _ := io.ReadAll(res.Body)
	res.Body.Close()
	if !strings.Contains(string(b), "datastar-patch-signals") {
		t.Errorf("a Datastar request got:\n%.200s", b)
	}
	if r, _ := get(t, c, ts.URL+"/demo/data/search?q=a&into=x.y"); r.StatusCode != 400 {
		t.Errorf("a bad signal name = %d, want 400", r.StatusCode)
	}
}
