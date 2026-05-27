// Package producertest provides test stubs for atlas-kafka producers.
//
// The production producer retries failed writes 10 times with exponential
// backoff (100ms → 10s, ~42s total) when BOOTSTRAP_SERVERS is unset or the
// broker is unreachable. In unit tests this turns every emit into a multi-
// second hang. InstallNoop swaps the process-wide producer manager to a
// writer factory that discards all messages, so calls to Produce / Emit
// succeed instantly without touching a broker.
//
// Usage from a service's test package:
//
//	func TestMain(m *testing.M) {
//	    producertest.InstallNoop()
//	    os.Exit(m.Run())
//	}
package producertest

import (
	"context"

	"github.com/Chronicle20/atlas/libs/atlas-kafka/producer"
	"github.com/segmentio/kafka-go"
)

// NoopWriter implements producer.Writer by discarding every message.
type NoopWriter struct {
	TopicName string
}

func (w NoopWriter) Topic() string                                              { return w.TopicName }
func (w NoopWriter) WriteMessages(_ context.Context, _ ...kafka.Message) error { return nil }
func (w NoopWriter) Close() error                                               { return nil }

// InstallNoop resets the producer manager singleton and reinstalls it with
// a writer factory that returns NoopWriter for every topic.
func InstallNoop() {
	producer.ResetInstance()
	producer.GetManager(producer.ConfigWriterFactory(func(topicName string) producer.Writer {
		return NoopWriter{TopicName: topicName}
	}))
}
