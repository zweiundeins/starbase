package web

import (
	"encoding/json"
	"fmt"
	"html"
	"net/http"
	"strings"
	"sync"

	"github.com/starfederation/datastar-go/datastar"

	"starbase/internal/demo"
)

// GET /demo/arrange/{kind} plays the server for the drag-and-drop demos of
// the components from PD rockets (sb-sortable-list and its siblings). The
// component's markup carries the arrangement the server rendered
// (data-state, ids from the example dataset); a move sends it with the
// event's detail, and the answer is the component rendered in the new
// arrangement, which the morph applies. Stateless: the arrangement travels
// with the request, so any page, and the playground's sandbox, can use it.
func (s *Server) demoArrange(w http.ResponseWriter, r *http.Request) {
	a, ok := arrangers[r.PathValue("kind")]
	if !ok {
		http.NotFound(w, r)
		return
	}
	var p struct {
		ID    string          `json:"id"`
		State string          `json:"state"`
		Move  json.RawMessage `json:"move"`
	}
	if err := datastar.ReadSignals(r, &p); err != nil || !elementIDRe.MatchString(p.ID) || len(p.State) > 4096 {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	state, err := a.arrange(p.State, p.Move)
	if err != nil {
		http.Error(w, err.Error(), http.StatusUnprocessableEntity)
		return
	}
	w.Header().Set("Access-Control-Allow-Origin", "*") // public; used from the playground sandbox
	datastar.NewSSE(w, r).PatchElements(a.render(p.ID, state))
}

// An arranger is one demo: how a move changes its arrangement, and its
// markup. A state is the data-state attribute: words a kind defines.
type arranger struct {
	arrange func(state string, move json.RawMessage) (string, error)
	render  func(id, state string) string
}

var arrangers = map[string]arranger{}

// bodies is the example dataset by id.
var bodies = sync.OnceValue(func() map[string]demo.Body {
	m := map[string]demo.Body{}
	for _, b := range demo.Universe() {
		m[b.ID] = b
	}
	return m
})

// ids reads a state's list of bodies: known, distinct, at most 40.
func ids(state string) ([]string, error) {
	list := strings.Fields(state)
	seen := map[string]bool{}
	for _, id := range list {
		if _, ok := bodies()[id]; !ok || seen[id] || len(list) > 40 {
			return nil, fmt.Errorf("%q is not a list of bodies", state)
		}
		seen[id] = true
	}
	return list, nil
}

// moveBefore takes item out of list and puts it back before another
// ("": at the end), as every sortable surface's move does.
func moveBefore(list []string, item, before string) ([]string, error) {
	out := make([]string, 0, len(list))
	found := false
	for _, id := range list {
		if id == item {
			found = true
		} else {
			out = append(out, id)
		}
	}
	if !found || item == before {
		return nil, fmt.Errorf("can't move %q before %q", item, before)
	}
	at := len(out)
	for i, id := range out {
		if id == before {
			at = i
		}
	}
	if before != "" && at == len(out) {
		return nil, fmt.Errorf("%q is not in the list", before)
	}
	return append(out[:at], append([]string{item}, out[at:]...)...), nil
}

// label is a body's icon and name, escaped.
func label(id string) string {
	b := bodies()[id]
	return html.EscapeString(demo.Icons[b.Kind] + " " + b.Name)
}

// arrangeOn is the handler a demo binds to its component's move event.
func arrangeOn(kind string) string {
	return "@get('/demo/arrange/" + kind + "', {payload: {id: el.id, state: el.dataset.state, move: evt.detail}})"
}
