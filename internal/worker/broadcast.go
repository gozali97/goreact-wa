package worker

import (
	"log"
	"math/rand"
	"time"

	"wa-proxy/internal/config"
	"wa-proxy/internal/logsvc"
	"wa-proxy/internal/model"
	"wa-proxy/internal/repository"
	"wa-proxy/internal/whatsapp"
	"wa-proxy/internal/ws"
)

// BroadcastWorker processes broadcast campaigns: queued sending with random
// delay between recipients and automatic retry of failures.
type BroadcastWorker struct {
	cfg     *config.Config
	repo    *repository.Repository
	wa      *whatsapp.Manager
	hub     *ws.Hub
	trigger chan uint
}

// NewBroadcastWorker creates a BroadcastWorker.
func NewBroadcastWorker(cfg *config.Config, repo *repository.Repository, wa *whatsapp.Manager, hub *ws.Hub) *BroadcastWorker {
	return &BroadcastWorker{
		cfg:     cfg,
		repo:    repo,
		wa:      wa,
		hub:     hub,
		trigger: make(chan uint, 64),
	}
}

// Start launches the worker loop and resumes any unfinished broadcasts.
func (w *BroadcastWorker) Start() {
	go w.run()
	go w.resumePending()
}

// Trigger asks the worker to process a specific broadcast id.
func (w *BroadcastWorker) Trigger(broadcastID uint) {
	w.trigger <- broadcastID
}

func (w *BroadcastWorker) resumePending() {
	// Give the WhatsApp client a moment to connect on startup.
	time.Sleep(5 * time.Second)
	pending, err := w.repo.PendingBroadcasts()
	if err != nil {
		log.Printf("broadcast: resume query: %v", err)
		return
	}
	for _, b := range pending {
		w.trigger <- b.ID
	}
}

func (w *BroadcastWorker) run() {
	for id := range w.trigger {
		w.process(id)
	}
}

func (w *BroadcastWorker) process(broadcastID uint) {
	b, err := w.repo.GetBroadcast(broadcastID)
	if err != nil || b == nil {
		log.Printf("broadcast: load %d: %v", broadcastID, err)
		return
	}

	// Don't burn through retries while the device is offline; requeue shortly.
	if !w.wa.IsConnected() {
		log.Printf("broadcast %d: whatsapp not connected, deferring", broadcastID)
		go func(id uint) {
			time.Sleep(10 * time.Second)
			w.trigger <- id
		}(broadcastID)
		return
	}

	b.Status = model.BroadcastSending
	_ = w.repo.UpdateBroadcast(b)
	w.emit(b)

	details, err := w.repo.PendingBroadcastDetails(broadcastID)
	if err != nil {
		log.Printf("broadcast: details %d: %v", broadcastID, err)
		return
	}

	for i := range details {
		d := &details[i]
		// Stop early if the device dropped mid-run; leave remaining pending so
		// the resume logic can finish later.
		if !w.wa.IsConnected() {
			log.Printf("broadcast %d: disconnected mid-run, pausing", broadcastID)
			break
		}
		w.sendOne(b, d)
		// Random delay between sends to look human and avoid bans.
		w.sleepRandom()
	}

	w.recount(b)
}

func (w *BroadcastWorker) sendOne(b *model.Broadcast, d *model.BroadcastDetail) {
	// Skip numbers not registered on WhatsApp — sending to them is a ban risk
	// and always fails. Mark as failed with a clear reason, no retries.
	if registered, err := w.wa.IsRegistered(d.Phone); err == nil && !registered {
		d.Status = model.BroadcastFailed
		d.ErrorMessage = "number not registered on WhatsApp"
		_ = w.repo.UpdateBroadcastDetail(d)
		return
	}

	maxRetry := w.cfg.BroadcastMaxRetry
	var lastErr error
	for attempt := 0; attempt <= maxRetry; attempt++ {
		_, err := w.wa.SendTextBulk(d.Phone, b.Message)
		if err == nil {
			d.Status = model.BroadcastSuccess
			d.ErrorMessage = ""
			_ = w.repo.UpdateBroadcastDetail(d)
			return
		}
		lastErr = err
		d.RetryCount++
		if attempt < maxRetry {
			time.Sleep(2 * time.Second)
		}
	}
	d.Status = model.BroadcastFailed
	if lastErr != nil {
		d.ErrorMessage = lastErr.Error()
	}
	_ = w.repo.UpdateBroadcastDetail(d)
	logsvc.Failed("broadcast.send", "broadcast send failed", map[string]interface{}{
		"broadcast_id": b.ID, "phone": d.Phone, "error": d.ErrorMessage,
	})
}

func (w *BroadcastWorker) recount(b *model.Broadcast) {
	details, err := w.repo.ListBroadcastDetails(b.ID)
	if err != nil {
		return
	}
	success, failed := 0, 0
	for _, d := range details {
		switch d.Status {
		case model.BroadcastSuccess:
			success++
		case model.BroadcastFailed:
			failed++
		}
	}
	b.TotalSuccess = success
	b.TotalFailed = failed
	b.TotalTarget = len(details)
	if failed == 0 {
		b.Status = model.BroadcastSuccess
	} else if success == 0 {
		b.Status = model.BroadcastFailed
	} else {
		b.Status = model.BroadcastSuccess // partial success
	}
	_ = w.repo.UpdateBroadcast(b)
	w.emit(b)
}

func (w *BroadcastWorker) sleepRandom() {
	min := w.cfg.BroadcastMinDelayMS
	max := w.cfg.BroadcastMaxDelayMS
	if max <= min {
		time.Sleep(time.Duration(min) * time.Millisecond)
		return
	}
	d := min + rand.Intn(max-min)
	time.Sleep(time.Duration(d) * time.Millisecond)
}

func (w *BroadcastWorker) emit(b *model.Broadcast) {
	w.hub.Emit(ws.EventBroadcast, b)
}
