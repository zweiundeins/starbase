package web_test

import (
	"context"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"strconv"
	"strings"
	"testing"
	"time"

	"starbase/internal/commands"
	"starbase/internal/demo"
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
	// &cells=rich: the name links to the star, the distance has its unit, and
	// only a star the naked eye can see has a badge on its magnitude.
	req, _ := http.NewRequest("GET", ts.URL+"/demo/data/rows?count=500&cells=rich", nil)
	req.Header.Set("Accept", "application/json")
	res, err := c.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	var rich struct{ Rows []map[string]any }
	json.NewDecoder(res.Body).Decode(&rich)
	res.Body.Close()
	badges, plain := 0, 0
	for _, r := range rich.Rows {
		name, _ := r["name"].(map[string]any)
		dist, _ := r["distance"].(map[string]any)
		if _, ok := name["value"].(string); !ok || name["href"] != fmt.Sprintf("#star-%v", r["id"]) || dist["suffix"] != "ly" || dist["value"] == nil || r["class"] == nil {
			t.Fatalf("rich row: %v", r)
		}
		switch m := r["magnitude"].(type) {
		case map[string]any:
			if m["tone"] != "info" || m["value"].(float64) > 6 {
				t.Errorf("a badge on magnitude %v", m)
			}
			badges++
		case float64:
			if m <= 6 {
				t.Errorf("no badge on magnitude %v", m)
			}
			plain++
		}
	}
	if len(rich.Rows) != 500 || badges == 0 || plain == 0 {
		t.Errorf("rich: %d rows, %d badges, %d plain magnitudes", len(rich.Rows), badges, plain)
	}

	// Datastar: a patch into the signal named by into.
	_, body := get(t, c, ts.URL+"/demo/data/rows?offset=5&count=2&key=name&into=_stars")
	if !strings.Contains(body, "datastar-patch-signals") || !strings.Contains(body, `"_stars":{`) || !strings.Contains(body, `"offset":5`) || !strings.Contains(body, `"sort":{"dir":"asc","key":"name"}`) {
		t.Errorf("patch:\n%.400s", body)
	}
}

