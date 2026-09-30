package web

import (
	"bufio"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"html"
	"math"
	"net/http"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/starfederation/datastar-go/datastar"

	"starbase/internal/demo"
	"starbase/internal/queries"
)

// Generic demo endpoints over the example dataset (internal/demo), for any
// component's docs and playgrounds. Stateless reads.
//
//	GET /demo/data/children?parent=<id>   a body's children ("" or no parent: the top level)
//	GET /demo/data/search?q=…&kind=a,b&limit=8
//	GET /demo/data/rows?offset=0&count=100&key=name&dir=desc   a window of the star catalog, in an order
//
// They answer Datastar requests with a signal patch into the signal named by
// &into= (default _tree, _found and _rows): the top level and search results
// as a list, children as {"<parent>": [...]} (patches merge, so a lazy tree's
// branches accumulate), rows as {rows, offset, total, sort: {key, dir}}.
// Other clients sending Accept: application/json get the list (or the
// window) as JSON.
// &delay=<ms> (up to 1500) makes loading states visible in demos.
//
// Items: {id, value, label, description, icon, kind, lazy}: value is the id
// (select options), lazy means it has children (tree items).
//
//	GET /demo/data/list?id=<host id>&offset=&count=   a window of a million-item list
//
// patches an sb-virtual-scroll instead (see demoList).
//
//	GET /demo/data/rows/export?format=csv&key=name&dir=desc&columns=name,distance&selected=1,2
//
// downloads the star catalog as CSV or JSON, rate limited (see demoRowsExport).

