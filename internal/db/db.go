// Package db opens the SQLite database through lith/sqlite with starbase's
// migrations: one write connection, which the writer owns, and a pool of
// query_only connections for everything else.
package db

import (
	"context"
	"database/sql"
	"embed"
	"fmt"
	"io/fs"
	"math"
	"runtime"
	"slices"
	"strconv"
	"strings"
	"testing"

	"github.com/zweiundeins/lith/sqlite"
)

//go:embed migrations/*.sql
var migrations embed.FS

// Open opens (and migrates) the database at path. Use a file in a temp
// folder in tests; in-memory databases do not share across pools.
func Open(ctx context.Context, path string) (*sqlite.DB, error) {
	return sqlite.Open(ctx, path, sqlite.Options{
		Synchronous: synchronous(),
		Migrate:     func(ctx context.Context, w *sql.DB) error { return migrateTo(ctx, w, math.MaxInt) },
		// Each render worker (one per CPU) holds a connection for good; the
		// other half serves documents, demos and downloads.
		Readers:  2 * max(4, runtime.GOMAXPROCS(0)),
		CacheKiB: 16000,
		// No Litestream: the writer checkpoints between batches.
		Checkpoint: sqlite.CheckpointWriter,
	})
}

// synchronous is NORMAL, or OFF (no fsync) for the throwaway databases of tests.
func synchronous() string {
	if testing.Testing() {
		return "OFF"
	}
	return "NORMAL"
}

// migrateTo applies migrations up to number last with lith's runner. A
// migration with the line "-- then: VACUUM" has the database vacuumed once
// it is committed.
func migrateTo(ctx context.Context, w *sql.DB, last int) error {
	var version int
	if err := w.QueryRowContext(ctx, "PRAGMA user_version").Scan(&version); err != nil {
		return err
	}
	m, err := fs.Sub(migrations, "migrations")
	if err != nil {
		return err
	}
	pending := upTo{m, last}
	if err := sqlite.Migrate(ctx, w, pending); err != nil {
		return err
	}
	entries, err := pending.ReadDir(".")
	if err != nil {
		return err
	}
	vacuum := slices.ContainsFunc(entries, func(e fs.DirEntry) bool {
		body, err := fs.ReadFile(m, e.Name())
		return err == nil && number(e.Name()) > version && strings.Contains(string(body), "\n-- then: VACUUM\n")
	})
	// Only a database that had rows before has space to give back.
	if vacuum && version > 0 {
		if _, err := w.ExecContext(ctx, "VACUUM"); err != nil {
			return fmt.Errorf("vacuum: %w", err)
		}
		_, err := w.ExecContext(ctx, "PRAGMA wal_checkpoint(TRUNCATE)") // VACUUM wrote the whole database into the WAL
		return err
	}
	return nil
}

// upTo lists the migrations numbered up to last; with last below the
// newest, lith's runner would otherwise apply them all.
type upTo struct {
	fs.FS
	last int
}

func (u upTo) ReadDir(name string) ([]fs.DirEntry, error) {
	entries, err := fs.ReadDir(u.FS, name)
	return slices.DeleteFunc(entries, func(e fs.DirEntry) bool { return number(e.Name()) > u.last }), err
}

func number(name string) int {
	n, _ := strconv.Atoi(strings.SplitN(name, "_", 2)[0])
	return n
}
