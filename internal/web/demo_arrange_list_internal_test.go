package web

import (
	"io/fs"
	"net/http"
	"net/http/httptest"
	"net/url"
	"regexp"
	"slices"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"

	"starbase/components"
	"starbase/internal/demo"
)

func TestArrangeSortableList(t *testing.T) {
	const planets = "earth mercury mars venus"
	for _, tc := range []struct{ move, want string }{
		{`{"itemId":"earth","before":"mars"}`, "mercury earth mars venus"},
		{`{"itemId":"venus","before":"earth"}`, "venus earth mercury mars"},
		{`{"itemId":"mercury","before":""}`, "earth mars venus mercury"},
		{`{"itemId":"venus","before":""}`, planets}, // dropped where it was
	} {
		if got, err := arrangeSortableList(planets, []byte(tc.move)); err != nil || got != tc.want {
			t.Errorf("%s: %q, %v; want %q", tc.move, got, err, tc.want)
		}
	}

	var many []string
	for _, b := range demo.Universe()[:41] {
		many = append(many, b.ID)
	}
	for _, tc := range []struct{ name, state, move string }{
		{"a body the dataset doesn't have", "earth nibiru", `{"itemId":"earth","before":""}`},
		{"a body twice", "earth mars earth", `{"itemId":"mars","before":""}`},
		{"more than 40 bodies", strings.Join(many, " "), `{"itemId":"` + many[0] + `","before":""}`},
		{"an item that isn't in the list", planets, `{"itemId":"jupiter","before":""}`},
		{"a before that isn't in the list", planets, `{"itemId":"earth","before":"jupiter"}`},
		{"an item before itself", planets, `{"itemId":"earth","before":"earth"}`},
		{"no move", planets, `null`},
		{"a move that isn't JSON", planets, `earth`},
		{"a move of the wrong shape", planets, `{"itemId":3}`},
	} {
		if got, err := arrangeSortableList(tc.state, []byte(tc.move)); err == nil {
			t.Errorf("%s: %q, want an error", tc.name, got)
		}
	}
}

func TestArrangeSortableListOverHTTP(t *testing.T) {
	get := func(query string) *httptest.ResponseRecorder {
		r := httptest.NewRequest("GET", "/demo/arrange/sortable-list?datastar="+url.QueryEscape(query), nil)
		r.SetPathValue("kind", "sortable-list")
		w := httptest.NewRecorder()
		(&Server{}).demoArrange(w, r)
		return w
	}
	w := get(`{"id":"inner-planets","state":"earth mercury mars venus","move":{"itemId":"earth","before":""}}`)
	if w.Code != http.StatusOK || !strings.Contains(w.Body.String(), `data-state="mercury mars venus earth"`) {
		t.Errorf("a move: %d\n%s", w.Code, w.Body)
	}
	for _, q := range []string{
		`{"id":"inner-planets","state":"earth mercury","move":{"itemId":"venus","before":""}}`,
		`{"id":"inner-planets","state":"earth <b>","move":{"itemId":"earth","before":""}}`,
	} {
		if w := get(q); w.Code != http.StatusUnprocessableEntity {
			t.Errorf("%s: %d, want 422", q, w.Code)
		}
	}
	if w := get(`{"id":"x\"><b>","state":"earth mercury","move":{"itemId":"earth","before":""}}`); w.Code != http.StatusBadRequest {
		t.Errorf("a bad id: %d, want 400", w.Code)
	}
}

func TestRenderSortableListEscapes(t *testing.T) {
	if got := renderSortableList(`x"><b>`, "earth"); strings.Contains(got, "<b>") {
		t.Errorf("unescaped:\n%s", got)
	}
}

var (
	sortableHostRe = regexp.MustCompile(`<sb-sortable-list\s[^>]*>`)
	attrRe         = regexp.MustCompile(`[a-z:-]+="[^"]*"`)
)

// The gallery card is the server's render with its attributes in another
// order, which TestArrangeDemosMatchTheServer can't see in YAML-indented text.
func TestSortableListCardMatchesTheServer(t *testing.T) {
	b, err := fs.ReadFile(components.FS, "sortable-list/README.md")
	if err != nil {
		t.Fatal(err)
	}
	var meta struct{ Preview string }
	if err := yaml.Unmarshal([]byte(strings.SplitN(string(b), "---\n", 3)[1]), &meta); err != nil {
		t.Fatal(err)
	}
	norm := func(s string) string {
		s = sortableHostRe.ReplaceAllStringFunc(s, func(tag string) string {
			attrs := attrRe.FindAllString(tag, -1)
			slices.Sort(attrs)
			return "<sb-sortable-list " + strings.Join(attrs, " ") + ">"
		})
		return strings.Join(strings.Fields(s), " ")
	}
	host := sortableHostRe.FindString(meta.Preview)
	id := regexp.MustCompile(`id="([^"]+)"`).FindStringSubmatch(host)
	state := regexp.MustCompile(`data-state="([^"]*)"`).FindStringSubmatch(host)
	if id == nil || state == nil {
		t.Fatalf("no id and data-state on the card's list: %s", host)
	}
	if want := renderSortableList(id[1], state[1]); !strings.Contains(norm(meta.Preview), norm(want)) {
		t.Errorf("the card isn't what the server renders; want:\n%s", want)
	}
	if !strings.Contains(meta.Preview, `id="`+id[1]+`-hint"`) {
		t.Errorf("no %s-hint on the card for its items' aria-describedby", id[1])
	}
}