var signalNameRe = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]{0,40}$`)

type demoItem struct {
	ID          string `json:"id"`
	Value       string `json:"value"`
	Label       string `json:"label"`
	Description string `json:"description"`
	Icon        string `json:"icon"`
	Kind        string `json:"kind"`
	Lazy        bool   `json:"lazy,omitempty"`
}

func demoItems(bodies []queries.DemoBody) []demoItem {
	out := make([]demoItem, 0, len(bodies))
	for _, b := range bodies {
		out = append(out, demoItem{ID: b.ID, Value: b.ID, Label: b.Name, Description: b.Detail, Icon: demo.Icons[b.Kind], Kind: b.Kind, Lazy: b.Children > 0})
	}
	return out
}

// demoChildren answers with the children of ?parent= (the top level without one).
func (s *Server) demoChildren(w http.ResponseWriter, r *http.Request) {
	parent := r.URL.Query().Get("parent")
	var bodies []queries.DemoBody
	if err := s.q.View(r.Context(), func(rd *queries.Reader) (err error) {
		bodies, err = rd.DemoChildren(r.Context(), parent)
		return
	}); err != nil {
		s.fail(w, r, err)
		return
	}
	items := demoItems(bodies)
	var patch any = items
	if parent != "" {
		patch = map[string]any{parent: items}
	}
	s.demoAnswer(w, r, "_tree", items, patch)
}

// demoSearch answers with the bodies whose name matches ?q=, at most ?limit=.
func (s *Server) demoSearch(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	limit, _ := strconv.Atoi(q.Get("limit"))
	limit = min(max(limit, 1), 50)
	if q.Get("limit") == "" {
		limit = 8
	}
	var kinds []string
	for _, k := range strings.Split(q.Get("kind"), ",") {
		if k = strings.TrimSpace(k); k != "" {
			kinds = append(kinds, k)
		}
	}
	var bodies []queries.DemoBody
	if err := s.q.View(r.Context(), func(rd *queries.Reader) (err error) {
		bodies, err = rd.DemoSearch(r.Context(), q.Get("q"), kinds, limit)
		return
	}); err != nil {
		s.fail(w, r, err)
		return
	}
	items := demoItems(bodies)
	s.demoAnswer(w, r, "_found", items, items)
}

// demoRows answers a table's window and sort requests (sb-data-table's
// sb-window and sb-sort) with ?count= stars from ?offset= (at most 500) in the
// order asked for (?key=, ?dir=), and says which order it used.
func (s *Server) demoRows(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	offset, _ := strconv.Atoi(q.Get("offset"))
	count, err := strconv.Atoi(q.Get("count"))
	if err != nil {
		count = 100
	}
	offset, count = max(offset, 0), min(max(count, 1), 500)
	key, dir := q.Get("key"), q.Get("dir")
	if !queries.DemoStarSortable(key) {
		key, dir = "", ""
	} else if dir != "desc" {
		dir = "asc"
	}
	var stars []demo.Star
	var total int
	if err := s.q.View(r.Context(), func(rd *queries.Reader) (err error) {
		stars, total, err = rd.DemoStars(r.Context(), key, dir == "desc", offset, count)
		return
	}); err != nil {
		s.fail(w, r, err)
		return
	}
	window := map[string]any{"rows": stars, "offset": offset, "total": total, "sort": map[string]string{"key": key, "dir": dir}}
	s.demoAnswer(w, r, "_rows", window, window)
}

// demoStarColumns are the star catalog's columns an export can hold, by
// their JSON names.
var demoStarColumns = []string{"id", "name", "class", "temp", "constellation", "distance", "magnitude", "planets"}

// demoStarField is the value of st's column key (one of demoStarColumns).
func demoStarField(st demo.Star, key string) any {
	switch key {
	case "id":
		return st.ID
	case "name":
		return st.Name
	case "class":
		return st.Class
	case "temp":
		return st.Temp
	case "constellation":
		return st.Constellation
	case "distance":
		return st.Distance
	case "magnitude":
		return st.Magnitude
	case "planets":
		return st.Planets
	}
	return nil
}

// demoRowsExport answers sb-data-table's sb-export with a download of the star
// catalog: ?format=csv or json, in the order asked for (?key=, ?dir=), the
// ?columns= in their order (default: all), and with ?selected= only those
// stars (at most 1000 ids). The rows stream from one read transaction, so the
// whole catalog never sits in memory, and go out compressed.
func (s *Server) demoRowsExport(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	format := q.Get("format")
	if format != "csv" && format != "json" {
		http.Error(w, "format must be csv or json", http.StatusBadRequest)
		return
	}
	key, dir := q.Get("key"), q.Get("dir")
	if !queries.DemoStarSortable(key) {
		key, dir = "", ""
	} else if dir != "desc" {
		dir = "asc"
	}
	columns := demoStarColumns
	if c := q.Get("columns"); c != "" {
		columns = strings.Split(c, ",")
		for i, k := range columns {
			if !slices.Contains(demoStarColumns, k) {
				http.Error(w, "unknown column "+strconv.Quote(k), http.StatusBadRequest)
				return
			}
			if slices.Contains(columns[:i], k) {
				http.Error(w, "column "+strconv.Quote(k)+" twice", http.StatusBadRequest)
				return
			}
		}
	}
	var ids []int
	if sel := q.Get("selected"); sel != "" {
		list := strings.Split(sel, ",")
		if len(list) > 1000 {
			http.Error(w, "at most 1000 selected ids", http.StatusBadRequest)
			return
		}
		for _, v := range list {
			id, err := strconv.Atoi(v)
			if err != nil {
				http.Error(w, "selected must be ids", http.StatusBadRequest)
				return
			}
			ids = append(ids, id)
		}
	}
	now := time.Now()
	if !s.exportLimit.allow(sessionID(r), 1, now) || !s.exportLimitIP.allow(clientIP(r), 1, now) {
		http.Error(w, "exporting too often", http.StatusTooManyRequests)
		return
	}
	// A download holds a read connection until it ends: slow ones must not
	// take them all from the pages, and one that stalls for a minute ends.
	select {
	case s.exports <- struct{}{}:
		defer func() { <-s.exports }()
	default:
		w.Header().Set("Retry-After", "5")
		http.Error(w, "too many downloads at once", http.StatusServiceUnavailable)
		return
	}
	setWriteDeadline(w, r, time.Now().Add(time.Minute))

	h := w.Header()
	h.Set("Access-Control-Allow-Origin", "*") // public; used from the playground sandbox
	h.Set("Cache-Control", "public, max-age=300")
	h.Set("Content-Disposition", `attachment; filename="stars.`+format+`"`)
	// Each star is written as it comes from the cursor.
	var write func(demo.Star) error
	var end func() error
	if format == "csv" {
		h.Set("Content-Type", "text/csv; charset=utf-8")
		cw := csv.NewWriter(w)
		cw.Write(columns)
		row := make([]string, len(columns))
		write = func(st demo.Star) error {
			for i, k := range columns {
				switch v := demoStarField(st, k).(type) {
				case float64:
					row[i] = strconv.FormatFloat(v, 'f', -1, 64) // 1234567, never 1.234567e+06
				default:
					row[i] = fmt.Sprint(v)
				}
			}
			return cw.Write(row)
		}
		end = func() error { cw.Flush(); return cw.Error() }
	} else {
		h.Set("Content-Type", "application/json")
		bw := bufio.NewWriter(w)
		bw.WriteString("[")
		sep := ""
		write = func(st demo.Star) error {
			obj := []byte(sep + "{")
			for i, k := range columns {
				if i > 0 {
					obj = append(obj, ',')
				}
				v, err := json.Marshal(demoStarField(st, k))
				if err != nil {
					return err
				}
				obj = append(append(strconv.AppendQuote(obj, k), ':'), v...)
			}
			sep = ",\n"
			_, err := bw.Write(append(obj, '}'))
			return err
		}
		end = func() error { bw.WriteString("]\n"); return bw.Flush() }
	}
	n := 0
	err := s.q.View(r.Context(), func(rd *queries.Reader) error {
		return rd.EachDemoStar(r.Context(), key, dir == "desc", ids, func(st demo.Star) error {
			if n++; n%1000 == 0 {
				setWriteDeadline(w, r, time.Now().Add(time.Minute))
			}
			return write(st)
		})
	})
	if err == nil {
		err = end()
	}
	if err != nil {
		// A file cut short must not look complete: the download fails instead.
		if r.Context().Err() == nil {
			s.log.Error("export failed", "err", err)
		}
		panic(http.ErrAbortHandler)
	}
}

// demoAnswer writes the list as JSON, or patches it into the requested signal.
func (s *Server) demoAnswer(w http.ResponseWriter, r *http.Request, defaultSignal string, list, patch any) {
	w.Header().Set("Access-Control-Allow-Origin", "*") // public; used from the playground sandbox
	if ms, _ := strconv.Atoi(r.URL.Query().Get("delay")); ms > 0 {
		select {
		case <-time.After(time.Duration(min(ms, 1500)) * time.Millisecond):
		case <-r.Context().Done():
			return
		}
	}
	// Datastar's own requests accept JSON too: they always get the patch.
	if r.Header.Get("Datastar-Request") == "" && strings.Contains(r.Header.Get("Accept"), "application/json") {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Cache-Control", "public, max-age=300")
		json.NewEncoder(w).Encode(list)
		return
	}
	into := r.URL.Query().Get("into")
	if into == "" {
		into = defaultSignal
	}
	if !signalNameRe.MatchString(into) {
		http.Error(w, "into must be a signal name", http.StatusBadRequest)
		return
	}
	datastar.NewSSE(w, r).MarshalAndPatchSignals(map[string]any{into: patch})
}

var elementIDRe = regexp.MustCompile(`^[A-Za-z][A-Za-z0-9_-]{0,63}$`)

// demoListKeep are the host attributes the page sets. This endpoint serves any
// page, so it sends only offset, total and the items, and the morph keeps these
// as the page has them. data-ignore-morph keeps the page's own frames, which
// know nothing of the window, away from the list.
const demoListKeep = "item-size columns buffer label role style class data-ignore-morph data-on:sb-window"

// demoList answers an sb-virtual-scroll's sb-window with ?count= items from
// ?offset= (at most 5000), patched into the host with the id ?id=. The list is
// the star catalog: its million stars, or the first ?total=. &header adds the
// column headings; &kind=pixels&columns=<n> sends pixels of a nebula n wide.
func (s *Server) demoList(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	id := q.Get("id")
	if !elementIDRe.MatchString(id) {
		http.Error(w, "id must be an element id", http.StatusBadRequest)
		return
	}
	total, _ := strconv.Atoi(q.Get("total"))
	if total <= 0 || total > demo.CatalogSize {
		total = demo.CatalogSize
	}
	offset, _ := strconv.Atoi(q.Get("offset"))
	count, _ := strconv.Atoi(q.Get("count"))
	offset = min(max(offset, 0), total)
	count = min(max(count, 0), 5000, total-offset)
	cols := 0
	if q.Get("kind") == "pixels" {
		cols, _ = strconv.Atoi(q.Get("columns"))
		cols = max(cols, 1)
	}

	// The whole host: the morph patches it like any other markup.
	var b strings.Builder
	fmt.Fprintf(&b, `<sb-virtual-scroll id="%s" offset="%d" total="%d" data-preserve-attr="%s">`, id, offset, total, demoListKeep)
	if q.Has("header") {
		b.WriteString(`<div slot="header" aria-hidden="true"><span>Star</span> <span>Class</span> <span>Constellation</span> <span>Brightness</span> <span>Distance</span> <span>Planets</span></div>`)
	}
	for i := offset; i < offset+count; i++ {
		// aria-posinset and aria-setsize: the item's place in the whole list.
		attrs, content := demoListItem(i, cols)
		fmt.Fprintf(&b, `<div role="listitem" aria-posinset="%d" aria-setsize="%d"%s>%s</div>`, i+1, total, attrs, content)
	}
	b.WriteString(`</sb-virtual-scroll>`)

	w.Header().Set("Access-Control-Allow-Origin", "*") // public; used from the playground sandbox
	sse := datastar.NewSSE(w, r, datastar.WithCompression(datastar.WithBrotli(datastar.WithBrotliLevel(5)), datastar.WithGzip()))
	sse.PatchElements(b.String())
}

// demoListItem is item i: a line of the star catalog, or with cols > 0 a pixel
// of a nebula cols pixels wide.
func demoListItem(i, cols int) (attrs, content string) {
	if cols > 0 {
		p := demo.Pixel(i%cols, i/cols)
		return fmt.Sprintf(` class="p%d" aria-label="%s"`, p, pixelNames[p]), ""
	}
	st := demo.StarAt(i)
	return "", fmt.Sprintf(`<b>%s</b> <span>%s</span> <span>%s</span> <span>%.2f mag</span> <span>%s ly</span> <span>%d</span>`,
		st.Name, st.Class, html.EscapeString(st.Constellation), st.Magnitude, thousands(int(math.Round(st.Distance))), st.Planets)
}

var pixelNames = [...]string{"Space", "Dust", "Gas", "Glow", "Core", "Star"}

// thousands formats 12345 as 12,345.
func thousands(n int) string {
	s := strconv.Itoa(n)
	for i := len(s) - 3; i > 0; i -= 3 {
		s = s[:i] + "," + s[i:]
	}
	return s
}
