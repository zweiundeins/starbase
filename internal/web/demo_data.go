package web

import (
	"encoding/json"
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
// branches accumulate). With Accept: application/json, they return the list.
// &delay=<ms> (up to 1500) makes loading states visible in demos.
//
// Items: {id, value, label, description, icon, kind, lazy}: value is the id
// (select options), lazy means it has children (tree items).

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
	if strings.Contains(r.Header.Get("Accept"), "application/json") {
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
