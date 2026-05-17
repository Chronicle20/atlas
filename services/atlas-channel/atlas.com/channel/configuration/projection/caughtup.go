package projection

import (
	"context"
	"sync"
	"sync/atomic"
)

// CaughtUp gates atlas-channel's readiness on having consumed past the
// end-offset snapshot taken at boot for each subscribed topic. The flag
// is one-way: once caught up, it never reverts (even if the consumed
// offsets logically lag behind a later end-offset snapshot).
type CaughtUp struct {
	mu         sync.Mutex
	snapshots  map[string]map[int]int64 // topic → partition → boot end offset
	consumed   map[string]map[int]int64 // topic → partition → highest consumed
	caughtUp   atomic.Bool
	readyChans []chan struct{} // one-shot signalers for WaitCaughtUp
}

// NewCaughtUp constructs a gate. SetEndOffsets must be called at least
// once before the gate can transition.
func NewCaughtUp() *CaughtUp {
	return &CaughtUp{
		snapshots: make(map[string]map[int]int64),
		consumed:  make(map[string]map[int]int64),
	}
}

// SetEndOffsets records the topic's boot end-offset snapshot. An empty
// offsets map (topic has no data yet) counts as trivially caught-up for
// that topic.
func (c *CaughtUp) SetEndOffsets(topic string, offsets map[int]int64) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if offsets == nil {
		offsets = map[int]int64{}
	}
	c.snapshots[topic] = offsets
	if c.consumed[topic] == nil {
		c.consumed[topic] = make(map[int]int64)
	}
	c.evaluateLocked()
}

// Observe records that the subscriber has consumed up to (and including)
// offset on partition p of topic. Idempotent: lower offsets are ignored.
func (c *CaughtUp) Observe(topic string, partition int, offset int64) {
	c.mu.Lock()
	defer c.mu.Unlock()
	cur, ok := c.consumed[topic]
	if !ok {
		cur = make(map[int]int64)
		c.consumed[topic] = cur
	}
	if existing, present := cur[partition]; present && existing >= offset {
		return
	}
	cur[partition] = offset
	c.evaluateLocked()
}

// CaughtUpNow is the cheap check the subscriber loop can call between
// every message.
func (c *CaughtUp) CaughtUpNow() bool { return c.caughtUp.Load() }

// WaitCaughtUp blocks until the gate flips or ctx is canceled.
func (c *CaughtUp) WaitCaughtUp(ctx context.Context) error {
	if c.caughtUp.Load() {
		return nil
	}
	c.mu.Lock()
	if c.caughtUp.Load() {
		c.mu.Unlock()
		return nil
	}
	ch := make(chan struct{})
	c.readyChans = append(c.readyChans, ch)
	c.mu.Unlock()
	select {
	case <-ch:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

// ReadyChecker returns a func suitable for a /health/ready endpoint.
func (c *CaughtUp) ReadyChecker() func() bool { return c.CaughtUpNow }

func (c *CaughtUp) evaluateLocked() {
	if len(c.snapshots) == 0 {
		// No topics registered yet — not caught up.
		return
	}
	for topic, ends := range c.snapshots {
		got := c.consumed[topic]
		for p, end := range ends {
			// "caught up" means we've consumed past end-1 (offsets are
			// 0-indexed; end is the high-water mark).
			if got[p] < end-1 {
				return
			}
		}
	}
	if !c.caughtUp.Load() {
		c.caughtUp.Store(true)
		for _, ch := range c.readyChans {
			close(ch)
		}
		c.readyChans = nil
	}
}
