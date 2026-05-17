package outbox

import (
	"context"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/segmentio/kafka-go"
)

// TopicWriterPool implements outboxlib.Publisher by maintaining one
// long-lived kafka.Writer per topic. The drainer only ever publishes a
// handful of topics from atlas-configurations (service + tenant config
// status), so the pool stays small and is keyed by the message's Topic
// field rather than the producer.Manager's env-var-token convention
// (which would require an inverse lookup from topic name to env var).
type TopicWriterPool struct {
	mu      sync.Mutex
	writers map[string]*kafka.Writer
	brokers []string
}

// NewTopicWriterPool reads BOOTSTRAP_SERVERS (comma-separated) once at
// construction. Pool entries are created lazily on first publish.
func NewTopicWriterPool() *TopicWriterPool {
	raw := os.Getenv("BOOTSTRAP_SERVERS")
	var brokers []string
	for _, b := range strings.Split(raw, ",") {
		if b = strings.TrimSpace(b); b != "" {
			brokers = append(brokers, b)
		}
	}
	return &TopicWriterPool{
		writers: make(map[string]*kafka.Writer),
		brokers: brokers,
	}
}

// WriteMessages groups msgs by topic and publishes each group through
// the writer for that topic, constructing writers on demand. Stops at the
// first publish error so the drainer can record it and retry.
func (p *TopicWriterPool) WriteMessages(ctx context.Context, msgs ...kafka.Message) error {
	if len(msgs) == 0 {
		return nil
	}
	byTopic := make(map[string][]kafka.Message)
	for _, m := range msgs {
		byTopic[m.Topic] = append(byTopic[m.Topic], m)
	}
	for topic, group := range byTopic {
		w := p.writerFor(topic)
		// kafka.Writer pins the topic from its Topic field; clear the
		// per-message topic so kafka-go doesn't complain about a topic
		// mismatch when the Writer has one configured.
		for i := range group {
			group[i].Topic = ""
		}
		if err := w.WriteMessages(ctx, group...); err != nil {
			return err
		}
	}
	return nil
}

// Close shuts down every cached writer. Safe to call repeatedly.
func (p *TopicWriterPool) Close() {
	p.mu.Lock()
	defer p.mu.Unlock()
	for _, w := range p.writers {
		_ = w.Close()
	}
	p.writers = map[string]*kafka.Writer{}
}

func (p *TopicWriterPool) writerFor(topic string) *kafka.Writer {
	p.mu.Lock()
	defer p.mu.Unlock()
	if w, ok := p.writers[topic]; ok {
		return w
	}
	w := &kafka.Writer{
		Addr:                   kafka.TCP(p.brokers...),
		Topic:                  topic,
		Balancer:               &kafka.LeastBytes{},
		BatchTimeout:           50 * time.Millisecond,
		AllowAutoTopicCreation: true,
	}
	p.writers[topic] = w
	return w
}
