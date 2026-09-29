package commands

import (
	"context"
	"database/sql"

	"starbase/internal/demo"
)

// SeedStars stores the first demo.StarCount stars of the catalog, for demos
// that sort them, replacing them when they changed since the last start. It is
// not part of SeedDemo: 100,000 rows take a second, which tests that don't need
// them shouldn't pay.
type SeedStars struct{}

func (SeedStars) Scope() string { return "" } // reference data: no view shows it live

func (SeedStars) Apply(ctx context.Context, tx *sql.Tx) error {
	version := demo.StarsVersion()
	var stored string
	err := tx.QueryRowContext(ctx, `SELECT stars FROM demo_meta WHERE id = 1`).Scan(&stored)
	if err == nil && stored == version {
		return nil
	}
	if err != nil && err != sql.ErrNoRows {
		return err
	}
	if _, err := tx.ExecContext(ctx, `DELETE FROM demo_stars`); err != nil {
		return err
	}
	ins, err := tx.PrepareContext(ctx, `INSERT INTO demo_stars (id, name, class, temp, constellation, distance, magnitude, planets) VALUES (?, ?, ?, ?, ?, ?, ?, ?)`)
	if err != nil {
		return err
	}
	defer ins.Close()
	for i := range demo.StarCount {
		s := demo.StarAt(i)
		if _, err := ins.ExecContext(ctx, s.ID, s.Name, s.Class, s.Temp, s.Constellation, s.Distance, s.Magnitude, s.Planets); err != nil {
			return err
		}
	}
	_, err = tx.ExecContext(ctx, `INSERT INTO demo_meta (id, version, stars) VALUES (1, '', ?) ON CONFLICT (id) DO UPDATE SET stars = excluded.stars`, version)
	return err
}
