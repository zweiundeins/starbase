package web_test

import (
	"context"
	"encoding/json"
	"regexp"
	"strings"
	"testing"

	"github.com/a-h/templ"
	"github.com/zweiundeins/starbase/templ/sb"

	"starbase/internal/commands"
	"starbase/internal/templgen/registry"
	"starbase/static"
)

var (
	importMapRe = regexp.MustCompile(`<script type="importmap" nonce="n0nce">(.*?)</script>`)
	moduleRe    = regexp.MustCompile(`<script nonce="n0nce" type="module" src="([^"]+)" integrity="([^"]+)"></script>`)
)

// TestTemplImportsPinServedFiles renders sb.Imports (the templ module's
// pins, generated from this catalog) at this server, with every component
// and with none (the snapshot's autoloader): every URL it pins is served,
// with the integrity it names. A pin generated from other files than the
// server's would fail here.
func TestTemplImportsPinServedFiles(t *testing.T) {
	ts, c, wr, cat := newServerWriter(t)
	// As in production (app.New), the snapshot also pins the Datastar build.
	pinned := *cat
	pinned.PinBuild(static.Datastar())
	if err := wr.Do(context.Background(), commands.SyncCatalog{Catalog: &pinned, Datastar: static.Datastar()}); err != nil {
		t.Fatal(err)
	}
	var all []sb.Tag
	for _, e := range registry.Entries {
		all = append(all, sb.Tag(e.Tag))
	}
	for _, tc := range []struct {
		name    string
		tags    []sb.Tag
		scripts int
	}{
		{"every component", all, 1 + len(all)}, // Datastar's own script first
		{"the autoloader", nil, 2},
	} {
		var b strings.Builder
		ctx := templ.WithNonce(context.Background(), "n0nce")
		if err := sb.Imports(sb.ImportsProps{Base: ts.URL + "/", Components: tc.tags}).Render(ctx, &b); err != nil {
			t.Fatal(err)
		}
		out := b.String()
		im := importMapRe.FindStringSubmatch(out)
		if im == nil {
			t.Fatalf("%s: no import map in %s", tc.name, out)
		}
		var m struct {
			Imports   map[string]string
			Integrity map[string]string
		}
		if err := json.Unmarshal([]byte(im[1]), &m); err != nil {
			t.Fatal(err)
		}
		pins := map[string]string{}
		for u, sri := range m.Integrity {
			pins[u] = sri
		}
		if pins[m.Imports["datastar"]] == "" {
			t.Errorf("%s: Datastar %s is not pinned", tc.name, m.Imports["datastar"])
		}
		scripts := moduleRe.FindAllStringSubmatch(out, -1)
		if len(scripts) != tc.scripts {
			t.Errorf("%s: %d module scripts, want %d", tc.name, len(scripts), tc.scripts)
		}
		for _, s := range scripts {
			pins[s[1]] = s[2]
		}
		for u, sri := range pins {
			if !strings.HasPrefix(u, ts.URL+"/c/") {
				t.Errorf("%s: %s is not on the server", tc.name, u)
				continue
			}
			if got := servedSRI(t, c, u); got != sri {
				t.Errorf("%s: %s is pinned to %s, served %s", tc.name, u, sri, got)
			}
		}
		t.Logf("%s: %d files pinned and served", tc.name, len(pins))
	}
}
