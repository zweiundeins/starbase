// Package queries is the read side. Every render of a view runs its queries
// inside one read transaction, so a view always reflects one consistent
// snapshot of the database.
package queries

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"strings"
	"unicode"

	"starbase/internal/model"
)

type Queries struct{ db *sql.DB }

func New(db *sql.DB) *Queries { return &Queries{db: db} }

// Reader runs queries against one snapshot.
type Reader struct{ tx *sql.Tx }

// View runs fn inside a read transaction.
func (q *Queries) View(ctx context.Context, fn func(*Reader) error) error {
	tx, err := q.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	return fn(&Reader{tx: tx})
}

// SessionUser returns the user bound to sid, or nil.
func (r *Reader) SessionUser(ctx context.Context, sid string) (*model.User, error) {
	if sid == "" {
		return nil, nil
	}
	var u model.User
	err := r.tx.QueryRowContext(ctx, `
		SELECT u.id, u.github_id, u.login, u.name, u.avatar_url
		FROM sessions s JOIN users u ON u.id = s.user_id WHERE s.sid = ?`, sid).
		Scan(&u.ID, &u.GitHubID, &u.Login, &u.Name, &u.AvatarURL)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &u, nil
}

// Tab returns the stored state of a tab; ok is false if nothing is stored.
func (r *Reader) Tab(ctx context.Context, sid, tabID string) (st model.TabState, ok bool, err error) {
	if sid == "" || tabID == "" {
		return st, false, nil
	}
	var raw string
	err = r.tx.QueryRowContext(ctx, `SELECT data FROM tab_state WHERE sid = ? AND tab_id = ?`, sid, tabID).Scan(&raw)
	if errors.Is(err, sql.ErrNoRows) {
		return st, false, nil
	}
	if err != nil {
		return st, false, err
	}
	if err := json.Unmarshal([]byte(raw), &st); err != nil {
		return model.TabState{}, false, nil
	}
	return st, true, nil
}

// Card is one component in the gallery.
type Card struct {
	Slug     string
	Tag      string
	Name     string
	Category string
	Summary  string
	Author   string
	Tags     []string
	Since    string
	Stars    int
	Starred  bool // by the current user
}

const cardCols = `c.slug, c.tag, c.name, c.category, c.summary, c.author, c.tags, c.since, c.stars,
	EXISTS (SELECT 1 FROM stars s WHERE s.slug = c.slug AND s.user_id = ?)`

func scanCard(sc interface{ Scan(...any) error }) (Card, error) {
	var c Card
	var tags string
	err := sc.Scan(&c.Slug, &c.Tag, &c.Name, &c.Category, &c.Summary, &c.Author, &tags, &c.Since, &c.Stars, &c.Starred)
	if err == nil {
		json.Unmarshal([]byte(tags), &c.Tags)
	}
	return c, err
}

// FTSQuery turns free text into a safe FTS5 prefix query: every word must
// match as a prefix. Returns "" when nothing searchable remains.
func FTSQuery(q string) string {
	words := strings.FieldsFunc(strings.ToLower(q), func(r rune) bool {
		return !unicode.IsLetter(r) && !unicode.IsDigit(r)
	})
	for i, w := range words {
		words[i] = `"` + w + `"*`
	}
	return strings.Join(words, " ")
}

type BrowseResult struct {
	Cards  []Card
	Counts map[string]int // per category, for the current search
	Total  int            // all categories, for the current search
}

