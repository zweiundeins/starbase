package commands

import (
	"context"
	"database/sql"
	"errors"
	"time"
)

// Star records that a signed-in user starred a component. Idempotent.
type Star struct {
	UserID int64
	Slug   string
}

func (c Star) Validate() error {
	if c.UserID == 0 {
		return errors.New("sign in to star components")
	}
	if c.Slug == "" {
		return errors.New("missing component")
	}
	return nil
}

func (c Star) Apply(ctx context.Context, tx *sql.Tx) error {
	res, err := tx.ExecContext(ctx, `
		INSERT INTO stars (slug, user_id, created_at)
		SELECT slug, ?, ? FROM components WHERE slug = ? AND active
		ON CONFLICT DO NOTHING`, c.UserID, time.Now().Unix(), c.Slug)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n == 1 {
		_, err = tx.ExecContext(ctx, `UPDATE components SET stars = stars + 1 WHERE slug = ?`, c.Slug)
	}
	return err
}

// Unstar removes a star. Idempotent.
type Unstar struct {
	UserID int64
	Slug   string
}

func (c Unstar) Validate() error { return Star(c).Validate() }

func (c Unstar) Apply(ctx context.Context, tx *sql.Tx) error {
	res, err := tx.ExecContext(ctx, `DELETE FROM stars WHERE slug = ? AND user_id = ?`, c.Slug, c.UserID)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n == 1 {
		_, err = tx.ExecContext(ctx, `UPDATE components SET stars = stars - 1 WHERE slug = ?`, c.Slug)
	}
	return err
}
