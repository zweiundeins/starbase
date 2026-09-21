package ui

import "starbase/internal/catalog"

// Code excerpts for the Showcase "How it works" sections, highlighted once.
var (
	telemetryGo = catalog.Highlight(`// internal/web/demo.go: a query stream, no state, no database.
func (s *Server) demoTelemetry(w http.ResponseWriter, r *http.Request) {
	sse := datastar.NewSSE(w, r)
	tick := time.NewTicker(250 * time.Millisecond)
	for {
		// A pure function of the clock: every viewer sees the same flight.
		sse.MarshalAndPatchSignals(map[string]any{"_tm": TelemetryAt(time.Now())})
		select {
		case <-tick.C:
		case <-r.Context().Done():
			return
		}
	}
}`, "go")

	telemetryHTML = catalog.Highlight(`<section data-ignore-morph
  data-signals="{_tm: {alt: 0, vel: 0, fuel: 100, temp: 18, pitch: 90}}"
  data-init="@get('/demo/telemetry')">

  <sb-starfield data-attr:speed="Math.round(3 + $_tm.vel * 11)"
                data-attr:warp="$_tm.vel > 5"></sb-starfield>
  <sb-gauge label="Velocity" unit=" km/s" max="8" decimals="2"
            data-attr:value="$_tm.vel"></sb-gauge>
  <sb-sparkline data-attr:value="$_tm.alt" length="80" show-value></sb-sparkline>
  <sb-meter label="Fuel" warn="30" danger="15"
            data-attr:value="Math.round($_tm.fuel)"></sb-meter>
</section>`, "html")

	boardHTML = catalog.Highlight(`<!-- rendered by the server into every frame of the page's stream -->
<sb-pixel-board id="board" size="48" cells="0000…5ff5…0d00"
  data-on:sb-paint="@post('/cmd/paint', {
    payload: {color: evt.detail.color, cells: evt.detail.cells},
    requestCancellation: 'disabled'
  })"></sb-pixel-board>`, "html")

	boardGo = catalog.Highlight(`// internal/commands/board.go: runs inside the single writer's batch.
func (c PaintPixels) Apply(ctx context.Context, tx *sql.Tx) error {
	for _, i := range c.Cells {
		tx.ExecContext(ctx, `+"`"+`INSERT INTO board_cells (board, idx, color, painted_at)
			VALUES (?, ?, ?, ?) ON CONFLICT (board, idx) DO UPDATE
			SET color = excluded.color, painted_at = excluded.painted_at`+"`"+`,
			c.Board, i, c.Color, now)
	}
	// bump the version: every open stream re-renders once the batch commits
	...
}`, "go")
)
