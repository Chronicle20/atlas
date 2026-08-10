package consumer

import (
	"context"
	"errors"
	"sync"
	"testing"

	"github.com/sirupsen/logrus/hooks/test"

	routine "github.com/Chronicle20/atlas/libs/atlas-routine"
)

// recordingCommitter captures every offset handed to advance's commit func
// and can be armed to fail a fixed number of times.
type recordingCommitter struct {
	mu       sync.Mutex
	offsets  []int64
	failNext int
}

func (r *recordingCommitter) commit(offset int64) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.failNext > 0 {
		r.failNext--
		return errors.New("commit failed")
	}
	r.offsets = append(r.offsets, offset)
	return nil
}

func (r *recordingCommitter) seen() []int64 {
	r.mu.Lock()
	defer r.mu.Unlock()
	out := make([]int64, len(r.offsets))
	copy(out, r.offsets)
	return out
}

// TestCursorCommitsOffsetPlusOne pins the exact committed value. *kafka.Reader
// commits msg.Offset+1 (kafka-go reader.go:1529); both engines must write the
// identical number or a KAFKA_CONSUMER_ENGINE rollback replays or skips.
func TestCursorCommitsOffsetPlusOne(t *testing.T) {
	cur := newCursor()
	rc := &recordingCommitter{}

	in := cur.track(41)
	in.mark(true)
	if err := cur.advance(rc.commit); err != nil {
		t.Fatalf("advance: %v", err)
	}

	got := rc.seen()
	if len(got) != 1 || got[0] != 42 {
		t.Fatalf("committed %v, want [42]", got)
	}
	if cur.committedOffset() != 42 {
		t.Fatalf("committedOffset = %d, want 42", cur.committedOffset())
	}
}

// TestCursorCommitsOnlyContiguousPrefix: message 1 completes before 0, so
// nothing may commit until 0 completes; then a single commit covers both.
func TestCursorCommitsOnlyContiguousPrefix(t *testing.T) {
	cur := newCursor()
	rc := &recordingCommitter{}

	first := cur.track(0)
	second := cur.track(1)

	second.mark(true)
	if err := cur.advance(rc.commit); err != nil {
		t.Fatalf("advance: %v", err)
	}
	if got := rc.seen(); len(got) != 0 {
		t.Fatalf("committed %v before the head completed, want none", got)
	}

	first.mark(true)
	if err := cur.advance(rc.commit); err != nil {
		t.Fatalf("advance: %v", err)
	}
	if got := rc.seen(); len(got) != 1 || got[0] != 2 {
		t.Fatalf("committed %v, want [2]", got)
	}
}

// TestCursorFailedMessageBlocksCursor: a handler failure at the head must
// stop the cursor permanently, so the failed message is redelivered.
func TestCursorFailedMessageBlocksCursor(t *testing.T) {
	cur := newCursor()
	rc := &recordingCommitter{}

	head := cur.track(10)
	tail := cur.track(11)
	head.mark(false)
	tail.mark(true)

	if err := cur.advance(rc.commit); err != nil {
		t.Fatalf("advance: %v", err)
	}
	if got := rc.seen(); len(got) != 0 {
		t.Fatalf("committed %v past a failed message, want none", got)
	}
	if cur.committedOffset() != -1 {
		t.Fatalf("committedOffset = %d, want -1", cur.committedOffset())
	}
}

// TestCursorCommitFailureDoesNotAdvance is FR-1.3: a failed commit leaves the
// high-water mark in place and the next advance retries the same value.
func TestCursorCommitFailureDoesNotAdvance(t *testing.T) {
	cur := newCursor()
	rc := &recordingCommitter{failNext: 1}

	in := cur.track(7)
	in.mark(true)

	if err := cur.advance(rc.commit); err == nil {
		t.Fatal("advance returned nil on a failing commit, want the error")
	}
	if cur.committedOffset() != -1 {
		t.Fatalf("committedOffset = %d after a failed commit, want -1", cur.committedOffset())
	}

	if err := cur.advance(rc.commit); err != nil {
		t.Fatalf("retry advance: %v", err)
	}
	if got := rc.seen(); len(got) != 1 || got[0] != 8 {
		t.Fatalf("retry committed %v, want [8]", got)
	}
}

