package consumer

import (
	"sync"
	"sync/atomic"
)

// noCommit marks "no offset has been committed yet". Real committed values
// are msg.Offset+1 and therefore always >= 1, so -1 is unambiguous.
const noCommit int64 = -1

// inflight is one fetched message tracked by a cursor. offset is the raw
// kafka.Message.Offset — the +1 conversion happens once, in advance.
type inflight struct {
	offset int64
	done   atomic.Bool
	ok     atomic.Bool
}

// mark records the outcome of the message's handlers. It must be called
// exactly once per inflight.
func (in *inflight) mark(ok bool) {
	in.ok.Store(ok)
	in.done.Store(true)
}

// cursor is the per-partition prefix-commit cursor. It commits only the
// highest CONTIGUOUSLY-completed offset, so a failed message blocks every
// commit behind it and that message is redelivered (at-least-once).
//
// OFFSET CONTRACT (task-209 design §5): the value handed to the commit func
// is msg.Offset+1 — identical to what *kafka.Reader writes
// (kafka-go reader.go:1529). Both consumer engines therefore write the same
// number for the same delivered message, which is what makes flipping
// KAFKA_CONSUMER_ENGINE a restart rather than a migration.
//
// mu guards the queue and the offset marks. cmu serializes the commit call
// itself so two concurrent advances cannot write offsets out of order; it is
// never held together with mu across a network call.
type cursor struct {
	mu            sync.Mutex
	cmu           sync.Mutex
	pending       []*inflight
	pendingCommit int64 // highest contiguous msg.Offset+1 ready to commit
	committed     int64 // highest msg.Offset+1 successfully committed
}

func newCursor() *cursor {
	return &cursor{pendingCommit: noCommit, committed: noCommit}
}

// track registers a freshly fetched message. Messages must be tracked in
// fetch order — the prefix walk depends on it.
func (c *cursor) track(offset int64) *inflight {
	in := &inflight{offset: offset}
	c.mu.Lock()
	c.pending = append(c.pending, in)
	c.mu.Unlock()
	return in
}

// len reports how many messages are still queued. The parallel fetch loop
// uses it for back-pressure.
func (c *cursor) len() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return len(c.pending)
}

// advance walks the completed-and-successful prefix, drops it from the queue,
// and commits the resulting high-water mark. On a commit error the mark is
// RETAINED (FR-1.3) — committed does not move, so the next advance retries
// the same value. Returns the commit error, if any.
func (c *cursor) advance(commit func(offset int64) error) error {
	c.mu.Lock()
	i := 0
	for i < len(c.pending) && c.pending[i].done.Load() && c.pending[i].ok.Load() {
		i++
	}
	if i > 0 {
		c.pendingCommit = c.pending[i-1].offset + 1
		c.pending = c.pending[i:]
	}
	c.mu.Unlock()

	c.cmu.Lock()
	defer c.cmu.Unlock()

	c.mu.Lock()
	target := c.pendingCommit
	already := c.committed
	c.mu.Unlock()
	if target == noCommit || target <= already {
		return nil
	}

	if err := commit(target); err != nil {
		return err
	}

	c.mu.Lock()
	if target > c.committed {
		c.committed = target
	}
	c.mu.Unlock()
	return nil
}

// committedOffset returns the last successfully committed value, or noCommit.
func (c *cursor) committedOffset() int64 {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.committed
}

// resumeOffset returns the offset a rebuilt partition reader must start from:
// the first still-uncommitted message if one is queued, else the committed
// high-water mark, else fallback (the generation's PartitionAssignment.Offset,
// which may be the kafka.FirstOffset/LastOffset sentinel).
func (c *cursor) resumeOffset(fallback int64) int64 {
	c.mu.Lock()
	defer c.mu.Unlock()
	if len(c.pending) > 0 {
		return c.pending[0].offset
	}
	if c.committed != noCommit {
		return c.committed
	}
	return fallback
}

// reset discards queued entries after the caller has quiesced in-flight
// handlers and computed a resume offset. The committed mark is never rewound.
func (c *cursor) reset() {
	c.mu.Lock()
	c.pending = nil
	c.pendingCommit = noCommit
	c.mu.Unlock()
}
