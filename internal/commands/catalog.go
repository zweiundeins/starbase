// Package commands contains every state change the application can make.
// Each command is a plain struct; HTTP handlers build one, the cqrs.Bus
// applies it inside the single writer transaction.
package commands

import (
	"context"
	"database/sql"
	"encoding/json"
	"strings"
	"time"

	"starbase/internal/catalog"
)

// SyncCatalog mirrors the embedded component folders into SQLite. Components
// whose folders were removed are deactivated, keeping their stars.
type SyncCatalog struct {
	Catalog  *catalog.Catalog
	Datastar []byte // the patched Datastar + Rocket build, kept for good like a component version
}

func (c SyncCatalog) Apply(ctx context.Context, tx *sql.Tx) error {
	now := time.Now().Unix()
	if _, err := tx.ExecContext(ctx, `UPDATE components SET active = 0`); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, `DELETE FROM components_fts`); err != nil {
		return err
	}
	for _, comp := range c.Catalog.Components {
		tags, _ := json.Marshal(comp.Tags)
		_, err := tx.ExecContext(ctx, `
			INSERT INTO components (slug, tag, name, category, summary, author, tags, since, content_hash, active, listed, updated_at)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, 1, ?, ?)
			ON CONFLICT (slug) DO UPDATE SET
				tag = excluded.tag, name = excluded.name, category = excluded.category,
				summary = excluded.summary, author = excluded.author, tags = excluded.tags,
				since = excluded.since, content_hash = excluded.content_hash, active = 1, listed = excluded.listed,
				updated_at = CASE WHEN content_hash = excluded.content_hash THEN updated_at ELSE excluded.updated_at END`,
			comp.Slug, comp.Tag, comp.Name, comp.Category, comp.Summary, comp.Author,
			string(tags), comp.Since, comp.Hash, boolInt(!comp.Unlisted), now)
		if err != nil {
			return err
		}
		if !comp.Unlisted { // out of the gallery means out of its search too
			_, err = tx.ExecContext(ctx, `INSERT INTO components_fts (slug, name, summary, tags) VALUES (?, ?, ?, ?)`,
				comp.Slug, comp.Name, comp.Summary, strings.Join(comp.Tags, " ")+" "+comp.Category+" "+comp.Tag)
			if err != nil {
				return err
			}
		}
		// Keep this version's public files for good (pinned URLs). The
		// minified ones too: stored once, their bytes never change even if a
		// later esbuild would minify differently.
		files, err := c.Catalog.ModuleFiles(comp)
		if err != nil {
			return err
		}
		mins, err := c.Catalog.MinFiles(comp)
		if err != nil {
			return err
		}
		for p, body := range mins {
			files[p] = body
		}
		for p, body := range files {
			if err := storeFile(ctx, tx, comp.Slug, comp.Hash, p, body, now); err != nil {
				return err
			}
		}
	}
	if len(c.Datastar) > 0 {
		if err := storeFile(ctx, tx, catalog.DatastarSlug, catalog.VersionHash(c.Datastar), catalog.DatastarFile, c.Datastar, now); err != nil {
			return err
		}
	}
	// And the snapshot of the whole catalog, with its frozen autoloader.
	auto := catalog.AutoloaderJS(c.Catalog, "../")
	res, err := tx.ExecContext(ctx, `INSERT OR IGNORE INTO catalog_snapshots (hash, autoloader, integrity, created_at) VALUES (?, ?, ?, ?)`,
		c.Catalog.Hash, auto, catalog.SRI([]byte(auto)), now)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n > 0 {
		for _, comp := range c.Catalog.Components {
			if _, err := tx.ExecContext(ctx, `INSERT INTO catalog_snapshot_components (snapshot, slug, hash) VALUES (?, ?, ?)`,
				c.Catalog.Hash, comp.Slug, comp.Hash); err != nil {
				return err
			}
		}
		// A new snapshot also pins the Datastar build it was published with.
		if len(c.Datastar) > 0 {
			if _, err := tx.ExecContext(ctx, `INSERT INTO catalog_snapshot_components (snapshot, slug, hash) VALUES (?, ?, ?)`,
				c.Catalog.Hash, catalog.DatastarSlug, catalog.VersionHash(c.Datastar)); err != nil {
				return err
			}
		}
	}
	return nil
}

// storeFile keeps one file of a version for good. The first bytes stored for
// it stay (INSERT OR IGNORE), and its body is stored once for every version
// that has the same bytes (file_bodies, migration 010).
func storeFile(ctx context.Context, tx *sql.Tx, slug, hash, path string, body []byte, now int64) error {
	sri := catalog.SRI(body)
	res, err := tx.ExecContext(ctx, `INSERT OR IGNORE INTO component_files (slug, hash, path, integrity, created_at) VALUES (?, ?, ?, ?, ?)`,
		slug, hash, path, sri, now)
	if err != nil {
		return err
	}
	if n, err := res.RowsAffected(); err != nil || n == 0 {
		return err
	}
	_, err = tx.ExecContext(ctx, `INSERT OR IGNORE INTO file_bodies (integrity, body) VALUES (?, ?)`, sri, body)
	return err
}

func boolInt(b bool) int {
	if b {
		return 1
	}
	return 0
}
