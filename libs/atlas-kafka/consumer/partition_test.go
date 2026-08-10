package consumer

import (
	"context"
	"errors"
	"io"
	"sync"
	"testing"
	"time"

	"github.com/segmentio/kafka-go"

	routine "github.com/Chronicle20/atlas/libs/atlas-routine"
)

// blockingReader serves a scripted sequence of messages, then blocks until
// its context is done. Each entry may instead be an error.
type scriptedPartitionReader struct {
	mu     sync.Mutex
	msgs   []kafka.Message
	errs   []error
	idx    int
	closed int
}

func (r *scriptedPartitionReader) FetchMessage(ctx context.Context) (kafka.Message, error) {
	r.mu.Lock()
	i := r.idx
	r.idx++
	var (
		msg kafka.Message
		err error
	)
	if i < len(r.msgs) {
		msg = r.msgs[i]
	}
	if i < len(r.errs) {
		err = r.errs[i]
	}
	past := i >= len(r.msgs) && i >= len(r.errs)
	r.mu.Unlock()

	if past {
		<-ctx.Done()
		return kafka.Message{}, ctx.Err()
	}
	if err != nil {
		return kafka.Message{}, err
	}
	return msg, nil
}

func (r *scriptedPartitionReader) CommitMessages(context.Context, ...kafka.Message) error {
	return errors.New("partition readers must never commit through the reader")
}

func (r *scriptedPartitionReader) Close() error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.closed++
	return nil
}

func (r *scriptedPartitionReader) closeCount() int {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.closed
}

// TestRunPartitionCommitsOffsetPlusOneThroughGeneration is the end-to-end
// form of the offset contract: a delivered message must reach
// Generation.CommitOffsets as msg.Offset+1, keyed by topic and partition.
func TestRunPartitionCommitsOffsetPlusOneThroughGeneration(t *testing.T) {
	l, _ := silentLogger()
	rd := &scriptedPartitionReader{msgs: []kafka.Message{{Offset: 41, Value: []byte("x")}}}

	c := newTestConsumer()
	c.topic = "EVENT_TOPIC_TEST"
	c.fetchTimeout = 50 * time.Millisecond
	c.maxInFlight = 1
	c.prp = func(kafka.ReaderConfig, int64) KafkaReader { return rd }

	gen := newFakeGeneration(1, map[string][]kafka.PartitionAssignment{
		"EVENT_TOPIC_TEST": {{ID: 0, Offset: kafka.FirstOffset}},
	})

	ctx, cancel := context.WithCancel(context.Background())
	gctx, gcancel := context.WithCancel(context.Background())
	defer gcancel()

	tl, _ := silentLogger()
	done := make(chan struct{})
	routine.Go(tl, context.Background(), func(_ context.Context) {
		c.runPartition(l, ctx, gctx, gen, kafka.PartitionAssignment{ID: 0, Offset: kafka.FirstOffset})
		close(done)
	})

	waitFor(t, func() bool { return len(gen.commits()) > 0 }, "no commit observed")
	cancel()
	<-done

	got := gen.commits()
	if got[0].Offset != 42 {
		t.Fatalf("committed offset %d, want 42 (msg.Offset+1)", got[0].Offset)
	}
	if got[0].Topic != "EVENT_TOPIC_TEST" || got[0].Partition != 0 {
		t.Fatalf("committed %+v, want topic EVENT_TOPIC_TEST partition 0", got[0])
	}
}

