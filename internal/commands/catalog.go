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
	Catalog *catalog.Catalog
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
			INSERT INTO components (slug, tag, name, category, summary, author, tags, since, content_hash, active, updated_at)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, 1, ?)
			ON CONFLICT (slug) DO UPDATE SET
				tag = excluded.tag, name = excluded.name, category = excluded.category,
				summary = excluded.summary, author = excluded.author, tags = excluded.tags,
				since = excluded.since, content_hash = excluded.content_hash, active = 1,
				updated_at = CASE WHEN content_hash = excluded.content_hash THEN updated_at ELSE excluded.updated_at END`,
			comp.Slug, comp.Tag, comp.Name, comp.Category, comp.Summary, comp.Author,
			string(tags), comp.Since, comp.Hash, now)
		if err != nil {
			return err
		}
		_, err = tx.ExecContext(ctx, `INSERT INTO components_fts (slug, name, summary, tags) VALUES (?, ?, ?, ?)`,
			comp.Slug, comp.Name, comp.Summary, strings.Join(comp.Tags, " ")+" "+comp.Category+" "+comp.Tag)
		if err != nil {
			return err
		}
	}
	return nil
}
