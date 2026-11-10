package commands

import (
	"database/sql"

	"github.com/zweiundeins/lith/writer"

	"starbase/internal/demo"
)

// SeedStars stores the first demo.StarCount stars of the catalog, for demos
// that sort them, replacing them when they changed since the last start. It is
// not part of SeedDemo: 100,000 rows take a second, which tests that don't need
// them shouldn't pay. Like SeedDemo, it touches no topic.
type SeedStars struct{}

func (SeedStars) Apply(tx *writer.Tx) error {
	version := demo.StarsVersion()
	var stored string
	err := tx.QueryRow(`SELECT stars FROM demo_meta WHERE id = 1`).Scan(&stored)
	if err == nil && stored == version {
		return nil
	}
	if err != nil && err != sql.ErrNoRows {
		return err
	}
	if _, err := tx.Exec(`DELETE FROM demo_stars`); err != nil {
		return err
	}
	for i := range demo.StarCount {
		s := demo.StarAt(i)
		if _, err := tx.Exec(`INSERT INTO demo_stars (id, name, class, temp, constellation, distance, magnitude, planets) VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
			s.ID, s.Name, s.Class, s.Temp, s.Constellation, s.Distance, s.Magnitude, s.Planets); err != nil {
			return err
		}
	}
	_, err = tx.Exec(`INSERT INTO demo_meta (id, version, stars) VALUES (1, '', ?) ON CONFLICT (id) DO UPDATE SET stars = excluded.stars`, version)
	return err
}
