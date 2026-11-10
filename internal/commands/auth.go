package commands

import (
	"errors"
	"time"

	"github.com/zweiundeins/lith/writer"

	"starbase/internal/model"
)

// SignIn upserts a GitHub user and binds the browser session to them.
type SignIn struct {
	SID  string
	User model.User // GitHubID, Login, Name, AvatarURL
}

func (c SignIn) Validate() error {
	if c.SID == "" || c.User.GitHubID == 0 || c.User.Login == "" {
		return errors.New("incomplete sign-in")
	}
	return nil
}

func (c SignIn) Apply(tx *writer.Tx) error {
	tx.Touch(SessionTopic(c.SID))
	now := time.Now().Unix()
	var id int64
	err := tx.QueryRow(`
		INSERT INTO users (github_id, login, name, avatar_url, created_at) VALUES (?, ?, ?, ?, ?)
		ON CONFLICT (github_id) DO UPDATE SET login = excluded.login, name = excluded.name, avatar_url = excluded.avatar_url
		RETURNING id`,
		c.User.GitHubID, c.User.Login, c.User.Name, c.User.AvatarURL, now).Scan(&id)
	if err != nil {
		return err
	}
	_, err = tx.Exec(`
		INSERT INTO sessions (sid, user_id, created_at) VALUES (?, ?, ?)
		ON CONFLICT (sid) DO UPDATE SET user_id = excluded.user_id`, c.SID, id, now)
	return err
}

// SignOut unbinds the session.
type SignOut struct{ SID string }

func (c SignOut) Apply(tx *writer.Tx) error {
	tx.Touch(SessionTopic(c.SID))
	_, err := tx.Exec(`DELETE FROM sessions WHERE sid = ?`, c.SID)
	return err
}
