package queries

import (
	"context"
	"strings"

	"starbase/internal/demo"
)

// DemoBody is a body of the example dataset, with how many children it has.
type DemoBody struct {
	ID, Parent, Kind, Name, Detail string
	Children                       int
}

const demoCols = `b.id, b.parent, b.kind, b.name, b.detail, (SELECT count(*) FROM demo_bodies c WHERE c.parent = b.id)`

func (r *Reader) demoBodies(ctx context.Context, query string, args ...any) ([]DemoBody, error) {
	rows, err := r.tx.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []DemoBody{}
	for rows.Next() {
		var b DemoBody
		if err := rows.Scan(&b.ID, &b.Parent, &b.Kind, &b.Name, &b.Detail, &b.Children); err != nil {
			return nil, err
		}
		out = append(out, b)
	}
	return out, rows.Err()
}

// DemoChildren lists a body's children in order ("" for the top level).
func (r *Reader) DemoChildren(ctx context.Context, parent string) ([]DemoBody, error) {
	return r.demoBodies(ctx, `SELECT `+demoCols+` FROM demo_bodies b WHERE b.parent = ? ORDER BY b.ord`, parent)
}

// DemoSearch finds bodies whose name (or detail) contains q, names that
// start with it first, optionally only of the given kinds.
func (r *Reader) DemoSearch(ctx context.Context, q string, kinds []string, limit int) ([]DemoBody, error) {
	q = demo.Fold(strings.TrimSpace(q))
	if q == "" {
		return []DemoBody{}, nil
	}
	like := "%" + strings.NewReplacer(`\`, `\\`, `%`, `\%`, `_`, `\_`).Replace(q) + "%"
	sqlq := `SELECT ` + demoCols + ` FROM demo_bodies b WHERE b.folded LIKE ? ESCAPE '\'`
	args := []any{like}
	if len(kinds) > 0 {
		sqlq += ` AND b.kind IN (?` + strings.Repeat(",?", len(kinds)-1) + `)`
		for _, k := range kinds {
			args = append(args, k)
		}
	}
	// Prefix matches of the name first, then the rest, alphabetically.
	sqlq += ` ORDER BY instr(b.folded, ?) != 1, b.name LIMIT ?`
	args = append(args, q, limit)
	return r.demoBodies(ctx, sqlq, args...)
}
