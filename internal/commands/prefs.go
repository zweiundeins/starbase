package commands

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"time"

	"starbase/internal/model"
)

// updatePrefs reads the session's preferences (or none), applies fn and
// writes them back.
func updatePrefs(ctx context.Context, tx *sql.Tx, sid string, fn func(*model.SessionPrefs)) error {
	var p model.SessionPrefs
	var raw string
	err := tx.QueryRowContext(ctx, `SELECT data FROM session_prefs WHERE sid = ?`, sid).Scan(&raw)
	switch {
	case errors.Is(err, sql.ErrNoRows):
	case err != nil:
		return err
	default:
		if err := json.Unmarshal([]byte(raw), &p); err != nil {
			p = model.SessionPrefs{}
		}
	}
	fn(&p)
	b, err := json.Marshal(p)
	if err != nil {
		return err
	}
	_, err = tx.ExecContext(ctx, `
		INSERT INTO session_prefs (sid, data, updated_at) VALUES (?, ?, ?)
		ON CONFLICT (sid) DO UPDATE SET data = excluded.data, updated_at = excluded.updated_at`,
		sid, string(b), time.Now().Unix())
	return err
}

// SetInstallTab remembers the installation tab chosen on a component page,
// for every component page of the session.
type SetInstallTab struct {
	SID, Tab string
}

func (c SetInstallTab) Validate() error {
	if c.SID == "" {
		return errors.New("no session")
	}
	if !model.ValidInstallTab(c.Tab) {
		return errors.New("unknown installation tab")
	}
	return nil
}

func (c SetInstallTab) Scope() string { return c.SID }

func (c SetInstallTab) Apply(ctx context.Context, tx *sql.Tx) error {
	return updatePrefs(ctx, tx, c.SID, func(p *model.SessionPrefs) { p.InstallTab = c.Tab })
}

// PruneSessionPrefs forgets the preferences of sessions idle for a while
// (the session cookie itself lasts 400 days).
type PruneSessionPrefs struct{ OlderThan time.Duration }

func (c PruneSessionPrefs) Scope() string { return "" } // touches no live view

func (c PruneSessionPrefs) Apply(ctx context.Context, tx *sql.Tx) error {
	_, err := tx.ExecContext(ctx, `DELETE FROM session_prefs WHERE updated_at < ?`, time.Now().Add(-c.OlderThan).Unix())
	return err
}