func TestDemoRowsExport(t *testing.T) {
	ts, c, bus, _ := newServerBus(t)
	if err := bus.Exec(context.Background(), commands.SeedStars{}); err != nil {
		t.Fatal(err)
	}
	// fresh has no cookie jar: a new session every time. With a new address
	// (X-Real-IP, believed from loopback) too, only the end meets the limits.
	ip := 0
	addr := func() string { ip++; return fmt.Sprintf("192.0.2.%d", ip) }
	exportFrom := func(client *http.Client, from, query string) (*http.Response, string) {
		t.Helper()
		req, _ := http.NewRequest("GET", ts.URL+"/demo/data/rows/export?"+query, nil)
		req.Header.Set("X-Real-IP", from)
		res, err := client.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		defer res.Body.Close()
		b, _ := io.ReadAll(res.Body)
		return res, string(b)
	}
	fresh := &http.Client{}
	export := func(client *http.Client, query string) (*http.Response, string) {
		t.Helper()
		return exportFrom(client, addr(), query)
	}
	readCSV := func(body string) [][]string {
		t.Helper()
		recs, err := csv.NewReader(strings.NewReader(body)).ReadAll()
		if err != nil {
			t.Fatal(err)
		}
		return recs
	}

	res, body := export(fresh, "format=csv&key=distance&dir=desc")
	if res.StatusCode != 200 || res.Header.Get("Content-Type") != "text/csv; charset=utf-8" || res.Header.Get("Content-Disposition") != `attachment; filename="stars.csv"` ||
		res.Header.Get("Cache-Control") != "public, max-age=300" || res.Header.Get("Access-Control-Allow-Origin") != "*" {
		t.Errorf("csv: %d %v", res.StatusCode, res.Header)
	}
	if !res.Uncompressed {
		t.Error("csv: not compressed")
	}
	recs := readCSV(body)
	if len(recs) != 100_001 || strings.Join(recs[0], ",") != "id,name,class,temp,constellation,distance,magnitude,planets" {
		t.Fatalf("csv: %d records, header %v", len(recs), recs[0])
	}
	var window struct{ Rows []struct{ ID int } }
	req, _ := http.NewRequest("GET", ts.URL+"/demo/data/rows?offset=0&count=3&key=distance&dir=desc", nil)
	req.Header.Set("Accept", "application/json")
	wres, err := c.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	json.NewDecoder(wres.Body).Decode(&window)
	wres.Body.Close()
	for i, r := range window.Rows {
		if recs[i+1][0] != strconv.Itoa(r.ID) {
			t.Errorf("csv row %d is star %s, the table's window has %d", i, recs[i+1][0], r.ID)
		}
	}

	_, body = export(fresh, "format=csv&columns=name,id")
	if recs := readCSV(body); strings.Join(recs[0], ",") != "name,id" || recs[1][0] != demo.StarAt(0).Name || recs[1][1] != "1" || len(recs[1]) != 2 {
		t.Errorf("columns: %v, %v", recs[0], recs[1])
	}

	res, body = export(fresh, "format=json&key=name&columns=id,distance&selected=5,3,999999999")
	if res.StatusCode != 200 || res.Header.Get("Content-Type") != "application/json" || res.Header.Get("Content-Disposition") != `attachment; filename="stars.json"` {
		t.Errorf("json: %d %v", res.StatusCode, res.Header)
	}
	var stars []map[string]any
	if err := json.Unmarshal([]byte(body), &stars); err != nil {
		t.Fatalf("json: %v\n%s", err, body)
	}
	if len(stars) != 2 || len(stars[0]) != 2 || !strings.HasPrefix(body, `[{"id":`) {
		t.Errorf("selected, two columns: %s", body)
	}
	a, b := demo.StarAt(2), demo.StarAt(4)
	if a.Name > b.Name {
		a, b = b, a
	}
	if len(stars) == 2 && (stars[0]["id"] != float64(a.ID) || stars[1]["id"] != float64(b.ID) || stars[0]["distance"] != a.Distance) {
		t.Errorf("selected in name order: %v, want %d then %d", stars, a.ID, b.ID)
	}
	if _, body := export(fresh, "format=json&selected=999999999"); body != "[]\n" {
		t.Errorf("nothing selected that exists: %q", body)
	}

	many := strings.TrimSuffix(strings.Repeat("1,", 1001), ",")
	for _, q := range []string{"", "format=xml", "format=csv&columns=name,nope", "format=csv&columns=name,", "format=csv&columns=name,id,name", "format=csv&selected=1,x", "format=csv&selected=" + many} {
		if res, _ := export(fresh, q); res.StatusCode != http.StatusBadRequest {
			t.Errorf("%.60s: %d, want 400", q, res.StatusCode)
		}
	}
	if res, _ := export(fresh, "format=csv&selected="+strings.TrimSuffix(many, ",1")); res.StatusCode != 200 {
		t.Errorf("1000 ids: %d", res.StatusCode)
	}

	// One session: three at once, then one every five seconds.
	for i := range 4 {
		if res, _ := export(c, "format=csv&selected=1"); (res.StatusCode == 200) != (i < 3) {
			t.Errorf("session download %d: %d", i+1, res.StatusCode)
		}
	}
	// One address: ten at once, across sessions.
	from := addr()
	for i := range 11 {
		if res, _ := exportFrom(fresh, from, "format=csv&selected=1"); (res.StatusCode == 200) != (i < 10) {
			t.Errorf("address download %d: %d", i+1, res.StatusCode)
		}
	}

	// Two stream at a time: while two downloads stall, a third is turned
	// away, and closing them frees their places.
	stalling := &http.Client{Transport: &http.Transport{
		DisableCompression: true,
		DialContext: func(ctx context.Context, network, address string) (net.Conn, error) {
			conn, err := (&net.Dialer{}).DialContext(ctx, network, address)
			if err == nil {
				err = conn.(*net.TCPConn).SetReadBuffer(4096) // the server's writes block soon
			}
			return conn, err
		},
	}}
	var stalled []io.Closer
	for range 2 {
		req, _ := http.NewRequest("GET", ts.URL+"/demo/data/rows/export?format=json", nil)
		req.Header.Set("X-Real-IP", addr())
		res, err := stalling.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() { res.Body.Close() })
		stalled = append(stalled, res.Body)
		if res.StatusCode != 200 {
			t.Fatalf("a stalling download: %d", res.StatusCode)
		}
	}
	if res, _ := export(fresh, "format=csv&selected=1"); res.StatusCode != http.StatusServiceUnavailable || res.Header.Get("Retry-After") != "5" {
		t.Errorf("a third download at a time: %d, Retry-After %q", res.StatusCode, res.Header.Get("Retry-After"))
	}
	for _, b := range stalled {
		b.Close()
	}
	status := 0
	for try := 0; try < 100 && status != 200; try++ {
		time.Sleep(20 * time.Millisecond)
		res, _ := export(fresh, "format=csv&selected=1")
		status = res.StatusCode
	}
	if status != 200 {
		t.Errorf("after the stalled downloads closed: %d", status)
	}
}
