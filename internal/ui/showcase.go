package ui

import "starbase/internal/catalog"

// Code excerpts for the Showcase "How it works" sections, highlighted once.
var (
	telemetryGo = catalog.Highlight(`// internal/web/demo.go: a query stream, no state, no database.
func (s *Server) demoTelemetry(w http.ResponseWriter, r *http.Request) {
	stream, err := sse.Open(w, r, sse.Options{})
	if err != nil {
		return
	}
	var last [][]byte
	defer func() { stream.CloseWith(last...) }()
	tick := time.NewTicker(250 * time.Millisecond)
	for {
		// A pure function of the clock: every viewer sees the same flight.
		tm, _ := json.Marshal(map[string]any{"_tm": TelemetryAt(time.Now())})
		stream.Write(sse.PatchSignals(tm, false))
		select {
		case <-tick.C:
		case <-r.Context().Done():
			return
		case <-s.ctx.Done(): // a deploy: the page opens the stream again
			last = append(last, sse.Reconnect("telemetry"))
			return
		}
	}
}`, "go")

	telemetryHTML = catalog.Highlight(`<section data-ignore-morph
  data-signals="{_tm: {alt: 0, vel: 0, fuel: 100, temp: 18, pitch: 90}}">
  <div id="telemetry" hidden
       data-init="@get('/demo/telemetry', {retry: 'error', retryMaxCount: Infinity, retryInterval: 400, retryMaxWait: 4000})"></div>

  <sb-starfield data-attr:speed="Math.round(3 + $_tm.vel * 11)"
                data-attr:warp="$_tm.vel > 5"></sb-starfield>
  <sb-gauge label="Velocity" unit=" km/s" max="8" decimals="2"
            data-attr:value="$_tm.vel"></sb-gauge>
  <sb-sparkline label="Altitude" data-attr:value="$_tm.alt" length="80" show-value></sb-sparkline>
  <sb-meter label="Fuel" warn="30" danger="15"
            data-attr:value="Math.round($_tm.fuel)"></sb-meter>
</section>`, "html")

	boardHTML = catalog.Highlight(`<!-- rendered by the board's view, one frame for every viewer -->
<sb-pixel-board id="board" size="48" cells="0000…5ff5…0d00"
  data-on:sb-paint="@post('/cmd/paint', {
    payload: {color: evt.detail.color, cells: evt.detail.cells},
    requestCancellation: 'disabled'
  })"
  data-on:datastar-fetch="evt.detail.el === el &&
    ['error', 'retries-failed'].includes(evt.detail.type) && el.revert()"></sb-pixel-board>`, "html")

	boardGo = catalog.Highlight(`// internal/commands/board.go: runs inside the single writer's batch.
func (c PaintPixels) Apply(tx *writer.Tx) error {
	tx.Touch(TopicBoard) // the board's view renders again once the batch commits
	for _, i := range c.Cells {
		tx.Exec(`+"`"+`INSERT INTO board_cells (board, idx, color, painted_at)
			VALUES (?, ?, ?, ?) ON CONFLICT (board, idx) DO UPDATE
			SET color = excluded.color, painted_at = excluded.painted_at`+"`"+`,
			c.Board, i, c.Color, now)
	}
	// bump the board's version, which its encoded cells are cached by
	...
}`, "go")
)
