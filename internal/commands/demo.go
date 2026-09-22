package commands

import (
	"context"
	"database/sql"

	"starbase/internal/demo"
)

// SeedDemo stores the example dataset (internal/demo), replacing it when it
// changed since the last start.
type SeedDemo struct{}

func (SeedDemo) Scope() string { return "" } // reference data: no view shows it live

func (SeedDemo) Apply(ctx context.Context, tx *sql.Tx) error {
	version := demo.Version()
	var stored string
	err := tx.QueryRowContext(ctx, `SELECT version FROM demo_meta WHERE id = 1`).Scan(&stored)
	if err == nil && stored == version {
		return nil
	}
	if err != nil && err != sql.ErrNoRows {
		return err
	}
	if _, err := tx.ExecContext(ctx, `DELETE FROM demo_bodies`); err != nil {
		return err
	}
	for _, b := range demo.Universe() {
		if _, err := tx.ExecContext(ctx, `INSERT INTO demo_bodies (id, parent, kind, name, detail, ord, folded) VALUES (?, ?, ?, ?, ?, ?, ?)`,
			b.ID, b.Parent, b.Kind, b.Name, b.Detail, b.Order, demo.Fold(b.Name+" "+b.Detail)); err != nil {
			return err
		}
	}
	_, err = tx.ExecContext(ctx, `INSERT INTO demo_meta (id, version) VALUES (1, ?) ON CONFLICT (id) DO UPDATE SET version = excluded.version`, version)
	return err
}
