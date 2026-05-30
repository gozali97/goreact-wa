// Package main is the wa-proxy single-binary entrypoint: it serves the REST
// API, WebSocket, embedded React dashboard, and runs the WhatsApp client.
package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"wa-proxy/internal/api"
	"wa-proxy/internal/apikey"
	"wa-proxy/internal/config"
	"wa-proxy/internal/database"
	"wa-proxy/internal/logsvc"
	"wa-proxy/internal/repository"
	"wa-proxy/internal/storage"
	"wa-proxy/internal/webhook"
	"wa-proxy/internal/whatsapp"
	"wa-proxy/internal/worker"
	"wa-proxy/internal/ws"
)

func main() {
	// Load .env (no-op in production where real env vars are set).
	config.LoadDotEnv(".env")
	cfg := config.Load()

	// Structured log service (daily JSON-lines files under LOG_PATH).
	if err := logsvc.Init(cfg.LogPath, !cfg.IsProduction()); err != nil {
		log.Fatalf("logsvc: %v", err)
	}
	logsvc.Info("app.start", "wa-proxy starting", map[string]interface{}{"env": cfg.AppEnv})

	// Database.
	if err := database.EnsureDatabase(cfg); err != nil {
		log.Fatalf("ensure database: %v", err)
	}
	db, err := database.Connect(cfg)
	if err != nil {
		log.Fatalf("database: %v", err)
	}
	if err := database.Migrate(db); err != nil {
		log.Fatalf("migrate: %v", err)
	}
	repo := repository.New(db)

	// API key service (DB-backed; generated on connect, cleared on logout).
	keys := apikey.New(repo)

	// Storage.
	store, err := storage.New(cfg.StoragePath)
	if err != nil {
		log.Fatalf("storage: %v", err)
	}

	// WebSocket hub.
	hub := ws.NewHub()
	go hub.Run()

	// Outbound webhook dispatcher (WAHA-style; no-op if WEBHOOK_URL unset).
	wh := webhook.New(cfg)
	wh.Start()

	// WhatsApp manager.
	wa, err := whatsapp.NewManager(cfg, repo, hub, store, wh, keys)
	if err != nil {
		log.Fatalf("whatsapp: %v", err)
	}

	// Workers.
	sender := worker.NewMessageSender(wa)
	sender.Start()
	broadcaster := worker.NewBroadcastWorker(cfg, repo, wa, hub)
	broadcaster.Start()
	sessionWorker := worker.NewSessionWorker(wa)
	sessionWorker.Start()
	cleanupWorker := worker.NewCleanupWorker(store, cfg.MediaRetentionDays)
	cleanupWorker.Start()

	// Connect existing WhatsApp session (if any) in the background.
	go wa.Init()

	// HTTP server.
	handler := api.NewHandler(cfg, repo, wa, hub, store, sender, broadcaster, keys)
	router := api.NewRouter(cfg, handler, hub)

	srv := &http.Server{
		Addr:    cfg.Addr(),
		Handler: router,
	}

	go func() {
		log.Printf("wa-proxy listening on http://%s (env=%s)", cfg.Addr(), cfg.AppEnv)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("http server: %v", err)
		}
	}()

	// Graceful shutdown.
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("shutting down...")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		log.Printf("server shutdown: %v", err)
	}
	sessionWorker.Stop()
	cleanupWorker.Stop()
	log.Println("bye")
}
