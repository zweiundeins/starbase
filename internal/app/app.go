// Package app wires the application together: catalog, database, command
// bus and HTTP handler. Used by the server binary and by tools that need the
// real app in-process (cmd/manifests).
package app

import (
	"context"
	"log/slog"
	"net/http"
	"time"

	"starbase/components"
	"starbase/content"
	"starbase/internal/catalog"
	"starbase/internal/commands"
	"starbase/internal/config"
	"starbase/internal/cqrs"
	"starbase/internal/db"
	"starbase/internal/queries"
	"starbase/internal/web"
	"starbase/static"
)

type App struct {
	Handler http.Handler
	Catalog *catalog.Catalog
	close   func()
}

// Close stops the command bus and closes the database. Call it after the
// HTTP server has shut down.
func (a *App) Close() { a.close() }

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

	if err := bus.Exec(ctx, commands.SyncCatalog{Catalog: cat}); err != nil {
		closeAll()
		return nil, err
	}
	bus.Send(commands.PruneTabs{OlderThan: 30 * 24 * time.Hour})
	bus.Send(commands.PruneSessionPrefs{OlderThan: 400 * 24 * time.Hour})
	bus.Send(commands.SeedBoard{})
	bus.Send(commands.SeedDemo{})
	log.Info("catalog synced", "components", len(cat.Components), "hash", cat.Hash)

	srv := web.New(ctx, web.Deps{
		Config:   cfg,
		Log:      log,
		Bus:      bus,
		Hub:      hub,
		Queries:  queries.New(database.R),
		Catalog:  cat,
		StaticFS: static.FS,
		Content:  content.FS,
	})
	return &App{Handler: srv.Handler(), Catalog: cat, close: closeAll}, nil
}
