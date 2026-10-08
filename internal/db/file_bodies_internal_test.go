package db

import (
	"bytes"
	"context"
	"database/sql"
	"io"
	"log/slog"
	"path/filepath"
	"testing"
	"testing/fstest"

	"starbase/internal/catalog"
	"starbase/internal/commands"
	"starbase/internal/cqrs"
	"starbase/internal/queries"
)

type storedFile struct {
	slug, hash, path string
	body             []byte
}

// Migration 010 stores each distinct body once: every row keeps its
// integrity, every pinned file reads back the same bytes, and a sync stores
// a body only for bytes it hasn't stored yet.
func TestFileBodiesMigration(t *testing.T) {
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "s.db")
	w, err := sql.Open("sqlite", dsn(path)+"&_txlock=immediate")
	if err != nil {
		t.Fatal(err)
	}
	w.SetMaxOpenConns(1)
	if err := migrateTo(ctx, w, 9); err != nil {
		t.Fatal(err)
	}
	lib := bytes.Repeat([]byte("export const lib = 1;\n"), 50_000) // a vendored library, 1.1 MB, in both versions
	datastar := []byte("// Datastar\n")
	files := []storedFile{
		{"gone", "aaaaaaaaaaaa", "gone.js", []byte("rocket('sb-gone', {})")},
		{"gone", "aaaaaaaaaaaa", "vendor/lib.min.js", lib},
		{"gone", "bbbbbbbbbbbb", "gone.js", []byte("rocket('sb-gone', { v: 2 })")},
		{"gone", "bbbbbbbbbbbb", "vendor/lib.min.js", lib},
		{catalog.DatastarSlug, catalog.VersionHash(datastar), catalog.DatastarFile, datastar},
	}
	for i, f := range files {
		if _, err := w.ExecContext(ctx, `INSERT INTO component_files (slug, hash, path, body, integrity, created_at) VALUES (?, ?, ?, ?, ?, ?)`,
			f.slug, f.hash, f.path, f.body, catalog.SRI(f.body), 1000+i); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := w.ExecContext(ctx, `INSERT INTO catalog_snapshots (hash, autoloader, integrity, created_at) VALUES ('cccccccccccc', '', '', 1);
		INSERT INTO catalog_snapshot_components (snapshot, slug, hash) VALUES ('cccccccccccc', 'gone', 'aaaaaaaaaaaa')`); err != nil {
		t.Fatal(err)
	}
	w.Close()

	d, err := Open(ctx, path)
	if err != nil {
		t.Fatal(err)
	}
	defer d.Close()
	count := func(q string, args ...any) int {
		t.Helper()
		var n int
		if err := d.R.QueryRowContext(ctx, q, args...).Scan(&n); err != nil {
			t.Fatal(err)
		}
		return n
	}
	if n := count(`SELECT count(*) FROM file_bodies`); n != 4 {
		t.Errorf("%d bodies stored, want 4 (the library once)", n)
	}
	if n := count(`SELECT count(*) FROM component_files WHERE body IS NULL`); n != len(files) {
		t.Errorf("%d of %d rows refer to file_bodies", n, len(files))
	}
	if n := count(`PRAGMA freelist_count`); n != 0 {
		t.Errorf("%d free pages: the migration should vacuum", n)
	}
	q := queries.New(d.R)
	check := func(step string, want []storedFile) {
		t.Helper()
		err := q.View(ctx, func(r *queries.Reader) error {
			for _, f := range want {
				body, sri, ok, err := r.ComponentFile(ctx, f.slug, f.hash, f.path)
				if err != nil {
					return err
				}
				if integrity, _ := r.FileIntegrity(ctx, f.slug, f.hash, f.path); !ok || !bytes.Equal(body, f.body) || sri != catalog.SRI(f.body) || integrity != sri {
					t.Errorf("%s: %s@%s/%s: ok %v, %d bytes, integrity %s / %s", step, f.slug, f.hash, f.path, ok, len(body), sri, integrity)
				}
			}
			snap, err := r.SnapshotFiles(ctx, "cccccccccccc")
			if len(snap) != 2 || snap[1].Path != "vendor/lib.min.js" || snap[1].Integrity != catalog.SRI(lib) {
				t.Errorf("%s: snapshot files %+v", step, snap)
			}
			return err
		})
		if err != nil {
			t.Fatal(err)
		}
	}
	check("migrated", files)

	// After a rollback, a binary from before 010 still starts: its sync's
	// insert of a stored file is ignored, and its read gets no body (it
	// serves the current version from memory).
	if _, err := d.W.ExecContext(ctx, `INSERT OR IGNORE INTO component_files (slug, hash, path, body, integrity, created_at) VALUES (?, ?, ?, ?, ?, ?)`,
		"gone", "aaaaaaaaaaaa", "gone.js", files[0].body, catalog.SRI(files[0].body), 2000); err != nil {
		t.Errorf("a binary from before 010 can't sync: %v", err)
	}
	var old []byte
	if err := d.R.QueryRowContext(ctx, `SELECT body FROM component_files WHERE slug = 'gone' AND hash = 'aaaaaaaaaaaa' AND path = 'gone.js'`).Scan(&old); err != nil || old != nil {
		t.Errorf("a binary from before 010 reads %q, %v", old, err)
	}

	// A new version of gone whose library didn't change: its new files get
	// bodies, the library is shared. A second sync stores nothing.
	cat, err := catalog.Load(fstest.MapFS{
		"gone/README.md":           {Data: []byte("---\nname: Gone\ntag: sb-gone\ncategory: forms\nsummary: Back.\nauthor: someone\nsince: 2026-01-01\npreview: <sb-gone></sb-gone>\n---\nDocs.\n")},
		"gone/gone.js":             {Data: []byte("rocket('sb-gone', { v: 3 })")},
		"gone/vendor/lib.min.js":   {Data: lib},
		"gone/vendor/README.txt":   {Data: []byte("not a module")},
		"gone/vendor/other.min.js": {Data: []byte("export const other = 1")},
	})
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	bus := cqrs.NewBus(d.W, cqrs.NewHub(), slog.New(slog.NewTextHandler(io.Discard, nil)))
	go bus.Run(ctx)
	sync := func() (rows, bodies int) {
		if err := bus.Exec(ctx, commands.SyncCatalog{Catalog: cat, Datastar: datastar}); err != nil {
			t.Fatal(err)
		}
		return count(`SELECT count(*) FROM component_files`), count(`SELECT count(*) FROM file_bodies`)
	}
	rows, bodies := sync()
	gone, _ := cat.Get("gone")
	mins, _ := cat.MinFiles(gone)
	// gone.js, gone.min.js, the bundle, the two vendored files.
	if rows != len(files)+5 || bodies != 4+4 {
		t.Errorf("after the first sync: %d rows, %d bodies; want %d and 8", rows, bodies, len(files)+5)
	}
	if again, againBodies := sync(); again != rows || againBodies != bodies {
		t.Errorf("a second sync stored %d rows and %d bodies more", again-rows, againBodies-bodies)
	}
	check("synced", append(files,
		storedFile{"gone", gone.Hash, "vendor/lib.min.js", lib},
		storedFile{"gone", gone.Hash, catalog.BundleName("gone"), mins[catalog.BundleName("gone")]}))
	if n := count(`SELECT count(*) FROM file_bodies WHERE integrity = ?`, catalog.SRI(lib)); n != 1 {
		t.Errorf("the library is stored %d times", n)
	}
}
