package web_test

import (
	"context"
	"database/sql"
	"encoding/json"
	"regexp"
	"strconv"
	"strings"
	"testing"

	"starbase/internal/catalog"
	"starbase/internal/ui"
)

// restamp replaces a stored file's bytes, as if that version had been first
// published by a build that made them differently.
type restamp struct{ slug, hash, path, body string }

func (c restamp) Apply(ctx context.Context, tx *sql.Tx) error {
	sri := catalog.SRI([]byte(c.body))
	if _, err := tx.ExecContext(ctx, `INSERT INTO file_bodies (integrity, body) VALUES (?, ?)`, sri, []byte(c.body)); err != nil {
		return err
	}
	_, err := tx.ExecContext(ctx, `UPDATE component_files SET integrity = ? WHERE slug = ? AND hash = ? AND path = ?`, sri, c.slug, c.hash, c.path)
	return err
}

// A component's one-file bundle is served next to its other files,
// immutable and open to other sites, with the integrity the database holds,
// in the snapshot's import map, and frozen: the stored bytes are served even
// when this binary would build them differently.
func TestComponentBundleURL(t *testing.T) {
	ts, c, bus, cat := newServerBus(t)
	kanban, _ := cat.Get("kanban-board")
	bundle, err := cat.BundleOf(kanban)
	if err != nil {
		t.Fatal(err)
	}
	u := ts.URL + "/c/" + kanban.VersionedBundle()
	if u != ts.URL+"/c/kanban-board@"+kanban.Hash+"/kanban-board.bundle.min.js" {
		t.Fatalf("bundle URL %s", u)
	}
	res, body := get(t, c, u)
	if res.StatusCode != 200 || !strings.Contains(res.Header.Get("Cache-Control"), "immutable") || res.Header.Get("Access-Control-Allow-Origin") != "*" ||
		!strings.HasPrefix(res.Header.Get("Content-Type"), "text/javascript") || body != string(bundle.Body) {
		t.Fatalf("bundle: %d %v, %d bytes", res.StatusCode, res.Header, len(body))
	}
	if r, latest := get(t, c, ts.URL+"/c/kanban-board/kanban-board.bundle.min.js"); r.StatusCode != 200 || latest != body {
		t.Errorf("the unversioned route should follow the current version: %d", r.StatusCode)
	}
	_, mapJSON := get(t, c, ts.URL+"/c/@"+cat.Hash+"/importmap.json")
	var im struct{ Integrity map[string]string }
	json.Unmarshal([]byte(mapJSON), &im)
	if im.Integrity[u] != catalog.SRI(bundle.Body) {
		t.Errorf("importmap.json: %q for the bundle", im.Integrity[u])
	}

	frozen := "/*! sb-kanban-board, version kanban-board@" + kanban.Hash + ", first published by another build */\n"
	if err := bus.Exec(context.Background(), restamp{"kanban-board", kanban.Hash, catalog.BundleName("kanban-board"), frozen}); err != nil {
		t.Fatal(err)
	}
	if _, body := get(t, c, u); body != frozen {
		t.Errorf("the stored bundle should be served, got %.100s", body)
	}
	_, page := get(t, c, ts.URL+"/components/kanban-board")
	if !strings.Contains(installPanels(t, page)["this-component"][0], `src="`+u+`" integrity="`+catalog.SRI([]byte(frozen))+`"`) {
		t.Error("the install snippet should pin the stored bundle's integrity")
	}
}

// The download size a page shows is the bundles': the gallery card, the
// component page's meta row and the head of its Size section, with the
// module files below, behind "As module files".
func TestBundleSizesOnPages(t *testing.T) {
	ts, c, _, cat := newServerBus(t)
	kanban, _ := cat.Get("kanban-board")
	size := ui.FmtBytes(kanban.Sizes.Total.Min)
	if kanban.Sizes.Total != kanban.Sizes.Bundle.Size || size == ui.FmtBytes(kanban.Sizes.ModulesTotal.Min) {
		t.Fatalf("kanban-board: %+v as one file, %+v as modules", kanban.Sizes.Total, kanban.Sizes.ModulesTotal)
	}
	_, home := get(t, c, ts.URL+"/")
	if card := regexp.MustCompile(`(?s)id="card-kanban-board".*?class="card__size"[^>]*>([^<]+)<`).FindStringSubmatch(home); card == nil || card[1] != size {
		t.Errorf("the gallery card shows %q, want %s", card, size)
	}
	_, page := get(t, c, ts.URL+"/components/kanban-board")
	if !strings.Contains(page, `href="#size" title="`+size+` with brotli: its one-file bundle">`+size+`</a>`) {
		t.Errorf("the meta row lacks %s", size)
	}
	i := strings.Index(page, `<h2 id="size">`)
	section := page[i : i+strings.Index(page[i:], "</details>")]
	bundle, modules := strings.Index(section, "<code>kanban-board.bundle.min.js</code>"), strings.Index(section, "<summary>As module files: "+strconv.Itoa(kanban.Sizes.ModuleFiles)+" requests, "+ui.FmtBytes(kanban.Sizes.ModulesTotal.Min))
	if i < 0 || bundle < 0 || modules < bundle {
		t.Errorf("the Size section should lead with the bundle, then the module files:\n%s", section)
	}
	// What a bundle loads on first use counts, and says so.
	editor, _ := cat.Get("code-editor")
	prism := ui.FmtBytes(editor.Sizes.Lazy[0].Min)
	_, page = get(t, c, ts.URL+"/components/code-editor")
	if !strings.Contains(page, `title="`+ui.FmtBytes(editor.Sizes.Total.Min)+` with brotli, of which `+prism+` loaded on first use: its one-file bundle, with what it loads">`) ||
		!strings.Contains(page, "<code>vendor/prism.min.js</code> <span class=\"muted\">on first use</span>") ||
		!strings.Contains(page, "Of the total, "+prism+" brotli is loaded on first use") {
		t.Error("the code-editor page should count the Prism it loads on first use, and say so")
	}
	// The This component tab counts the module files as the Size section does.
	if !strings.Contains(page, "Or the minified module files (2 files,") || editor.Sizes.ModuleFiles != 2 {
		t.Error("the This component tab should count code-editor's two module files")
	}
}
