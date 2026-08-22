package message

import (
	"sync"

	"github.com/Chronicle20/atlas/libs/atlas-kafka/producer"

	"github.com/segmentio/kafka-go"

	"github.com/Chronicle20/atlas/libs/atlas-model/model"
)

// Buffer accumulates outbound Kafka messages by topic so they can be emitted
// together after the operation that produced them succeeds, rather than
// firing independently on a DB-write success path (DOM-30). Mirrors
// services/atlas-npc-shops/atlas.com/npc/kafka/message/message.go.
type Buffer struct {
	mu     sync.Mutex
	buffer map[string][]kafka.Message
}

func NewBuffer() *Buffer {
	return &Buffer{
		buffer: make(map[string][]kafka.Message),
	}
}

func (b *Buffer) Put(t string, p model.Provider[[]kafka.Message]) error {
	ms, err := p()
	if err != nil {
		return err
	}
	b.mu.Lock()
	defer b.mu.Unlock()
	b.buffer[t] = append(b.buffer[t], ms...)
	return nil
}

func (b *Buffer) GetAll() map[string][]kafka.Message {
	b.mu.Lock()
	defer b.mu.Unlock()
	result := make(map[string][]kafka.Message)
	for k, v := range b.buffer {
		result[k] = append([]kafka.Message(nil), v...)
	}
	return result
}

// Emit runs f against a fresh Buffer and only publishes the accumulated
// messages if f returns nil — so a failure never emits a partial or
// premature ack.
func Emit(p producer.Provider) func(f func(buf *Buffer) error) error {
	return func(f func(buf *Buffer) error) error {
		b := NewBuffer()
		err := f(b)
		if err != nil {
			return err
		}
		for t, ms := range b.GetAll() {
			err = p(t)(model.FixedProvider(ms))
			if err != nil {
				return err
			}
		}
		return nil
	}
}
