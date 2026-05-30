package worker

import (
	"time"

	"wa-proxy/internal/logsvc"
	"wa-proxy/internal/storage"
)

// CleanupWorker periodically deletes stored media older than the configured
// retention window, guarding against unbounded disk growth.
type CleanupWorker struct {
	store         *storage.Store
	retentionDays int
	stop          chan struct{}
}

// NewCleanupWorker creates a CleanupWorker. If retentionDays <= 0, the worker
// is a no-op (auto-cleanup disabled).
func NewCleanupWorker(store *storage.Store, retentionDays int) *CleanupWorker {
	return &CleanupWorker{
		store:         store,
		retentionDays: retentionDays,
		stop:          make(chan struct{}),
	}
}

// Start launches the cleanup loop (runs at startup, then every 6 hours).
func (w *CleanupWorker) Start() {
	if w.retentionDays <= 0 {
		logsvc.Info("storage.cleanup", "auto-cleanup disabled (MEDIA_RETENTION_DAYS=0)", nil)
		return
	}
	go w.run()
}

// Stop halts the cleanup loop.
func (w *CleanupWorker) Stop() { close(w.stop) }

func (w *CleanupWorker) run() {
	// Initial sweep shortly after boot.
	time.Sleep(30 * time.Second)
	w.sweep()

	ticker := time.NewTicker(6 * time.Hour)
	defer ticker.Stop()
	for {
		select {
		case <-w.stop:
			return
		case <-ticker.C:
			w.sweep()
		}
	}
}

func (w *CleanupWorker) sweep() {
	cutoff := time.Now().AddDate(0, 0, -w.retentionDays)
	removed, freed, err := w.store.DeleteOlderThan(cutoff)
	if err != nil {
		logsvc.Errf("storage.cleanup", "auto-cleanup failed", err, nil)
		return
	}
	if removed > 0 {
		logsvc.Info("storage.cleanup", "auto-cleanup removed old media", map[string]interface{}{
			"removed": removed, "freed_bytes": freed, "retention_days": w.retentionDays,
		})
	}
}
