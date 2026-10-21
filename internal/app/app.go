// Package app wires the application together: catalog, database, writer
// and HTTP handler. Used by the server binary and by tools that need the
// real app in-process (cmd/manifests).
package app

import (
	"context"
	"database/sql"
	"log/slog"
	"net/http"
	"path/filepath"
	"time"

	"github.com/klauspost/compress/zstd"
	"github.com/zweiundeins/lith/live"
	"github.com/zweiundeins/lith/sse"
	"github.com/zweiundeins/lith/writer"

	"starbase/components"
	"starbase/content"
	"starbase/internal/catalog"
	"starbase/internal/commands"
	"starbase/internal/config"
	"starbase/internal/db"
	"starbase/internal/queries"
	"starbase/internal/tscheck"
	"starbase/internal/ui"
	"starbase/internal/web"
	"starbase/static"
)

// MemoryLimit is the GC's soft limit, which main sets unless GOMEMLIMIT is
// set. Without it the heap grows to twice what is live: 150 streams took
// 449 to 494 MiB while the board was painted, 247 to 263 MiB with it.
const MemoryLimit = 256 << 20

// streamBudget caps the render streams by memory, not by a count: what
// they may hold together as lith's budget books it, each stream's encoder,
// 32 KiB besides and every view's newest frame. A tab without room for a
// window that holds its first frame gets 503, and its page asks again
// (retry 'error'). The rest of MemoryHigh is for the app, the garbage
// between collections and a type check (about 65 MiB): with 150 MiB of
// zstd streams the server alone reached 317 MiB resident, against
// MemoryHigh's 320.
const streamBudget = 120 << 20

// HubOptions configure the render loop. The budget refuses the streams it
// has no room for, so every stream it lets in compresses with a window
// that holds its frame, and live's MaxStreams (2000) is only a backstop.
// Each stream gets the smallest window that holds its largest frame, up to
// 2^19. Every heartbeat patches $_ping, which the page's watchdog
// (ui.Watchdog) waits for. Each call has a budget of its own.
func HubOptions(read *sql.DB, log *slog.Logger, heartbeat time.Duration) live.Options {
	return live.Options{Read: read, Log: log, Heartbeat: heartbeat, HeartbeatSignal: ui.PingSignal, SSE: sse.Options{
		ZstdLevel: zstd.SpeedFastest, ZstdWindow: 19, ZstdLowMemory: true,
		BrotliWindow: 19,
		Budget:       sse.NewBudget(sse.BudgetOptions{Limit: streamBudget, Refuse: true}),
	}}
}

type App struct {
	Handler http.Handler
	Catalog *catalog.Catalog
	// Writer and Hub are for the metrics; the hub's Budget is HubOptions'.
	Writer *writer.Writer
	Hub    *live.Hub
	close  func()
}

// Close ends the render streams, applies the commands still queued, stops
// the writer and closes the database. Call it after the HTTP server has shut
// down.
func (a *App) Close() { a.close() }

// Settle returns once the commands sent before it, such as the startup
// seeding, are applied.
func (a *App) Settle(ctx context.Context) error { return a.Writer.Flush(ctx) }

// New builds the app. ctx bounds the lifetime of render streams: cancel it
// to end them (e.g. before http.Server.Shutdown).
func New(ctx context.Context, cfg config.Config, log *slog.Logger) (*App, error) {
	cat, err := catalog.Load(components.FS)
	if err != nil {
		return nil, err
	}
	database, err := db.Open(ctx, cfg.DBPath)
	if err != nil {
		return nil, err
	}
	hub := live.New(HubOptions(database.R, log, cfg.StreamHeartbeat()))
	w, err := writer.New(database, writer.Options{OnCommit: hub.Changed, Log: log})
	if err != nil {
		database.Close()
		return nil, err
	}
	// The writer outlives the HTTP server, so in-flight commands can finish:
	// Close applies what is queued.
	go w.Run(context.Background())
	hubCtx, stopHub := context.WithCancel(ctx)
	hubDone := make(chan struct{})
	go func() {
		defer close(hubDone)
		if err := hub.Run(hubCtx); err != nil {
			log.Error("render loop", "err", err)
		}
	}()
	closeAll := func() {
		stopHub()
		<-hubDone // its workers give their read connections back
		if err := w.Close(); err != nil {
			log.Warn("closing the writer", "err", err)
		}
		database.Close()
	}

	if err := w.Do(ctx, commands.SyncCatalog{Catalog: cat, Datastar: static.Datastar()}); err != nil {
		closeAll()
		return nil, err
	}
	w.Send(commands.PruneTabs{OlderThan: 30 * 24 * time.Hour})
	w.Send(commands.PruneSessionPrefs{OlderThan: 400 * 24 * time.Hour})
	w.Send(commands.SeedBoard{})
	w.Send(commands.SeedDemo{})
	w.Send(commands.SeedStars{})
	log.Info("catalog synced", "components", len(cat.Components), "hash", cat.Hash)

	// The type check's compiler unpacks next to the database on the first check.
	checker := tscheck.New(filepath.Dir(cfg.DBPath))
	srv := web.New(ctx, web.Deps{
		Config:   cfg,
		Log:      log,
		Writer:   w,
		Hub:      hub,
		Queries:  queries.New(database),
		Catalog:  cat,
		StaticFS: static.FS,
		Content:  content.FS,
		Checker:  checker,
	})
	return &App{Handler: srv.Handler(), Catalog: cat, Writer: w, Hub: hub, close: closeAll}, nil
}