// TestRunPartitionRebuildsReaderWithoutEndingGeneration is FR-2.3 plus the
// design §1.1 inversion: a reader error rebuilds THAT reader in place; the
// generation must survive, because ending it would rejoin the group and
// rebalance every topic in it.
func TestRunPartitionRebuildsReaderWithoutEndingGeneration(t *testing.T) {
	l, _ := silentLogger()

	var built int
	var mu sync.Mutex
	c := newTestConsumer()
	c.topic = "t"
	c.fetchTimeout = 50 * time.Millisecond
	c.maxInFlight = 1
	c.prp = func(kafka.ReaderConfig, int64) KafkaReader {
		mu.Lock()
		built++
		n := built
		mu.Unlock()
		if n == 1 {
			return &scriptedPartitionReader{errs: []error{io.EOF}}
		}
		return &scriptedPartitionReader{}
	}

	gen := newFakeGeneration(1, map[string][]kafka.PartitionAssignment{
		"t": {{ID: 0, Offset: kafka.FirstOffset}},
	})

	ctx, cancel := context.WithCancel(context.Background())
	gctx, gcancel := context.WithCancel(context.Background())
	defer gcancel()

	tl, _ := silentLogger()
	done := make(chan struct{})
	routine.Go(tl, context.Background(), func(_ context.Context) {
		c.runPartition(l, ctx, gctx, gen, kafka.PartitionAssignment{ID: 0, Offset: kafka.FirstOffset})
		close(done)
	})

	waitFor(t, func() bool {
		mu.Lock()
		defer mu.Unlock()
		return built >= 2
	}, "reader was not rebuilt after io.EOF")

	if gen.ended() {
		t.Fatal("generation ended on a partition-reader error; recovery must stay local")
	}
	if got := c.Snapshot().RecreateCount; got < 1 {
		t.Fatalf("RecreateCount = %d, want >= 1", got)
	}

	cancel()
	<-done
}

// TestRunPartitionWedgeRebuildsReader: a reader that makes no progress for
// maxConsecutiveTimeouts ticks is still recovered (FR-2.3) — the task-136
// behaviour, now scoped to one partition reader.
func TestRunPartitionWedgeRebuildsReader(t *testing.T) {
	l, hook := silentLogger()

	var built int
	var mu sync.Mutex
	c := newTestConsumer()
	c.topic = "t"
	c.fetchTimeout = 20 * time.Millisecond
	c.maxConsecutiveTimeouts = 2
	c.maxInFlight = 1
	c.prp = func(kafka.ReaderConfig, int64) KafkaReader {
		mu.Lock()
		built++
		mu.Unlock()
		return &scriptedPartitionReader{} // blocks → every tick is a deadline
	}

	gen := newFakeGeneration(1, map[string][]kafka.PartitionAssignment{
		"t": {{ID: 0, Offset: kafka.FirstOffset}},
	})

	ctx, cancel := context.WithCancel(context.Background())
	gctx, gcancel := context.WithCancel(context.Background())
	defer gcancel()

	tl, _ := silentLogger()
	done := make(chan struct{})
	routine.Go(tl, context.Background(), func(_ context.Context) {
		c.runPartition(l, ctx, gctx, gen, kafka.PartitionAssignment{ID: 0, Offset: kafka.FirstOffset})
		close(done)
	})

	waitFor(t, func() bool {
		mu.Lock()
		defer mu.Unlock()
		return built >= 2
	}, "wedged reader was not rebuilt")

	if !hasLogContaining(hook, "wedged") {
		t.Fatal("no wedge warning logged for an ASSIGNED partition; FR-2.3 requires it")
	}
	if gen.ended() {
		t.Fatal("generation ended on a wedge; recovery must stay local")
	}

	cancel()
	<-done
}

// TestRunPartitionDrainCommitsCompletedWork is FR-1.6: work finished before
// the generation ends must be committed, not left for redelivery.
func TestRunPartitionDrainCommitsCompletedWork(t *testing.T) {
	l, _ := silentLogger()
	rd := &scriptedPartitionReader{msgs: []kafka.Message{{Offset: 0}, {Offset: 1}}}

	c := newTestConsumer()
	c.topic = "t"
	c.fetchTimeout = 50 * time.Millisecond
	c.maxInFlight = 1
	c.prp = func(kafka.ReaderConfig, int64) KafkaReader { return rd }

	gen := newFakeGeneration(1, map[string][]kafka.PartitionAssignment{
		"t": {{ID: 0, Offset: kafka.FirstOffset}},
	})

	ctx := context.Background()
	gctx, gcancel := context.WithCancel(context.Background())

	tl, _ := silentLogger()
	done := make(chan struct{})
	routine.Go(tl, context.Background(), func(_ context.Context) {
		c.runPartition(l, ctx, gctx, gen, kafka.PartitionAssignment{ID: 0, Offset: kafka.FirstOffset})
		close(done)
	})

	waitFor(t, func() bool { return len(gen.commits()) >= 2 }, "messages were not committed")
	gcancel()
	<-done

	last := gen.commits()
	if last[len(last)-1].Offset != 2 {
		t.Fatalf("final committed offset %d, want 2", last[len(last)-1].Offset)
	}
	if rd.closeCount() == 0 {
		t.Fatal("partition reader was not closed on generation end")
	}
}

