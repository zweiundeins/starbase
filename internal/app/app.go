// Package app wires the application together: catalog, database, command
// bus and HTTP handler. Used by the server binary and by tools that need the
// real app in-process (cmd/manifests).
package app

import (
	"context"
	"database/sql"
	"log/slog"
	"net/http"
	"path/filepath"
	"time"

	"starbase/components"
	"starbase/content"
	"starbase/internal/catalog"
	"starbase/internal/commands"
	"starbase/internal/config"
	"starbase/internal/cqrs"
	"starbase/internal/db"
	"starbase/internal/queries"
	"starbase/internal/tscheck"
	"starbase/internal/web"
	"starbase/static"
)

type App struct {
	Handler http.Handler
	Catalog *catalog.Catalog
	bus     *cqrs.Bus
	close   func()
}

// Close stops the command bus and closes the database. Call it after the
// HTTP server has shut down.
func (a *App) Close() { a.close() }

// Settle returns once the commands sent before it, such as the startup
// seeding, are applied: the bus applies them in order.
func (a *App) Settle(ctx context.Context) error { return a.bus.Exec(ctx, settled{}) }

// settled changes nothing; Settle waits for it.
type settled struct{}

func (settled) Apply(context.Context, *sql.Tx) error { return nil }
func (settled) Scope() string                        { return "" } // touches no view

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
	hub := cqrs.NewHub()
	bus := cqrs.NewBus(database.W, hub, log)
	// The bus outlives the HTTP server so in-flight commands can finish.
	busCtx, stopBus := context.WithCancel(context.Background())
	busDone := make(chan struct{})
	go func() { bus.Run(busCtx); close(busDone) }()
	closeAll := func() {
		stopBus()
		<-busDone
		database.Close()
	}

	if err := bus.Exec(ctx, commands.SyncCatalog{Catalog: cat, Datastar: static.Datastar()}); err != nil {
		closeAll()
		return nil, err
	}
	bus.Send(commands.PruneTabs{OlderThan: 30 * 24 * time.Hour})
	bus.Send(commands.PruneSessionPrefs{OlderThan: 400 * 24 * time.Hour})
	bus.Send(commands.SeedBoard{})
	bus.Send(commands.SeedDemo{})
	bus.Send(commands.SeedStars{})
	log.Info("catalog synced", "components", len(cat.Components), "hash", cat.Hash)

	// The type check's compiler unpacks next to the database on the first check.
	checker := tscheck.New(filepath.Dir(cfg.DBPath))
	srv := web.New(ctx, web.Deps{
		Config:   cfg,
		Log:      log,
		Bus:      bus,
		Hub:      hub,
		Queries:  queries.New(database.R),
		Catalog:  cat,
		StaticFS: static.FS,
		Content:  content.FS,
		Checker:  checker,
	})
	return &App{Handler: srv.Handler(), Catalog: cat, bus: bus, close: closeAll}, nil
}
