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
//
// InstallCapturing is the same swap, except the writers retain what was
// written so a test can assert on the emitted messages.
package producertest

import (
	"context"
	"errors"
	"sync"

	"github.com/segmentio/kafka-go"

	"github.com/Chronicle20/atlas/libs/atlas-kafka/producer"
)

// ErrSimulatedProduceFailure is returned by a CapturingWriter whose topic was
// marked failing via Capture.FailTopic — used to test a caller's handling of
// an unproduceable message (e.g. a broker outage) without a service-local
// stub writer.
var ErrSimulatedProduceFailure = errors.New("producertest: simulated produce failure")

// NoopWriter implements producer.Writer by discarding every message.
type NoopWriter struct {
	TopicName string
}

func (w NoopWriter) Topic() string                                             { return w.TopicName }
func (w NoopWriter) WriteMessages(_ context.Context, _ ...kafka.Message) error { return nil }
func (w NoopWriter) Close() error                                              { return nil }

// InstallNoop resets the producer manager singleton and reinstalls it with
// a writer factory that returns NoopWriter for every topic.
func InstallNoop() {
	producer.ResetInstance()
	producer.GetManager(producer.ConfigWriterFactory(func(topicName string) producer.Writer {
		return NoopWriter{TopicName: topicName}
	}))
}

// CapturingWriter implements producer.Writer by recording every message
// written to one topic instead of discarding it.
type CapturingWriter struct {
	TopicName string
	mu        sync.Mutex
	msgs      []kafka.Message
	fail      bool
}

func (w *CapturingWriter) Topic() string { return w.TopicName }

func (w *CapturingWriter) WriteMessages(_ context.Context, msgs ...kafka.Message) error {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.fail {
		return ErrSimulatedProduceFailure
	}
	w.msgs = append(w.msgs, msgs...)
	return nil
}

func (w *CapturingWriter) Close() error { return nil }

// Messages returns a copy of everything written to this topic so far.
func (w *CapturingWriter) Messages() []kafka.Message {
	w.mu.Lock()
	defer w.mu.Unlock()
	out := make([]kafka.Message, len(w.msgs))
	copy(out, w.msgs)
	return out
}

// Capture holds one CapturingWriter per topic the code under test emitted to.
type Capture struct {
	mu      sync.Mutex
	writers map[string]*CapturingWriter
}

// Topic returns the writer for a topic, or nil if nothing was ever written to
// it. Topics resolve through topic.EnvProvider, which falls back to the env
// var token name when the variable is unset — so in tests the topic is
// normally the EnvCommandTopic / EnvEventTopic constant itself.
func (c *Capture) Topic(topicName string) *CapturingWriter {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.writers[topicName]
}

// Messages returns everything written to a topic, or nil if there was nothing.
func (c *Capture) Messages(topicName string) []kafka.Message {
	if w := c.Topic(topicName); w != nil {
		return w.Messages()
	}
	return nil
}

// Reset clears every recorded message (and any FailTopic marker) so one
// test's emissions are not visible to the next. Call it at the top of each
// test — the manager singleton itself is installed once per package from
// TestMain and must not be reset per test.
//
// Reset clears each existing CapturingWriter's state IN PLACE rather than
// replacing the writer objects: producer.Manager caches one Writer per topic
// for the lifetime of the singleton (only producer.ResetInstance clears
// that cache), so a topic already touched by an earlier test already has a
// Manager-cached reference to its *CapturingWriter. Discarding and
// recreating that object here would silently orphan it — later writes would
// keep landing on the Manager's stale reference while this Capture's map
// pointed at a new, empty one, so Messages() would read back nothing for a
// topic that in fact received messages.
func (c *Capture) Reset() {
	c.mu.Lock()
	defer c.mu.Unlock()
	for _, w := range c.writers {
		w.mu.Lock()
		w.msgs = nil
		w.fail = false
		w.mu.Unlock()
	}
}

// FailTopic marks (or clears) a topic's writer as failing: WriteMessages
// returns ErrSimulatedProduceFailure instead of recording, without disturbing
// any other topic's writer. Used to test produce-failure handling (e.g. a
// broker outage on one specific topic) via the shared Capture rather than a
// service-local stub writer. The marker persists until cleared or the next
// Reset.
func (c *Capture) FailTopic(topicName string, fail bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	w, ok := c.writers[topicName]
	if !ok {
		w = &CapturingWriter{TopicName: topicName}
		c.writers[topicName] = w
	}
	w.mu.Lock()
	w.fail = fail
	w.mu.Unlock()
}

func (c *Capture) writer(topicName string) producer.Writer {
	c.mu.Lock()
	defer c.mu.Unlock()
	if w, ok := c.writers[topicName]; ok {
		return w
	}
	w := &CapturingWriter{TopicName: topicName}
	c.writers[topicName] = w
	return w
}

// InstallCapturing resets the producer manager singleton and reinstalls it
// with a writer factory that records every message, returning the Capture the
// test asserts against. Like InstallNoop it is called once per package:
//
//	var emitted *producertest.Capture
//
//	func TestMain(m *testing.M) {
//	    emitted = producertest.InstallCapturing()
//	    os.Exit(m.Run())
//	}
func InstallCapturing() *Capture {
	c := &Capture{writers: map[string]*CapturingWriter{}}
	producer.ResetInstance()
	producer.GetManager(producer.ConfigWriterFactory(c.writer))
	return c
}
