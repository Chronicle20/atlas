package consumer_test

import (
	"context"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/segmentio/kafka-go"
	"github.com/sirupsen/logrus"
	"github.com/sirupsen/logrus/hooks/test"
	"go.opentelemetry.io/otel"

	"github.com/Chronicle20/atlas/libs/atlas-kafka/consumer"
)

// statsStubReader blocks every FetchMessage on ctx (so each call expires the
// per-call deadline) and reports scripted ReaderStats deltas: entries in
// deltas are cycled per Stats() call; an empty slice means a permanent
// zero delta (no progress).
type statsStubReader struct {
	mu     sync.Mutex
	deltas []kafka.ReaderStats
	i      int
	closes int
}

func (r *statsStubReader) FetchMessage(ctx context.Context) (kafka.Message, error) {
	<-ctx.Done()
	return kafka.Message{}, ctx.Err()
}

func (r *statsStubReader) CommitMessages(_ context.Context, _ ...kafka.Message) error {
	return nil
}

func (r *statsStubReader) Close() error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.closes++
	return nil
}

func (r *statsStubReader) Closes() int {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.closes
}

func (r *statsStubReader) Stats() kafka.ReaderStats {
	r.mu.Lock()
	defer r.mu.Unlock()
	if len(r.deltas) == 0 {
		return kafka.ReaderStats{}
	}
	d := r.deltas[r.i%len(r.deltas)]
	r.i++
	return d
}

// Compile-time check: the stub must satisfy the new StatsProvider.
var _ consumer.StatsProvider = (*statsStubReader)(nil)

// TestIdleTickNeverWedges: a reader that keeps making fetch attempts
// (Fetches delta > 0) ticks through many deadlines without a warn log,
// without recreating, and with ConsecutiveTimeouts pinned at 0.
func TestIdleTickNeverWedges(t *testing.T) {
	consumer.ResetInstance()
	l, hook := test.NewNullLogger()
	l.SetLevel(logrus.DebugLevel)
	wg := &sync.WaitGroup{}
	otel.SetTracerProvider(&MockTracerProvider{})

	r1 := &statsStubReader{deltas: []kafka.ReaderStats{{Fetches: 3}}}
	rp := consumer.ConfigReaderProducer(readerFactory(t, r1))

	ctx, cancel := context.WithCancel(context.Background())
	defer wg.Wait()

	cm := consumer.GetManager(rp)
	c := consumer.NewConfig([]string{""}, "idle-consumer", "idle-topic", "test-group")
	cm.AddConsumer(l, ctx, wg)(
		c,
		// statsStubReader ignores readerConfig entirely (scripted Stats()
		// deltas, not real broker long-polling), so maxWait has no effect on
		// this test's exercised behavior — it only needs to stay below
		// fetchTimeout so registration doesn't trip the maxWait>=fetchTimeout
		// misconfiguration guard (which would otherwise fire here since the
		// library default maxWait is 10s).
		consumer.SetMaxWait(5*time.Millisecond),
		consumer.SetFetchTimeout(30*time.Millisecond),
		consumer.SetMaxConsecutiveTimeouts(3),
	)

	// >10 deadline ticks — far past the legacy wedge threshold of 3.
	time.Sleep(400 * time.Millisecond)

	s := snapshotForTopic(t, cm, "idle-topic")
	if s.RecreateCount != 0 {
		t.Fatalf("expected no recreates on an idle-but-healthy reader, got %d", s.RecreateCount)
	}
	if r1.Closes() != 0 {
		t.Fatalf("expected reader never closed while idle-healthy, got %d closes", r1.Closes())
	}
	if s.IdleTicks < 3 {
		t.Fatalf("expected IdleTicks >= 3, got %d", s.IdleTicks)
	}
	if s.LastIdleTickAt.IsZero() {
		t.Fatal("expected LastIdleTickAt to be set")
	}
	if s.ConsecutiveTimeouts != 0 {
		t.Fatalf("expected ConsecutiveTimeouts == 0 (idle ticks reset it), got %d", s.ConsecutiveTimeouts)
	}
	if s.NoProgressTicks != 0 {
		t.Fatalf("expected NoProgressTicks == 0, got %d", s.NoProgressTicks)
	}
	for _, e := range hook.AllEntries() {
		if e.Level == logrus.WarnLevel {
			t.Fatalf("expected no Warn logs for idle ticks, got %q", e.Message)
		}
	}

	cancel()
}

