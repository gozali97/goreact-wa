package webhook

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"log"
	"net/http"
	"time"

	"wa-proxy/internal/config"
)

// Event names emitted to webhooks (WAHA-inspired).
const (
	EventMessage       = "message"        // incoming message
	EventMessageAck    = "message.ack"    // delivery/read receipt
	EventSessionStatus = "session.status" // device connection status changed
)

// Payload is the envelope POSTed to the configured webhook URL.
type Payload struct {
	Event     string      `json:"event"`
	Timestamp int64       `json:"timestamp"`
	Session   string      `json:"session"`
	Data      interface{} `json:"data"`
}

// Dispatcher delivers events to an external webhook with HMAC signing and a
// bounded retry queue. It is a no-op when no webhook URL is configured.
type Dispatcher struct {
	cfg     *config.Config
	client  *http.Client
	queue   chan Payload
	enabled bool
	events  map[string]bool // nil/empty = all events
}

// New creates a Dispatcher. Call Start to begin processing.
func New(cfg *config.Config) *Dispatcher {
	d := &Dispatcher{
		cfg:     cfg,
		client:  &http.Client{Timeout: 15 * time.Second},
		queue:   make(chan Payload, 512),
		enabled: cfg.WebhookEnabled(),
	}
	if list := cfg.WebhookEventList(); len(list) > 0 {
		d.events = make(map[string]bool, len(list))
		for _, e := range list {
			d.events[e] = true
		}
	}
	return d
}

// Start launches the delivery worker.
func (d *Dispatcher) Start() {
	if !d.enabled {
		return
	}
	go d.run()
	log.Printf("webhook: dispatching to %s", d.cfg.WebhookURL)
}

// Emit queues an event for delivery (non-blocking; dropped if the buffer is
// full or webhooks are disabled / filtered out).
func (d *Dispatcher) Emit(event string, data interface{}) {
	if !d.enabled {
		return
	}
	if d.events != nil && !d.events[event] {
		return
	}
	p := Payload{
		Event:     event,
		Timestamp: time.Now().Unix(),
		Session:   "default",
		Data:      data,
	}
	select {
	case d.queue <- p:
	default:
		log.Printf("webhook: queue full, dropping %s event", event)
	}
}

func (d *Dispatcher) run() {
	for p := range d.queue {
		d.deliver(p)
	}
}

func (d *Dispatcher) deliver(p Payload) {
	body, err := json.Marshal(p)
	if err != nil {
		log.Printf("webhook: marshal %s: %v", p.Event, err)
		return
	}

	var signature string
	if d.cfg.WebhookSecret != "" {
		mac := hmac.New(sha256.New, []byte(d.cfg.WebhookSecret))
		mac.Write(body)
		signature = hex.EncodeToString(mac.Sum(nil))
	}

	maxAttempts := d.cfg.WebhookMaxRetry + 1
	if maxAttempts < 1 {
		maxAttempts = 1
	}

	for attempt := 1; attempt <= maxAttempts; attempt++ {
		ok := d.post(body, signature)
		if ok {
			return
		}
		if attempt < maxAttempts {
			time.Sleep(time.Duration(attempt) * 2 * time.Second)
		}
	}
	log.Printf("webhook: gave up delivering %s after %d attempts", p.Event, maxAttempts)
}

func (d *Dispatcher) post(body []byte, signature string) bool {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, d.cfg.WebhookURL, bytes.NewReader(body))
	if err != nil {
		return false
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "wa-proxy-webhook/1.0")
	if signature != "" {
		req.Header.Set("X-Webhook-Signature", "sha256="+signature)
	}

	resp, err := d.client.Do(req)
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	return resp.StatusCode >= 200 && resp.StatusCode < 300
}