// Browse lists active components matching the filter.
func (r *Reader) Browse(ctx context.Context, b model.Browse, userID int64) (BrowseResult, error) {
	res := BrowseResult{Counts: map[string]int{}}
	where := []string{"c.active"}
	var args []any
	if fts := FTSQuery(b.Q); fts != "" {
		where = append(where, "c.slug IN (SELECT slug FROM components_fts WHERE components_fts MATCH ?)")
		args = append(args, fts)
	} else if strings.TrimSpace(b.Q) != "" {
		return res, nil // only punctuation: nothing matches
	}
	filter := strings.Join(where, " AND ")

	rows, err := r.tx.QueryContext(ctx, `SELECT c.category, count(*) FROM components c WHERE `+filter+` GROUP BY c.category`, args...)
	if err != nil {
		return res, err
	}
	for rows.Next() {
		var cat string
		var n int
		if err := rows.Scan(&cat, &n); err != nil {
			rows.Close()
			return res, err
		}
		res.Counts[cat] = n
		res.Total += n
	}
	rows.Close()

	if b.Category != "" {
		filter += " AND c.category = ?"
		args = append(args, b.Category)
	}
	order := "c.stars DESC, c.name COLLATE NOCASE"
	switch b.Sort {
	case model.SortNewest:
		order = "c.since DESC, c.name COLLATE NOCASE"
	case model.SortName:
		order = "c.name COLLATE NOCASE"
	}
	rows, err = r.tx.QueryContext(ctx, `SELECT `+cardCols+` FROM components c WHERE `+filter+` ORDER BY `+order,
		append([]any{userID}, args...)...)
	if err != nil {
		return res, err
	}
	defer rows.Close()
	for rows.Next() {
		c, err := scanCard(rows)
		if err != nil {
			return res, err
		}
		res.Cards = append(res.Cards, c)
	}
	return res, rows.Err()
}

// Component returns one active component, or nil.
func (r *Reader) Component(ctx context.Context, slug string, userID int64) (*Card, error) {
	c, err := scanCard(r.tx.QueryRowContext(ctx, `SELECT `+cardCols+` FROM components c WHERE c.slug = ? AND c.active`, userID, slug))
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &c, nil
}

type Stats struct {
	Components, Authors, Stars int
}

func (r *Reader) Stats(ctx context.Context) (Stats, error) {
	var s Stats
	err := r.tx.QueryRowContext(ctx, `
		SELECT count(*), count(DISTINCT author), coalesce(sum(stars), 0) FROM components WHERE active`).
		Scan(&s.Components, &s.Authors, &s.Stars)
	return s, err
}

type Snippet struct {
	ID        string
	Files     map[string]string
	Component string
	Author    string // GitHub login, if saved while signed in
}

// Snippet returns a saved playground snippet, or nil.
func (r *Reader) Snippet(ctx context.Context, id string) (*Snippet, error) {
	var sn Snippet
	var files string
	var comp, author sql.NullString
	err := r.tx.QueryRowContext(ctx, `
		SELECT s.id, s.files, s.component, u.login
		FROM snippets s LEFT JOIN users u ON u.id = s.user_id WHERE s.id = ?`, id).
		Scan(&sn.ID, &files, &comp, &author)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if err := json.Unmarshal([]byte(files), &sn.Files); err != nil {
		return nil, err
	}
	sn.Component, sn.Author = comp.String, author.String
	return &sn, nil
}

// BoardMeta returns a board's version and total painted pixels.
func (r *Reader) BoardMeta(ctx context.Context, board string) (version, pixels int64, err error) {
	err = r.tx.QueryRowContext(ctx, `SELECT version, pixels FROM boards WHERE board = ?`, board).Scan(&version, &pixels)
	if errors.Is(err, sql.ErrNoRows) {
		return 0, 0, nil
	}
	return
}

// BoardCells encodes a board as one hex digit (palette index) per cell,
// row by row: the value of sb-pixel-board's cells attribute.
func (r *Reader) BoardCells(ctx context.Context, board string, size int) (string, error) {
	cells := []byte(strings.Repeat("0", size*size))
	rows, err := r.tx.QueryContext(ctx, `SELECT idx, color FROM board_cells WHERE board = ?`, board)
	if err != nil {
		return "", err
	}
	defer rows.Close()
	const digits = "0123456789abcdef"
	for rows.Next() {
		var idx, color int
		if err := rows.Scan(&idx, &color); err != nil {
			return "", err
		}
		if idx >= 0 && idx < len(cells) && color >= 0 && color < 16 {
			cells[idx] = digits[color]
		}
	}
	return string(cells), rows.Err()
}