// TestNoProgressTicksEscalateToWedge: a reader whose Stats() delta shows
// zero progress across the threshold count is recreated, with the wedge
// warn naming topic and group.
func TestNoProgressTicksEscalateToWedge(t *testing.T) {
	consumer.ResetInstance()
	l, hook := test.NewNullLogger()
	wg := &sync.WaitGroup{}
	otel.SetTracerProvider(&MockTracerProvider{})

	r1 := &statsStubReader{} // permanent zero delta: stuck
	r2 := &statsStubReader{deltas: []kafka.ReaderStats{{Fetches: 1}}}
	rp := consumer.ConfigReaderProducer(readerFactory(t, r1, r2))

	ctx, cancel := context.WithCancel(context.Background())
	defer wg.Wait()

	cm := consumer.GetManager(rp)
	c := consumer.NewConfig([]string{""}, "stuck-consumer", "stuck-topic", "test-group")
	cm.AddConsumer(l, ctx, wg)(
		c,
		consumer.SetFetchTimeout(30*time.Millisecond),
		consumer.SetMaxConsecutiveTimeouts(3),
	)

	// Wait for the recreate to be OBSERVABLE, not merely for r1 to close.
	// r1.Close() fires the instant the wedge is detected, but recreateCount
	// only increments ~500ms later when the outer backoff elapses and r2 is
	// built (pre-existing onReaderCreated semantics). Polling on r1.Closes()
	// alone races that backoff window; poll on RecreateCount instead.
	deadline := time.Now().Add(5 * time.Second)
	for snapshotForTopic(t, cm, "stuck-topic").RecreateCount < 1 && time.Now().Before(deadline) {
		time.Sleep(10 * time.Millisecond)
	}
	if r1.Closes() == 0 {
		t.Fatal("stuck reader was never recreated")
	}

	s := snapshotForTopic(t, cm, "stuck-topic")
	if s.RecreateCount < 1 {
		t.Fatalf("expected RecreateCount >= 1 after a genuine stall, got %d", s.RecreateCount)
	}
	if s.NoProgressTicks < 3 {
		t.Fatalf("expected NoProgressTicks >= 3, got %d", s.NoProgressTicks)
	}
	if s.LastNoProgressAt.IsZero() {
		t.Fatal("expected LastNoProgressAt to be set")
	}

	foundWedgeWarn := false
	for _, e := range hook.AllEntries() {
		if e.Level == logrus.WarnLevel &&
			strings.Contains(e.Message, "FetchMessage wedged") &&
			strings.Contains(e.Message, "stuck-topic") &&
			strings.Contains(e.Message, "test-group") {
			foundWedgeWarn = true
			break
		}
	}
	if !foundWedgeWarn {
		t.Fatalf("expected a Warn log containing 'FetchMessage wedged' with topic+group, got: %v", hook.AllEntries())
	}

	cancel()
}

// TestIdleTickResetsNoProgressCount: alternating no-progress and progress
// ticks never reach the threshold — a transiently quiet reader that
// recovers is not recreated.
func TestIdleTickResetsNoProgressCount(t *testing.T) {
	consumer.ResetInstance()
	l, _ := test.NewNullLogger()
	wg := &sync.WaitGroup{}
	otel.SetTracerProvider(&MockTracerProvider{})

	r1 := &statsStubReader{deltas: []kafka.ReaderStats{{}, {Fetches: 1}}} // stuck, idle, stuck, idle...
	rp := consumer.ConfigReaderProducer(readerFactory(t, r1))

	ctx, cancel := context.WithCancel(context.Background())
	defer wg.Wait()

	cm := consumer.GetManager(rp)
	c := consumer.NewConfig([]string{""}, "flappy-consumer", "flappy-topic", "test-group")
	cm.AddConsumer(l, ctx, wg)(
		c,
		consumer.SetFetchTimeout(30*time.Millisecond),
		consumer.SetMaxConsecutiveTimeouts(2),
	)

	// >12 ticks alternating no-progress/progress: consecutive never hits 2.
	time.Sleep(400 * time.Millisecond)

	s := snapshotForTopic(t, cm, "flappy-topic")
	if s.RecreateCount != 0 {
		t.Fatalf("expected no recreate when progress interleaves, got %d", s.RecreateCount)
	}
	if r1.Closes() != 0 {
		t.Fatalf("expected reader never closed, got %d closes", r1.Closes())
	}
	if s.NoProgressTicks < 2 {
		t.Fatalf("expected cumulative NoProgressTicks >= 2, got %d", s.NoProgressTicks)
	}
	if s.IdleTicks < 2 {
		t.Fatalf("expected cumulative IdleTicks >= 2, got %d", s.IdleTicks)
	}

	cancel()
}
