package web

import (
	"encoding/json"
	"errors"
	"io/fs"
	"net/http"
	"net/http/httptest"
	"net/url"
	"regexp"
	"slices"
	"strings"
	"testing"
	"time"

	"starbase/components"
)

// indent is what the front matter's YAML adds to a gallery card's lines.
var indent = regexp.MustCompile(`(?m)^[ \t]+`)

var arrangeHostRe = regexp.MustCompile(`id="([^"]+)"[^>]*data-state="([^"]*)"\s+data-on:[a-z-]+="@get\('/demo/arrange/([a-z-]+)[?']`)

// Every arrange demo in a README is markup the server renders for its state,
// so the first move doesn't change anything but the order.
func TestArrangeDemosMatchTheServer(t *testing.T) {
	readmes, _ := fs.Glob(components.FS, "*/README.md")
	found := 0
	for _, p := range readmes {
		b, _ := fs.ReadFile(components.FS, p)
		b = indent.ReplaceAll(b, nil)
		for _, m := range arrangeHostRe.FindAllStringSubmatch(string(b), -1) {
			a, ok := arrangers[m[3]]
			if !ok {
				t.Errorf("%s: no arranger %q", p, m[3])
				continue
			}
			found++
			if _, attrs, _ := startTag(a.render(m[1], m[2])); !slices.ContainsFunc(attrs, func(a attr) bool { return a.name == "data-ignore-morph" }) {
				t.Errorf("%s: the %s demo's host needs data-ignore-morph, or the page's next frame undoes its moves", p, m[1])
			}
			if want := indent.ReplaceAllString(a.render(m[1], m[2]), ""); !strings.Contains(string(b), want) {
				t.Errorf("%s: the %s demo isn't what the server renders; want:\n%s", p, m[1], want)
			}
		}
	}
	if found == 0 {
		t.Skip("no arrange demos yet")
	}
}

func TestMoveBefore(t *testing.T) {
	for _, tc := range []struct {
		list         string
		item, before string
		want         string
	}{
		{"a b c d", "a", "", "b c d a"},
		{"a b c d", "d", "a", "d a b c"},
		{"a b c d", "b", "d", "a c b d"},
		{"a b c d", "c", "b", "a c b d"},
	} {
		got, err := moveBefore(strings.Fields(tc.list), tc.item, tc.before)
		if err != nil || strings.Join(got, " ") != tc.want {
			t.Errorf("%s: %s before %q = %v %v, want %s", tc.list, tc.item, tc.before, got, err, tc.want)
		}
	}
	for _, bad := range [][2]string{{"x", ""}, {"a", "a"}, {"a", "x"}} {
		if _, err := moveBefore([]string{"a", "b"}, bad[0], bad[1]); err == nil {
			t.Errorf("%q before %q: no error", bad[0], bad[1])
		}
	}
}

// Refusals carry the CORS header too: without it the playground's sandbox
// sees a network error, and Datastar retries it for minutes.
func TestArrangeRefusalsAreReadable(t *testing.T) {
	arrangers["refuse"] = arranger{arrange: func(string, json.RawMessage) (string, error) { return "", errors.New("no") }}
	defer delete(arrangers, "refuse")
	for _, tc := range []struct {
		kind, query string
		code        int
	}{
		{"nope", `{}`, http.StatusNotFound},
		{"refuse", `{"id":"1 2"}`, http.StatusBadRequest},
		{"refuse", `{"id":"demo"}`, http.StatusUnprocessableEntity},
	} {
		r := httptest.NewRequest("GET", "/demo/arrange/"+tc.kind+"?datastar="+url.QueryEscape(tc.query), nil)
		r.SetPathValue("kind", tc.kind)
		w := httptest.NewRecorder()
		(&Server{}).demoArrange(w, r)
		if w.Code != tc.code || w.Header().Get("Access-Control-Allow-Origin") != "*" {
			t.Errorf("%s %s: %d, ACAO %q", tc.kind, tc.query, w.Code, w.Header().Get("Access-Control-Allow-Origin"))
		}
	}
}

func TestAnswerKeepsTheHostAnIsland(t *testing.T) {
	for _, tc := range []struct{ in, want string }{
		{`<sb-x id="a" data-ignore-morph data-state="b c"><p>x</p></sb-x>`, `<sb-x id="a" data-state="b c" data-preserve-attr="data-ignore-morph"><p>x</p></sb-x>`},
		{"<sb-x\n\tdata-ignore-morph id=a data-on:x=\"$y > 1 && @get('/z')\">", `<sb-x id=a data-on:x="$y > 1 && @get('/z')" data-preserve-attr="data-ignore-morph">`},
		{`<sb-x data-preserve-attr="style" data-ignore-morph="">`, `<sb-x data-preserve-attr="data-ignore-morph style">`},
		{`<sb-x id="a"><i data-ignore-morph></i></sb-x>`, `<sb-x id="a"><i data-ignore-morph></i></sb-x>`},
		{`<sb-x id="a`, `<sb-x id="a`},
	} {
		if got := answer(tc.in); got != tc.want {
			t.Errorf("answer(%q)\n got %q\nwant %q", tc.in, got, tc.want)
		}
	}
}

// &delay= holds the answer, up to 1.5 s, like a server far away.
func TestArrangeDelay(t *testing.T) {
	arrangers["echo"] = arranger{arrange: func(s string, _ json.RawMessage) (string, error) { return s, nil }, render: func(id, s string) string { return `<div id="` + id + `"></div>` }}
	defer delete(arrangers, "echo")
	for _, tc := range []struct {
		delay       string
		least, most time.Duration
	}{{"", 0, 100 * time.Millisecond}, {"120", 120 * time.Millisecond, 400 * time.Millisecond}, {"99999", 1500 * time.Millisecond, 2 * time.Second}} {
		r := httptest.NewRequest("GET", "/demo/arrange/echo?delay="+tc.delay+"&datastar="+url.QueryEscape(`{"id":"x","state":"s"}`), nil)
		r.SetPathValue("kind", "echo")
		w := httptest.NewRecorder()
		start := time.Now()
		(&Server{}).demoArrange(w, r)
		if took := time.Since(start); w.Code != http.StatusOK || took < tc.least || took > tc.most {
			t.Errorf("delay %q: %d after %v, want %v to %v", tc.delay, w.Code, took, tc.least, tc.most)
		}
	}
}
