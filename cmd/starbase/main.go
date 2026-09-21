// Command starbase serves the Rocket community component gallery.
package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
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

func main() {
	if err := run(); err != nil {
		slog.Error("fatal", "err", err)
		os.Exit(1)
	}
}

func run() error {
	cfg := config.Load()
	level := slog.LevelInfo
	if cfg.Dev {
		level = slog.LevelDebug
	}
	log := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: level}))

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	cat, err := catalog.Load(components.FS)
	if err != nil {
		return err
	}

	database, err := db.Open(ctx, cfg.DBPath)
	if err != nil {
		return err
	}
	defer database.Close()

	hub := cqrs.NewHub()
	bus := cqrs.NewBus(database.W, hub, log)
	// The bus outlives the HTTP server so in-flight commands can finish.
	busCtx, stopBus := context.WithCancel(context.Background())
	busDone := make(chan struct{})
	go func() { bus.Run(busCtx); close(busDone) }()
	defer func() { stopBus(); <-busDone }()

	if err := bus.Exec(ctx, commands.SyncCatalog{Catalog: cat}); err != nil {
		return err
	}
	bus.Send(commands.PruneTabs{OlderThan: 30 * 24 * time.Hour})
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
	httpSrv := &http.Server{
		Addr:              cfg.Addr,
		Handler:           srv.Handler(),
		ReadHeaderTimeout: 10 * time.Second,
		IdleTimeout:       120 * time.Second,
		// No WriteTimeout: render streams are long-lived.
	}

	errc := make(chan error, 1)
	go func() {
		log.Info("listening", "addr", cfg.Addr, "url", cfg.BaseURL, "dev", cfg.Dev)
		errc <- httpSrv.ListenAndServe()
	}()

	select {
	case err := <-errc:
		if !errors.Is(err, http.ErrServerClosed) {
			return err
		}
	case <-ctx.Done():
	}
	log.Info("shutting down")
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	return httpSrv.Shutdown(shutdownCtx) // render streams end via ctx
}
