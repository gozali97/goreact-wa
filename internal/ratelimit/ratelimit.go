// Package ratelimit provides a lightweight, in-memory send rate limiter used to
// pace outgoing WhatsApp messages and reduce the risk of the account being
// banned for spam-like behavior.
//
// Two independent limits are enforced:
//   - Per-recipient: the same phone number can only be messaged once per
//     PerRecipient interval (the primary "min 30s per chat" guard).
//   - Global: at most one send across all recipients per Global interval
//     (0 disables it; broadcast pacing already covers bulk sends).
package ratelimit

import (
	"sync"
	"time"
)

// Limiter enforces minimum intervals between sends.
type Limiter struct {
	mu sync.Mutex

	enabled     bool
	perRecip    time.Duration
	global      time.Duration
	lastByPhone map[string]time.Time
	lastGlobal  time.Time
}

// New creates a Limiter. If enabled is false, Acquire always succeeds.
func New(enabled bool, perRecipient, global time.Duration) *Limiter {
	return &Limiter{
		enabled:     enabled,
		perRecip:    perRecipient,
		global:      global,
		lastByPhone: make(map[string]time.Time),
	}
}

// Acquire attempts to reserve a send slot for the given phone. If allowed it
// records the send time and returns ok=true. Otherwise it returns ok=false and
// the duration the caller should wait before retrying.
func (l *Limiter) Acquire(phone string) (ok bool, retryAfter time.Duration) {
	if l == nil || !l.enabled {
		return true, 0
	}

	l.mu.Lock()
	defer l.mu.Unlock()

	now := time.Now()
	var wait time.Duration

	if l.perRecip > 0 {
		if last, seen := l.lastByPhone[phone]; seen {
			if d := l.perRecip - now.Sub(last); d > wait {
				wait = d
			}
		}
	}
	if l.global > 0 && !l.lastGlobal.IsZero() {
		if d := l.global - now.Sub(l.lastGlobal); d > wait {
			wait = d
		}
	}

	if wait > 0 {
		return false, wait
	}

	// Reserve the slot.
	l.lastByPhone[phone] = now
	l.lastGlobal = now
	return true, 0
}

// Wait blocks until a send slot for the phone is available, then reserves it.
// Intended for background workers (e.g. broadcast) that should pace rather than
// fail. Honors a cap to avoid pathological waits.
func (l *Limiter) Wait(phone string, max time.Duration) {
	if l == nil || !l.enabled {
		return
	}
	for {
		ok, retry := l.Acquire(phone)
		if ok {
			return
		}
		if max > 0 && retry > max {
			retry = max
		}
		time.Sleep(retry)
	}
}

// Reset clears the recorded time for a phone (e.g. on logout). Optional.
func (l *Limiter) Reset() {
	if l == nil {
		return
	}
	l.mu.Lock()
	l.lastByPhone = make(map[string]time.Time)
	l.lastGlobal = time.Time{}
	l.mu.Unlock()
}
