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
	"sync"

	"github.com/segmentio/kafka-go"

	"github.com/Chronicle20/atlas/libs/atlas-kafka/producer"
)

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
}

func (w *CapturingWriter) Topic() string { return w.TopicName }

func (w *CapturingWriter) WriteMessages(_ context.Context, msgs ...kafka.Message) error {
	w.mu.Lock()
	defer w.mu.Unlock()
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

// Reset drops every recorded writer. Call it at the top of each test so one
// test's emissions are not visible to the next — the manager singleton itself
// is installed once per package from TestMain and must not be reset per test.
func (c *Capture) Reset() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.writers = map[string]*CapturingWriter{}
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
