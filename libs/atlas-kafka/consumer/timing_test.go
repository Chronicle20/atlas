package consumer_test

import (
	"context"
	"io"
	"sync"
	"testing"
	"time"

	"github.com/Chronicle20/atlas/libs/atlas-kafka/consumer"
	"github.com/segmentio/kafka-go"
	"github.com/sirupsen/logrus"
	"github.com/sirupsen/logrus/hooks/test"
	"go.opentelemetry.io/otel"
)

// snapshotForTopic returns the Snapshot of the registered consumer for topic,
// failing the test if none exists. Shared by timing and idle/stuck tests.
func snapshotForTopic(t *testing.T, cm *consumer.Manager, topic string) consumer.Snapshot {
	t.Helper()
	for _, c := range cm.Consumers() {
		s := c.Snapshot()
		if s.Topic == topic {
			return s
		}
	}
	t.Fatalf("no consumer registered for topic %s", topic)
	return consumer.Snapshot{}
}

// TestSnapshotPhaseTimings drives one recreate (io.EOF) followed by one
// handled message and asserts every phase-timing field is populated:
// TotalBackoff from the recreate wait, TimeToFirstFetch on the second
// reader, LastFetchDuration from the successful fetch, and handler
// durations from a deliberately slow handler.
func TestSnapshotPhaseTimings(t *testing.T) {
	consumer.ResetInstance()
	l, _ := test.NewNullLogger()
	wg := &sync.WaitGroup{}
	otel.SetTracerProvider(&MockTracerProvider{})

	r1 := &scriptedReader{script: []scriptedFetch{{err: io.EOF}}}
	r2 := &scriptedReader{script: []scriptedFetch{{msg: kafka.Message{Value: []byte("timed")}}}}
	rp := consumer.ConfigReaderProducer(readerFactory(t, r1, r2))

	ctx, cancel := context.WithCancel(context.Background())
	defer wg.Wait()

	cm := consumer.GetManager(rp)
	c := consumer.NewConfig([]string{""}, "timing-consumer", "timing-topic", "test-group")
	cm.AddConsumer(l, ctx, wg)(c)

	handled := make(chan struct{})
	_, _ = cm.RegisterHandler("timing-topic", func(_ logrus.FieldLogger, _ context.Context, _ kafka.Message) (bool, error) {
		time.Sleep(30 * time.Millisecond)
		close(handled)
		return true, nil
	})

	select {
	case <-handled:
	case <-time.After(5 * time.Second):
		t.Fatal("message was never handled after recreate")
	}

	// Handler duration is recorded after processMessage returns; poll
	// briefly for the snapshot to reflect it.
	var s consumer.Snapshot
	deadline := time.Now().Add(2 * time.Second)
	for {
		s = snapshotForTopic(t, cm, "timing-topic")
		if s.MaxHandlerDuration > 0 || time.Now().After(deadline) {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}

	if s.TotalBackoff < 500*time.Millisecond {
		t.Fatalf("expected TotalBackoff >= 500ms (one recreate backoff), got %v", s.TotalBackoff)
	}
	if s.TimeToFirstFetch <= 0 {
		t.Fatalf("expected TimeToFirstFetch > 0 after a successful fetch, got %v", s.TimeToFirstFetch)
	}
	if s.LastFetchDuration <= 0 {
		t.Fatalf("expected LastFetchDuration > 0, got %v", s.LastFetchDuration)
	}
	if s.MaxFetchDuration < s.LastFetchDuration {
		t.Fatalf("expected MaxFetchDuration >= LastFetchDuration, got %v < %v", s.MaxFetchDuration, s.LastFetchDuration)
	}
	if s.MaxHandlerDuration < 30*time.Millisecond {
		t.Fatalf("expected MaxHandlerDuration >= 30ms (sleeping handler), got %v", s.MaxHandlerDuration)
	}
	if s.LastHandlerDuration < 30*time.Millisecond {
		t.Fatalf("expected LastHandlerDuration >= 30ms, got %v", s.LastHandlerDuration)
	}

	cancel()
}
