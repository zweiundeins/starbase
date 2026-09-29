package web_test

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"testing"

	"starbase/internal/commands"
)

func TestDemoRows(t *testing.T) {
	ts, c, bus, _ := newServerBus(t)
	if err := bus.Exec(context.Background(), commands.SeedStars{}); err != nil {
		t.Fatal(err)
	}
	type window struct {
		Rows []struct {
			ID       int
			Name     string
			Distance float64
		}
		Offset, Total int
		Sort          struct{ Key, Dir string }
	}
	getJSON := func(query string) window {
		req, _ := http.NewRequest("GET", ts.URL+"/demo/data/rows?"+query, nil)
		req.Header.Set("Accept", "application/json")
		res, err := c.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		defer res.Body.Close()
		var w window
		if err := json.NewDecoder(res.Body).Decode(&w); err != nil {
			t.Fatal(err)
		}
		return w
	}
	w := getJSON("offset=99990&count=50&key=distance&dir=desc")
	if w.Total != 100_000 || w.Offset != 99990 || len(w.Rows) != 10 || w.Sort.Key != "distance" || w.Sort.Dir != "desc" {
		t.Errorf("the last window: total %d, offset %d, %d rows, sort %+v", w.Total, w.Offset, len(w.Rows), w.Sort)
	}
	if w.Rows[0].Distance < w.Rows[9].Distance {
		t.Errorf("not descending: %v … %v", w.Rows[0].Distance, w.Rows[9].Distance)
	}
	// count is capped; an unknown key is the catalog's order, and says so.
	if w := getJSON("count=100000&key=nope&dir=desc"); len(w.Rows) != 500 || w.Rows[0].ID != 1 || w.Sort.Key != "" || w.Sort.Dir != "" {
		t.Errorf("capped, unknown key: %d rows, first %d, sort %+v", len(w.Rows), w.Rows[0].ID, w.Sort)
	}
	// Datastar: a patch into the signal named by into.
	_, body := get(t, c, ts.URL+"/demo/data/rows?offset=5&count=2&key=name&into=_stars")
	if !strings.Contains(body, "datastar-patch-signals") || !strings.Contains(body, `"_stars":{`) || !strings.Contains(body, `"offset":5`) || !strings.Contains(body, `"sort":{"dir":"asc","key":"name"}`) {
		t.Errorf("patch:\n%.400s", body)
	}
}
