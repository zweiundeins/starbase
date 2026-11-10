package commands

import (
	"database/sql"

	"github.com/zweiundeins/lith/writer"

	"starbase/internal/demo"
)

// SeedDemo stores the example dataset (internal/demo), replacing it when it
// changed since the last start. It touches no topic: reference data, which
// no page shows live.
type SeedDemo struct{}

func (SeedDemo) Apply(tx *writer.Tx) error {
	version := demo.Version()
	var stored string
	err := tx.QueryRow(`SELECT version FROM demo_meta WHERE id = 1`).Scan(&stored)
	if err == nil && stored == version {
		return nil
	}
	if err != nil && err != sql.ErrNoRows {
		return err
	}
	if _, err := tx.Exec(`DELETE FROM demo_bodies`); err != nil {
		return err
	}
	for _, b := range demo.Universe() {
		if _, err := tx.Exec(`INSERT INTO demo_bodies (id, parent, kind, name, detail, ord, folded) VALUES (?, ?, ?, ?, ?, ?, ?)`,
			b.ID, b.Parent, b.Kind, b.Name, b.Detail, b.Order, demo.Fold(b.Name+" "+b.Detail)); err != nil {
			return err
		}
	}
	_, err = tx.Exec(`INSERT INTO demo_meta (id, version) VALUES (1, ?) ON CONFLICT (id) DO UPDATE SET version = excluded.version`, version)
	return err
}
