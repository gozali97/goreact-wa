package worker

import (
	"time"

	"wa-proxy/internal/model"
	"wa-proxy/internal/whatsapp"
)

// SessionWorker periodically checks the WhatsApp connection and triggers a
// reconnect when the session is paired but the socket dropped. whatsmeow's own
// EnableAutoReconnect handles most cases; this is a safety-net monitor.
type SessionWorker struct {
	wa   *whatsapp.Manager
	stop chan struct{}
}

// NewSessionWorker creates a SessionWorker.
func NewSessionWorker(wa *whatsapp.Manager) *SessionWorker {
	return &SessionWorker{wa: wa, stop: make(chan struct{})}
}

// Start launches the monitor loop.
func (s *SessionWorker) Start() {
	go s.run()
}

// Stop halts the monitor loop.
func (s *SessionWorker) Stop() {
	close(s.stop)
}

func (s *SessionWorker) run() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-s.stop:
			return
		case <-ticker.C:
			s.check()
		}
	}
}

func (s *SessionWorker) check() {
	status := s.wa.Status()
	// If we believe we should be connected but the client reports otherwise,
	// ask the manager to reconnect.
	if status == model.DeviceConnecting || status == model.DeviceDisconnected {
		if s.wa.HasSession() && !s.wa.IsConnected() {
			_ = s.wa.Reconnect()
		}
	}
}