// TestCursorAdvanceIsIdempotent: advancing with nothing new must not re-issue
// a commit for an offset already committed.
func TestCursorAdvanceIsIdempotent(t *testing.T) {
	cur := newCursor()
	rc := &recordingCommitter{}

	in := cur.track(3)
	in.mark(true)
	for i := 0; i < 3; i++ {
		if err := cur.advance(rc.commit); err != nil {
			t.Fatalf("advance %d: %v", i, err)
		}
	}
	if got := rc.seen(); len(got) != 1 {
		t.Fatalf("committed %v, want exactly one commit", got)
	}
}

// TestCursorResumeOffset: after a partition-reader rebuild the loop must
// resume at the first uncommitted message, or at the committed high-water
// mark, or at the assignment's fallback offset — in that order.
func TestCursorResumeOffset(t *testing.T) {
	cur := newCursor()
	if got := cur.resumeOffset(-2); got != -2 {
		t.Fatalf("empty cursor resumeOffset = %d, want the fallback -2", got)
	}

	rc := &recordingCommitter{}
	done := cur.track(5)
	done.mark(true)
	if err := cur.advance(rc.commit); err != nil {
		t.Fatalf("advance: %v", err)
	}
	if got := cur.resumeOffset(-2); got != 6 {
		t.Fatalf("resumeOffset = %d after committing 6, want 6", got)
	}

	cur.track(6)
	if got := cur.resumeOffset(-2); got != 6 {
		t.Fatalf("resumeOffset = %d with offset 6 in flight, want 6", got)
	}
}

// TestCursorResetKeepsCommitted: reset discards pending entries (they will be
// re-fetched by the rebuilt reader) but must never rewind the committed mark.
func TestCursorResetKeepsCommitted(t *testing.T) {
	cur := newCursor()
	rc := &recordingCommitter{}
	in := cur.track(0)
	in.mark(true)
	if err := cur.advance(rc.commit); err != nil {
		t.Fatalf("advance: %v", err)
	}
	cur.track(1)
	cur.reset()

	if cur.len() != 0 {
		t.Fatalf("len = %d after reset, want 0", cur.len())
	}
	if cur.committedOffset() != 1 {
		t.Fatalf("committedOffset = %d after reset, want 1", cur.committedOffset())
	}
}

// TestCursorConcurrentAdvance: many handlers completing at once must not
// commit out of order. Under -race this also proves the cursor is safe for
// concurrent use from maxInFlight handler goroutines.
func TestCursorConcurrentAdvance(t *testing.T) {
	cur := newCursor()
	rc := &recordingCommitter{}

	const n = 50
	ins := make([]*inflight, n)
	for i := 0; i < n; i++ {
		ins[i] = cur.track(int64(i))
	}

	var wg sync.WaitGroup
	l, _ := test.NewNullLogger()
	for i := 0; i < n; i++ {
		wg.Add(1)
		idx := i
		routine.Go(l, context.Background(), func(_ context.Context) {
			defer wg.Done()
			ins[idx].mark(true)
			_ = cur.advance(rc.commit)
		})
	}
	wg.Wait()
	_ = cur.advance(rc.commit)

	got := rc.seen()
	if len(got) == 0 {
		t.Fatal("no commits recorded")
	}
	for i := 1; i < len(got); i++ {
		if got[i] <= got[i-1] {
			t.Fatalf("commits went backwards: %v", got)
		}
	}
	if got[len(got)-1] != int64(n) {
		t.Fatalf("final commit %d, want %d", got[len(got)-1], n)
	}
}
