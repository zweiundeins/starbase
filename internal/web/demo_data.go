package web

import (
	"encoding/json"
	"fmt"
	"html"
	"net/http"
	"regexp"
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
//
// They answer Datastar requests with a signal patch into the signal named by
// &into= (default _tree and _found): the top level and search results as a
// list, children as {"<parent>": [...]} (patches merge, so a lazy tree's
// branches accumulate). Other clients sending Accept: application/json get
// the list as JSON.
// &delay=<ms> (up to 1500) makes loading states visible in demos.
//
// Items: {id, value, label, description, icon, kind, lazy}: value is the id
// (select options), lazy means it has children (tree items).
//
//	GET /demo/data/list?id=<host id>&offset=&count=   a window of a million-item list
//
// patches an sb-virtual-scroll instead (see demoList).

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
		b.WriteString(`<div slot="header" aria-hidden="true"><span>Star</span> <span>Class</span> <span>Constellation</span> <span>Brightness</span> <span>Distance</span></div>`)
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
	st := demo.Star(i)
	return "", fmt.Sprintf(`<b>%s</b> <span>%s</span> <span>%s</span> <span>%.2f mag</span> <span>%s ly</span>`,
		st.Name, st.Class, html.EscapeString(st.Constellation), st.Magnitude, thousands(st.Distance))
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
