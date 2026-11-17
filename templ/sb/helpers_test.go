package sb

import (
	"context"
	"encoding/json"
	"html"
	"io"
	"os"
	"strings"
	"testing"

	"github.com/a-h/templ"
)

// usage is a component's install markup, as HTML and through the wrappers
// (usage_gen_test.go).
type usage struct {
	slug, html string
	render     templ.Component
}

// withChildren renders c with children as its templ children, as
// @c { children } does.
func withChildren(c templ.Component, children ...templ.Component) templ.Component {
	return templ.ComponentFunc(func(ctx context.Context, w io.Writer) error {
		return c.Render(templ.WithChildren(ctx, templ.Join(children...)), w)
	})
}

func render(t *testing.T, c templ.Component) string {
	t.Helper()
	var b strings.Builder
	if err := c.Render(context.Background(), &b); err != nil {
		t.Fatal(err)
	}
	return b.String()
}

// manifest is a component's manifest.json (testdata/manifests.json).
type manifest struct {
	Slug  string `json:"slug"`
	Tag   string `json:"tag"`
	Props []struct {
		Name      string          `json:"name"`
		Attribute string          `json:"attribute"`
		Type      string          `json:"type"`
		Default   json.RawMessage `json:"default"`
		Values    []any           `json:"values"`
	} `json:"props"`
}

func manifests(t *testing.T) []manifest {
	t.Helper()
	b, err := os.ReadFile("testdata/manifests.json")
	if err != nil {
		t.Fatal(err)
	}
	var all []manifest
	if err := json.Unmarshal(b, &all); err != nil {
		t.Fatal(err)
	}
	return all
}

// token is a piece of parsed HTML: a start tag (with its attributes as
// written), an end tag, or text.
type token struct {
	kind  string // "start", "end", "text"
	tag   string
	attrs []tokAttr
	text  string
}

type tokAttr struct {
	name, value string
	bare        bool
}

func (t token) attr(name string) (tokAttr, bool) {
	for _, a := range t.attrs {
		if a.name == name {
			return a, true
		}
	}
	return tokAttr{}, false
}

var rawText = map[string]bool{"script": true, "style": true, "textarea": true, "title": true}

// tokens parses HTML into tokens: enough for well-formed markup, with
// raw-text elements (a script's content is one text token).
func tokens(t *testing.T, s string) []token {
	t.Helper()
	var out []token
	for i := 0; i < len(s); {
		switch {
		case strings.HasPrefix(s[i:], "<!--"):
			end := strings.Index(s[i:], "-->")
			if end < 0 {
				t.Fatalf("unclosed comment in %s", s)
			}
			i += end + 3
		case strings.HasPrefix(s[i:], "</"):
			end := strings.IndexByte(s[i:], '>')
			out = append(out, token{kind: "end", tag: strings.ToLower(strings.TrimSpace(s[i+2 : i+end]))})
			i += end + 1
		case s[i] == '<' && i+1 < len(s) && (s[i+1] >= 'a' && s[i+1] <= 'z' || s[i+1] >= 'A' && s[i+1] <= 'Z'):
			tok, n := startTag(t, s[i:])
			out = append(out, tok)
			i += n
			if rawText[tok.tag] {
				end := strings.Index(strings.ToLower(s[i:]), "</"+tok.tag)
				if end < 0 {
					t.Fatalf("unclosed <%s>", tok.tag)
				}
				out = append(out, token{kind: "text", text: s[i : i+end]})
				i += end
			}
		default:
			end := strings.IndexByte(s[i+1:], '<')
			if end < 0 {
				end = len(s) - i
			} else {
				end++
			}
			out = append(out, token{kind: "text", text: html.UnescapeString(s[i : i+end])})
			i += end
		}
	}
	return out
}

func startTag(t *testing.T, s string) (token, int) {
	t.Helper()
	i := 1
	for i < len(s) && !strings.ContainsRune(" \t\n\r\f/>", rune(s[i])) {
		i++
	}
	tok := token{kind: "start", tag: strings.ToLower(s[1:i])}
	for {
		for i < len(s) && strings.ContainsRune(" \t\n\r\f", rune(s[i])) {
			i++
		}
		if i >= len(s) {
			t.Fatalf("unclosed tag in %q", s)
		}
		if s[i] == '>' {
			return tok, i + 1
		}
		if strings.HasPrefix(s[i:], "/>") {
			return tok, i + 2
		}
		start := i
		for i < len(s) && !strings.ContainsRune(" \t\n\r\f=>", rune(s[i])) && !strings.HasPrefix(s[i:], "/>") {
			i++
		}
		a := tokAttr{name: strings.ToLower(s[start:i])}
		if i < len(s) && s[i] == '=' {
			i++
			if q := s[i]; q == '"' || q == '\'' {
				end := strings.IndexByte(s[i+1:], q)
				a.value = html.UnescapeString(s[i+1 : i+1+end])
				i += end + 2
			} else {
				start := i
				for i < len(s) && !strings.ContainsRune(" \t\n\r\f>", rune(s[i])) {
					i++
				}
				a.value = html.UnescapeString(s[start:i])
			}
		} else {
			a.bare = true
		}
		tok.attrs = append(tok.attrs, a)
	}
}

// host returns the start tag of the first element tag in s.
func host(t *testing.T, s, tag string) token {
	t.Helper()
	for _, tok := range tokens(t, s) {
		if tok.kind == "start" && tok.tag == tag {
			return tok
		}
	}
	t.Fatalf("no <%s> in %s", tag, s)
	return token{}
}
