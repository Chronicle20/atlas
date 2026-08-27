package system_message

import (
	"sync"
	"time"

	"github.com/google/uuid"
)

type throttleKey struct {
	tenantId    uuid.UUID
	characterId uint32
}

// Throttle bounds how often a given (tenant, character) pair may be sent a
// hint. State is per-process: atlas-monster-death runs multiple replicas, so
// the effective bound is one hint per replica per window (D10). That is an
// approximation of Cosmic's per-character nextWarningTime, chosen over shared
// state in Redis because a cosmetic notice does not justify a new
// infrastructure dependency.
type Throttle struct {
	mu       sync.Mutex
	window   time.Duration
	capacity int
	now      func() time.Time
	last     map[throttleKey]time.Time
}

// NewThrottle constructs a Throttle with the given window, capacity, and
// clock source.
func NewThrottle(window time.Duration, capacity int, now func() time.Time) *Throttle {
	return &Throttle{
		window:   window,
		capacity: capacity,
		now:      now,
		last:     make(map[throttleKey]time.Time),
	}
}

// Allow reports whether a hint may be emitted now, recording the emission when
// it returns true.
func (t *Throttle) Allow(tenantId uuid.UUID, characterId uint32) bool {
	t.mu.Lock()
	defer t.mu.Unlock()

	k := throttleKey{tenantId: tenantId, characterId: characterId}
	n := t.now()

	if prior, ok := t.last[k]; ok && n.Sub(prior) < t.window {
		return false
	}

	if len(t.last) >= t.capacity {
		cutoff := n.Add(-t.window)
		for key, ts := range t.last {
			if ts.Before(cutoff) {
				delete(t.last, key)
			}
		}
	}

	t.last[k] = n
	return true
}

var (
	hintThrottle     *Throttle
	hintThrottleOnce sync.Once
)

// GetHintThrottle returns the process-wide hint throttle: a one-minute window,
// a 4096-key cap, and the real clock.
func GetHintThrottle() *Throttle {
	hintThrottleOnce.Do(func() {
		hintThrottle = NewThrottle(time.Minute, 4096, time.Now)
	})
	return hintThrottle
}
