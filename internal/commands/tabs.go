package commands

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"regexp"
	"time"

	"starbase/internal/model"
)

var tabIDRe = regexp.MustCompile(`^[a-z0-9]{8,32}$`)

func validTab(sid, tabID string) error {
	if sid == "" || !tabIDRe.MatchString(tabID) {
		return errors.New("bad tab")
	}
	return nil
}

// updateTab reads the tab's state (or seed when absent), applies fn and
// writes it back.
func updateTab(ctx context.Context, tx *sql.Tx, sid, tabID string, seed model.TabState, fn func(*model.TabState)) error {
	st := seed
	var raw string
	err := tx.QueryRowContext(ctx, `SELECT data FROM tab_state WHERE sid = ? AND tab_id = ?`, sid, tabID).Scan(&raw)
	switch {
	case errors.Is(err, sql.ErrNoRows):
	case err != nil:
		return err
	default:
		if err := json.Unmarshal([]byte(raw), &st); err != nil {
			st = seed
		}
	}
	fn(&st)
	b, err := json.Marshal(st)
	if err != nil {
		return err
	}
	_, err = tx.ExecContext(ctx, `
		INSERT INTO tab_state (sid, tab_id, data, updated_at) VALUES (?, ?, ?, ?)
		ON CONFLICT (sid, tab_id) DO UPDATE SET data = excluded.data, updated_at = excluded.updated_at`,
		sid, tabID, string(b), time.Now().Unix())
	return err
}

// SetBrowseFilter stores the gallery's search, category and sort for a tab.
type SetBrowseFilter struct {
	SID, TabID string
	Browse     model.Browse
}

func (c SetBrowseFilter) Validate() error { return validTab(c.SID, c.TabID) }
func (c SetBrowseFilter) Scope() string   { return c.SID }

func (c SetBrowseFilter) Apply(ctx context.Context, tx *sql.Tx) error {
	return updateTab(ctx, tx, c.SID, c.TabID, model.TabState{}, func(st *model.TabState) {
		st.Browse = c.Browse.Normalize()
	})
}

// PruneTabs forgets tab state that has not been touched for a while.
type PruneTabs struct{ OlderThan time.Duration }

func (c PruneTabs) Scope() string { return "" } // touches no live view

func (c PruneTabs) Apply(ctx context.Context, tx *sql.Tx) error {
	_, err := tx.ExecContext(ctx, `DELETE FROM tab_state WHERE updated_at < ?`, time.Now().Add(-c.OlderThan).Unix())
	return err
}