// TestRunPartitionCommitFailureDoesNotAdvance is the FR-1.3 wiring test: a
// failing CommitOffsets must be logged and retried, never silently skipped.
func TestRunPartitionCommitFailureDoesNotAdvance(t *testing.T) {
	l, hook := silentLogger()
	rd := &scriptedPartitionReader{msgs: []kafka.Message{{Offset: 0}}}

	c := newTestConsumer()
	c.topic = "t"
	c.fetchTimeout = 20 * time.Millisecond
	c.maxInFlight = 1
	c.prp = func(kafka.ReaderConfig, int64) KafkaReader { return rd }

	gen := newFakeGeneration(1, map[string][]kafka.PartitionAssignment{
		"t": {{ID: 0, Offset: kafka.FirstOffset}},
	})
	gen.setCommitErr(errors.New("broker unavailable"))

	ctx, cancel := context.WithCancel(context.Background())
	gctx, gcancel := context.WithCancel(context.Background())
	defer gcancel()

	tl, _ := silentLogger()
	done := make(chan struct{})
	routine.Go(tl, context.Background(), func(_ context.Context) {
		c.runPartition(l, ctx, gctx, gen, kafka.PartitionAssignment{ID: 0, Offset: kafka.FirstOffset})
		close(done)
	})

	waitFor(t, func() bool { return hasLogContaining(hook, "commit") }, "no commit-failure warning logged")
	if len(gen.commits()) != 0 {
		t.Fatalf("commits recorded despite a failing committer: %v", gen.commits())
	}

	gen.setCommitErr(nil)
	waitFor(t, func() bool { return len(gen.commits()) > 0 }, "commit was never retried")
	if got := gen.commits()[0].Offset; got != 1 {
		t.Fatalf("retried commit offset %d, want 1", got)
	}

	cancel()
	<-done
}

// TestQuiesceAbandonsAfterDrainTimeout exercises the drain-timeout-exceeded
// branch of quiesce directly: a handler that never finishes must not block
// quiesce past drainTimeout, and the abandonment must be logged (task-209
// review finding 1). drainTimeout is temporarily overridden to a small value
// so the test is fast; production code never reassigns it, so the default
// stays exactly 5s. The wait for "quiesce returned" and "drain-timeout
// warning logged" is channel/poll-based, not a sleep used as the pass/fail
// gate — the only sleep-shaped thing here is the drainTimeout override
// itself, which IS the condition under test.
func TestQuiesceAbandonsAfterDrainTimeout(t *testing.T) {
	l, hook := silentLogger()

	orig := drainTimeout
	drainTimeout = 30 * time.Millisecond
	defer func() { drainTimeout = orig }()

	c := newTestConsumer()
	c.topic = "t"

	var wg sync.WaitGroup
	stuck := make(chan struct{})
	wg.Add(1)
	tl, _ := silentLogger()
	routine.Go(tl, context.Background(), func(_ context.Context) {
		defer wg.Done()
		<-stuck // deliberately never closed until the test is done asserting
	})

	cur := newCursor()
	commit := func(int64) error { return nil }

	done := make(chan struct{})
	routine.Go(tl, context.Background(), func(_ context.Context) {
		c.quiesce(l, &wg, cur, commit)
		close(done)
	})

	select {
	case <-done:
	case <-time.After(3 * time.Second):
		t.Fatal("quiesce did not return within the drain timeout; the abandoned-handler path is not bounded")
	}

	if !hasLogContaining(hook, "Drain timeout") {
		t.Fatal("no drain-timeout warning logged for the abandoned handler")
	}

	// Release the stuck handler so its goroutine (and the quiesce waiter
	// still blocked on wg.Wait()) can exit before the test ends.
	close(stuck)
	wg.Wait()
}
