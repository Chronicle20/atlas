// Package message is atlas-events' producer-emit idiom, copied from
// services/atlas-buffs/atlas.com/buffs/kafka/message/buffer.go — the
// sibling-service precedent named in the task brief. This service has never
// produced to Kafka before this task; rather than hand-roll a writer, this
// is the same Buffer/Emit shape every other Atlas service producing
// multiple messages per call already uses.
package message

import (
	"context"

	"github.com/Chronicle20/atlas/libs/atlas-kafka/producer"

	"github.com/segmentio/kafka-go"
	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-model/model"
)

// Buffer accumulates kafka messages for atomic emission.
type Buffer struct {
	buffer map[string][]kafka.Message
}

// NewBuffer creates a new message buffer.
func NewBuffer() *Buffer {
	return &Buffer{
		buffer: make(map[string][]kafka.Message),
	}
}

// Put adds messages to the buffer for the given topic.
func (b *Buffer) Put(t string, p model.Provider[[]kafka.Message]) error {
	ms, err := p()
	if err != nil {
		return err
	}
	b.buffer[t] = append(b.buffer[t], ms...)
	return nil
}

// GetAll returns all buffered messages.
func (b *Buffer) GetAll() map[string][]kafka.Message {
	return b.buffer
}

// Emit sends all buffered messages using the provided producer.
// Returns error if any topic fails to emit.
func Emit(l logrus.FieldLogger, ctx context.Context) func(f func(buf *Buffer) error) error {
	return func(f func(buf *Buffer) error) error {
		b := NewBuffer()
		err := f(b)
		if err != nil {
			return err
		}

		p := producer.ProviderImpl(l)(ctx)
		for t, ms := range b.GetAll() {
			err = p(t)(model.FixedProvider(ms))
			if err != nil {
				return err
			}
		}
		return nil
	}
}
