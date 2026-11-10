package commands

import (
	"errors"
	"time"

	"github.com/zweiundeins/lith/writer"
)

// Star records that a signed-in user starred a component. Idempotent. A new
// star touches TopicAll: every page that lists the component shows its count.
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

func (c Star) Apply(tx *writer.Tx) error {
	res, err := tx.Exec(`
		INSERT INTO stars (slug, user_id, created_at)
		SELECT slug, ?, ? FROM components WHERE slug = ? AND active
		ON CONFLICT DO NOTHING`, c.UserID, time.Now().Unix(), c.Slug)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n == 1 {
		tx.Touch(TopicAll)
		_, err = tx.Exec(`UPDATE components SET stars = stars + 1 WHERE slug = ?`, c.Slug)
	}
	return err
}

// Unstar removes a star. Idempotent.
type Unstar struct {
	UserID int64
	Slug   string
}

func (c Unstar) Validate() error { return Star(c).Validate() }

func (c Unstar) Apply(tx *writer.Tx) error {
	res, err := tx.Exec(`DELETE FROM stars WHERE slug = ? AND user_id = ?`, c.Slug, c.UserID)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n == 1 {
		tx.Touch(TopicAll)
		_, err = tx.Exec(`UPDATE components SET stars = stars - 1 WHERE slug = ?`, c.Slug)
	}
	return err
}
