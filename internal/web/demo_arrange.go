package web

import (
	"encoding/json"
	"fmt"
	"html"
	"net/http"
	"slices"
	"strconv"
	"strings"
	"sync"
	"time"

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
// &delay=<ms> (up to 1500) holds the answer, like a server far away.
func (s *Server) demoArrange(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*") // public, and the playground's sandbox reads refusals too
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
	if ms, _ := strconv.Atoi(r.URL.Query().Get("delay")); ms > 0 {
		select {
		case <-time.After(time.Duration(min(ms, 1500)) * time.Millisecond):
		case <-r.Context().Done():
			return
		}
	}
	state, err := a.arrange(p.State, p.Move)
	if err != nil {
		http.Error(w, err.Error(), http.StatusUnprocessableEntity)
		return
	}
	datastar.NewSSE(w, r).PatchElements(answer(a.render(p.ID, state)))
}

// answer readies a demo's markup for its own patch. Its host carries
// data-ignore-morph, so the page's frames leave a rearranged demo alone, but
// Datastar also skips a patch whose old and new element both carry it: the
// answer swaps it for data-preserve-attr, which keeps it on the host.
func answer(markup string) string {
	name, attrs, end := startTag(markup)
	i := slices.IndexFunc(attrs, func(a attr) bool { return a.name == "data-ignore-morph" })
	if i < 0 {
		return markup
	}
	preserve := ` data-preserve-attr="data-ignore-morph"`
	var b strings.Builder
	b.WriteString("<" + name)
	for _, a := range slices.Delete(attrs, i, i+1) {
		if v, ok := strings.CutPrefix(a.raw, `data-preserve-attr="`); ok {
			a.raw, preserve = `data-preserve-attr="data-ignore-morph `+v, ""
		}
		b.WriteString(" " + a.raw)
	}
	return b.String() + preserve + ">" + markup[end:]
}

type attr struct{ name, raw string }

// startTag reads the start tag markup opens with: its name, its attributes
// and where it ends (after its >). Values may hold > inside quotes.
func startTag(markup string) (name string, attrs []attr, end int) {
	i := strings.IndexAny(markup, " \t\n\r>")
	if !strings.HasPrefix(markup, "<") || i < 0 {
		return "", nil, 0
	}
	name = markup[1:i]
	for i < len(markup) {
		for i < len(markup) && strings.ContainsRune(" \t\n\r", rune(markup[i])) {
			i++
		}
		if i >= len(markup) || markup[i] == '>' || markup[i] == '/' {
			break
		}
		start := i
		for i < len(markup) && !strings.ContainsRune(" \t\n\r=>", rune(markup[i])) {
			i++
		}
		name := markup[start:i]
		if i < len(markup) && markup[i] == '=' {
			i++
			if i < len(markup) && (markup[i] == '"' || markup[i] == '\'') {
				q := markup[i]
				if j := strings.IndexByte(markup[i+1:], q); j >= 0 {
					i += j + 2
				} else {
					return "", nil, 0
				}
			} else {
				for i < len(markup) && !strings.ContainsRune(" \t\n\r>", rune(markup[i])) {
					i++
				}
			}
		}
		attrs = append(attrs, attr{name, markup[start:i]})
	}
	if i >= len(markup) || markup[i] != '>' {
		return "", nil, 0
	}
	return name, attrs, i + 1
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

// arrangeOnAfter is arrangeOn with the answer held for ms milliseconds.
func arrangeOnAfter(kind string, ms int) string {
	return strings.Replace(arrangeOn(kind), kind+"'", kind+"?delay="+strconv.Itoa(ms)+"'", 1)
}
