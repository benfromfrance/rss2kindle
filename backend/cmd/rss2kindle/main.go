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

	"github.com/benoit/rss2kindle/internal/api"
	"github.com/benoit/rss2kindle/internal/config"
	"github.com/benoit/rss2kindle/internal/delivery"
	"github.com/benoit/rss2kindle/internal/feed"
	"github.com/benoit/rss2kindle/internal/mailer"
	"github.com/benoit/rss2kindle/internal/refresher"
	"github.com/benoit/rss2kindle/internal/scheduler"
	"github.com/benoit/rss2kindle/internal/store"
	"github.com/benoit/rss2kindle/web"
)

func main() {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))
	slog.SetDefault(logger)

	slog.Info("démarrage de RSS to Kindle")

	cfg, err := config.Load()
	if err != nil {
		slog.Error("échec chargement configuration", "err", err)
		os.Exit(1)
	}

	st, err := store.Open(cfg.DataDir, cfg.Secret)
	if err != nil {
		slog.Error("échec ouverture base de données", "err", err)
		os.Exit(1)
	}
	defer st.Close()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	fetcher := feed.NewFetcher()
	refr := refresher.New(st, fetcher)
	mailService := mailer.New()
	pipeline := delivery.NewPipeline(st, mailService, cfg.DataDir)
	sched := scheduler.New(st, pipeline)

	// Lancement des services en arrière-plan
	pipeline.Start(ctx)
	sched.Start(ctx)
	go refr.Run(ctx)

	// Serveur HTTP
	server := api.NewServer(cfg, st, fetcher, refr, pipeline, sched, mailService, web.Dist())

	httpServer := &http.Server{
		Addr:         cfg.Addr,
		Handler:      server.Handler(),
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 60 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	go func() {
		slog.Info("serveur HTTP à l'écoute", "addr", cfg.Addr)
		if err := httpServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			slog.Error("erreur serveur HTTP", "err", err)
			os.Exit(1)
		}
	}()

	// Arrêt propre
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	slog.Info("arrêt de RSS to Kindle en cours...")
	cancel()

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()

	if err := httpServer.Shutdown(shutdownCtx); err != nil {
		slog.Error("erreur arrêt serveur HTTP", "err", err)
	}
	slog.Info("arrêt terminé")
}
