package commands

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"starbase/internal/pixelart"
)

// The one shared board.
const (
	BoardName = "main"
	BoardSize = 48
	MaxPaint  = 64 // cells per command
)

// PaintPixels paints cells of the shared board. Not Scoped: every viewer's
// stream re-renders (pages without the board produce identical frames,
// which the stream skips).
type PaintPixels struct {
	Board string
	Color int
	Cells []int
}

func (c PaintPixels) Validate() error {
	if c.Board != BoardName {
		return errors.New("unknown board")
	}
	if c.Color < 0 || c.Color > 15 {
		return errors.New("colour must be 0-15")
	}
	if len(c.Cells) == 0 || len(c.Cells) > MaxPaint {
		return errors.New("paint 1 to 64 cells at a time")
	}
	for _, i := range c.Cells {
		if i < 0 || i >= BoardSize*BoardSize {
			return errors.New("cell out of range")
		}
	}
	return nil
}

func (c PaintPixels) Apply(ctx context.Context, tx *sql.Tx) error {
	now := time.Now().Unix()
	for _, i := range c.Cells {
		if _, err := tx.ExecContext(ctx, `
			INSERT INTO board_cells (board, idx, color, painted_at) VALUES (?, ?, ?, ?)
			ON CONFLICT (board, idx) DO UPDATE SET color = excluded.color, painted_at = excluded.painted_at`,
			c.Board, i, c.Color, now); err != nil {
			return err
		}
	}
	_, err := tx.ExecContext(ctx, `
		INSERT INTO boards (board, version, pixels) VALUES (?, 1, ?)
		ON CONFLICT (board) DO UPDATE SET version = version + 1, pixels = pixels + excluded.pixels`,
		c.Board, len(c.Cells))
	return err
}

// SeedBoard paints a pixel rocket and a few stars onto an empty board, so
// the first visitor doesn't face a blank void. A no-op once the board exists.
type SeedBoard struct{}

func (SeedBoard) Scope() string { return "" } // nobody is watching yet

// palette indices of sb-pixel-board for the sprite's colour keys
var seedColors = map[rune]int{'D': 1, 'W': 5, 'G': 4, 'S': 3, 'R': 15, 'r': 15, 'P': 6, 'B': 9, 'b': 10, 'Y': 13, 'O': 14, 'F': 15}

func (SeedBoard) Apply(ctx context.Context, tx *sql.Tx) error {
	var n int
	if err := tx.QueryRowContext(ctx, `SELECT count(*) FROM boards WHERE board = ?`, BoardName).Scan(&n); err != nil || n > 0 {
		return err
	}
	cells := map[int]int{}
	rows := pixelart.RocketRows()
	ox, oy := (BoardSize-len(rows[0]))/2, (BoardSize-len(rows))/2
	for y, row := range rows {
		for x, ch := range row {
			if c, ok := seedColors[ch]; ok {
				cells[(oy+y)*BoardSize+ox+x] = c
			}
		}
	}
	for _, s := range [][3]int{{5, 6, 5}, {40, 4, 13}, {9, 30, 7}, {42, 22, 5}, {30, 41, 13}, {14, 44, 9}, {37, 35, 7}, {22, 3, 5}} {
		cells[s[1]*BoardSize+s[0]] = s[2] // x, y, colour
	}
	now := time.Now().Unix()
	for i, c := range cells {
		if _, err := tx.ExecContext(ctx, `INSERT INTO board_cells (board, idx, color, painted_at) VALUES (?, ?, ?, ?)`, BoardName, i, c, now); err != nil {
			return err
		}
	}
	_, err := tx.ExecContext(ctx, `INSERT INTO boards (board, version, pixels) VALUES (?, 1, 0)`, BoardName)
	return err
}
