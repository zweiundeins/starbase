package web

import (
	"bytes"
	"context"
	"net/http"
	"time"

	"github.com/starfederation/datastar-go/datastar"
	"github.com/zweiundeins/lith/edge"
	"github.com/zweiundeins/lith/live"

	"starbase/internal/commands"
	"starbase/internal/queries"
	"starbase/internal/ui"
)

// boardCache keeps the encoded board of one version for the board's view
// and the Showcase's documents: only the first render after a paint reads
// the cells.
type boardCache struct {
	version int64
	cells   string
}

// boardState returns the encoded board and its total painted pixels.
func (s *Server) boardState(ctx context.Context, r *queries.Reader) (cells string, pixels int64, err error) {
	version, pixels, err := r.BoardMeta(ctx, commands.BoardName)
	if err != nil {
		return "", 0, err
	}
	if c := s.board.Load(); c != nil && c.version == version {
		return c.cells, pixels, nil
	}
	cells, err = r.BoardCells(ctx, commands.BoardName, commands.BoardSize)
	if err != nil {
		return "", 0, err
	}
	s.board.Store(&boardCache{version: version, cells: cells})
	return cells, pixels, nil
}

// newBoardRegion makes the Showcase's board (ui.Board): who is watching,
// the pixels painted and the cells. Its view's key and topic are
// commands.TopicBoard, so a paint renders it alone, once for every viewer,
// and a join or a leave renders it again for the viewer count.
func (s *Server) newBoardRegion() *region {
	v := live.View{Key: commands.TopicBoard, Topics: []string{commands.TopicBoard}, Presence: true}
	return newRegion(v, `<header id="board-head" class="mission__head"></header><div id="board-grid" class="board-section__grid"></div>`, s.boardHTML)
}

// boardHTML renders the board's region in rd's snapshot.
func (s *Server) boardHTML(ctx context.Context, rd *queries.Reader) ([]byte, error) {
	cells, pixels, err := s.boardState(ctx, rd)
	if err != nil {
		return nil, err
	}
	var buf bytes.Buffer
	v := ui.BoardView{Cells: cells, Size: commands.BoardSize, Pixels: pixels, Viewers: s.viewers()}
	if err := ui.Board(v).Render(ctx, &buf); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// viewers is the number of open streams on the Showcase, at least one: the
// page being rendered, whose stream may not be open yet.
func (s *Server) viewers() int { return max(1, s.hub.Count(commands.TopicBoard)) }

// cmdPaint paints cells of the shared board (anyone may paint, rate-limited).
func (s *Server) cmdPaint(w http.ResponseWriter, r *http.Request) {
	var p struct {
		Color int   `json:"color"`
		Cells []int `json:"cells"`
	}
	if err := datastar.ReadSignals(r, &p); err != nil {
		http.Error(w, "bad payload", http.StatusBadRequest)
		return
	}
	now := time.Now()
	if !s.paintLimitIP.Allow(edge.ClientKey(r), len(p.Cells), now) || !s.paintLimit.Allow(edge.SessionID(r.Context()), len(p.Cells), now) {
		http.Error(w, "painting too fast", http.StatusTooManyRequests)
		return
	}
	s.send(w, r, commands.PaintPixels{Board: commands.BoardName, Color: p.Color, Cells: p.Cells})
}
