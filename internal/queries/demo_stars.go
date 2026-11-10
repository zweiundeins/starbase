package queries

import (
	"context"
	"strings"

	"github.com/zweiundeins/lith/sqlite"

	"starbase/internal/demo"
)

// starOrder maps a sort key of the star catalog to its column: the class
// sorts by temperature.
var starOrder = map[string]string{
	"name": "name", "class": "temp", "constellation": "constellation",
	"distance": "distance", "magnitude": "magnitude",
}

// DemoStarSortable reports whether the star catalog sorts by key.
func DemoStarSortable(key string) bool { return starOrder[key] != "" }

// starOrderBy is the ORDER BY for sort (a key of starOrder, else the
// catalog's own order). Ties go in id order (the indexes hold it), so every
// window of one order fits the next.
func starOrderBy(sort string, desc bool) string {
	col, dir := starOrder[sort], " ASC"
	if col == "" {
		col = "id"
	}
	if desc {
		dir = " DESC"
	}
	return col + dir + ", id" + dir
}

// DemoStars returns up to count stars from offset, sorted by sort (a key of
// starOrder, else the catalog's own order), and how many stars there are.
func (r *Reader) DemoStars(ctx context.Context, sort string, desc bool, offset, count int) ([]demo.Star, int, error) {
	var total int
	if err := r.tx.QueryRowContext(ctx, `SELECT count(*) FROM demo_stars`).Scan(&total); err != nil {
		return nil, 0, err
	}
	rows, err := r.tx.QueryContext(ctx, `SELECT id, name, class, temp, constellation, distance, magnitude, planets FROM demo_stars
		ORDER BY `+starOrderBy(sort, desc)+` LIMIT ? OFFSET ?`, count, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	out := []demo.Star{}
	for rows.Next() {
		var s demo.Star
		if err := rows.Scan(&s.ID, &s.Name, &s.Class, &s.Temp, &s.Constellation, &s.Distance, &s.Magnitude, &s.Planets); err != nil {
			return nil, 0, err
		}
		out = append(out, s)
	}
	return out, total, rows.Err()
}

// DemoStarLongest is what the star catalog's widest values are made of: the
// most characters of each text column, and the range of each number.
type DemoStarLongest struct {
	Name, Class, Constellation int
	Distance, MagnitudeMin     float64
	MagnitudeMax               float64
	Planets                    int
}

// DemoStarLongest scans the catalog for its longest values.
func (r *Reader) DemoStarLongest(ctx context.Context) (DemoStarLongest, error) {
	var l DemoStarLongest
	err := r.tx.QueryRowContext(ctx, `SELECT coalesce(max(length(name)), 0), coalesce(max(length(class)), 0), coalesce(max(length(constellation)), 0),
		coalesce(max(distance), 0), coalesce(min(magnitude), 0), coalesce(max(magnitude), 0), coalesce(max(planets), 0) FROM demo_stars`).
		Scan(&l.Name, &l.Class, &l.Constellation, &l.Distance, &l.MagnitudeMin, &l.MagnitudeMax, &l.Planets)
	return l, err
}

// EachDemoStar calls fn with every star in the order DemoStars gives them, or
// only with the stars whose id is in ids, and stops at fn's first error. The
// rows stream from the cursor: a whole export never sits in memory.
func (r *Reader) EachDemoStar(ctx context.Context, sort string, desc bool, ids []int, fn func(demo.Star) error) error {
	where, args := "", make([]any, len(ids))
	if len(ids) > 0 {
		ctx = sqlite.Uncached(ctx) // a text per length of the list
		where = ` WHERE id IN (?` + strings.Repeat(", ?", len(ids)-1) + `)`
		for i, id := range ids {
			args[i] = id
		}
	}
	rows, err := r.tx.QueryContext(ctx, `SELECT id, name, class, temp, constellation, distance, magnitude, planets FROM demo_stars`+where+`
		ORDER BY `+starOrderBy(sort, desc), args...)
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		var s demo.Star
		if err := rows.Scan(&s.ID, &s.Name, &s.Class, &s.Temp, &s.Constellation, &s.Distance, &s.Magnitude, &s.Planets); err != nil {
			return err
		}
		if err := fn(s); err != nil {
			return err
		}
	}
	return rows.Err()
}
