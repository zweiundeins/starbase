// Package db opens the SQLite database as two pools: a single-connection
// writer used only by the command bus, and a read-only pool for queries.
package db

import (
	"context"
	"database/sql"
	"embed"
	"errors"
	"fmt"
	"io/fs"
	"net/url"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strconv"
	"strings"

	_ "modernc.org/sqlite"
)

//go:embed migrations/*.sql
var migrations embed.FS

type DB struct {
	W *sql.DB // writer: exactly one connection, BEGIN IMMEDIATE
	R *sql.DB // readers: query_only
}

var commonPragmas = []string{
	"journal_mode(WAL)",
	"synchronous(NORMAL)",
	"busy_timeout(5000)",
	"temp_store(MEMORY)",
	"foreign_keys(ON)",
	"cache_size(-16000)",
}

func dsn(path string, extra ...string) string {
	q := url.Values{}
	for _, p := range append(commonPragmas, extra...) {
		q.Add("_pragma", p)
	}
	return "file:" + path + "?" + q.Encode()
}

// Open opens (and migrates) the database at path. Use ":memory:"-like
// temp files in tests; in-memory databases do not share across pools.
func Open(ctx context.Context, path string) (*DB, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil, err
	}
	w, err := sql.Open("sqlite", dsn(path)+"&_txlock=immediate")
	if err != nil {
		return nil, err
	}
	w.SetMaxOpenConns(1)
	w.SetConnMaxLifetime(0)
	if err := migrate(ctx, w); err != nil {
		w.Close()
		return nil, fmt.Errorf("migrate: %w", err)
	}
	r, err := sql.Open("sqlite", dsn(path, "query_only(1)", "mmap_size(268435456)"))
	if err != nil {
		w.Close()
		return nil, err
	}
	r.SetMaxOpenConns(max(4, runtime.NumCPU()))
	r.SetMaxIdleConns(max(4, runtime.NumCPU()))
	return &DB{W: w, R: r}, nil
}

func (d *DB) Close() error {
	return errors.Join(d.R.Close(), d.W.Close())
}

// migrate applies migrations/NNN_*.sql whose number exceeds PRAGMA user_version.
func migrate(ctx context.Context, w *sql.DB) error {
	var version int
	if err := w.QueryRowContext(ctx, "PRAGMA user_version").Scan(&version); err != nil {
		return err
	}
	entries, err := fs.ReadDir(migrations, "migrations")
	if err != nil {
		return err
	}
	sort.Slice(entries, func(i, j int) bool { return entries[i].Name() < entries[j].Name() })
	for _, e := range entries {
		n, err := strconv.Atoi(strings.SplitN(e.Name(), "_", 2)[0])
		if err != nil {
			return fmt.Errorf("bad migration name %q", e.Name())
		}
		if n <= version {
			continue
		}
		body, err := migrations.ReadFile("migrations/" + e.Name())
		if err != nil {
			return err
		}
		tx, err := w.BeginTx(ctx, nil)
		if err != nil {
			return err
		}
		if _, err := tx.ExecContext(ctx, string(body)); err != nil {
			tx.Rollback()
			return fmt.Errorf("%s: %w", e.Name(), err)
		}
		if _, err := tx.ExecContext(ctx, fmt.Sprintf("PRAGMA user_version = %d", n)); err != nil {
			tx.Rollback()
			return err
		}
		if err := tx.Commit(); err != nil {
			return err
		}
	}
	return nil
}
