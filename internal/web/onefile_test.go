package web_test

import (
	"context"
	"database/sql"
	"encoding/json"
	"strings"
	"testing"

	"starbase/internal/catalog"
)

// restamp replaces a stored file's bytes, as if that version had been first
// published by a build that made them differently.
type restamp struct{ slug, hash, path, body string }

func (c restamp) Apply(ctx context.Context, tx *sql.Tx) error {
	_, err := tx.ExecContext(ctx, `UPDATE component_files SET body = ?, integrity = ? WHERE slug = ? AND hash = ? AND path = ?`,
		[]byte(c.body), catalog.SRI([]byte(c.body)), c.slug, c.hash, c.path)
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
}
