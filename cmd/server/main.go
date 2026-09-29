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

	"famslide/internal/config"
	"famslide/internal/icloud"
	"famslide/internal/media"
	"famslide/internal/storage"
	"famslide/internal/syncer"
	"famslide/web"
)

func main() {
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, nil)))
	cfg, err := config.Load()
	if err != nil {
		slog.Error("configuration failed", "error", err)
		os.Exit(1)
	}
	source, err := icloud.NewSource(cfg.AlbumURL)
	if err != nil {
		slog.Error("album configuration failed", "error", err)
		os.Exit(1)
	}
	store, err := storage.Open(cfg.DataDir)
	if err != nil {
		slog.Error("cache open failed", "error", err)
		os.Exit(1)
	}
	if err := media.EnsureDynamicManifest(store); err != nil {
		slog.Error("scene migration failed", "error", err)
		os.Exit(1)
	}
	syncWorker := syncer.New(source, store, cfg.Prune)
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()
	go func() {
		syncWorker.Run(ctx)
		ticker := time.NewTicker(cfg.SyncInterval)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				syncWorker.Run(ctx)
			case <-ctx.Done():
				return
			}
		}
	}()
	server := &http.Server{Addr: ":" + cfg.Port, Handler: (&web.Server{Store: store, Sync: syncWorker, ManualSync: cfg.ManualSync}).Handler(), ReadHeaderTimeout: 10 * time.Second}
	go func() {
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		server.Shutdown(shutdownCtx)
	}()
	slog.Info("server listening", "port", cfg.Port)
	if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		slog.Error("server failed", "error", err)
		os.Exit(1)
	}
}
