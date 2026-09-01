package message

import (
	"context"

	"github.com/Chronicle20/atlas/libs/atlas-kafka/producer"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/topic"

	"github.com/segmentio/kafka-go"
	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-model/model"
)

// Buffer accumulates kafka messages for atomic emission.
type Buffer struct {
	buffer map[topic.Token][]kafka.Message
}

// NewBuffer creates a new message buffer.
func NewBuffer() *Buffer {
	return &Buffer{
		buffer: make(map[topic.Token][]kafka.Message),
	}
}

// Put adds messages to the buffer for the given topic.
func (b *Buffer) Put(t topic.Token, p model.Provider[[]kafka.Message]) error {
	ms, err := p()
	if err != nil {
		return err
	}
	b.buffer[t] = append(b.buffer[t], ms...)
	return nil
}

// GetAll returns all buffered messages.
func (b *Buffer) GetAll() map[topic.Token][]kafka.Message {
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

// EmitWithResult is a variant of Emit that returns a result value along with handling emissions.
func EmitWithResult[M any](l logrus.FieldLogger, ctx context.Context) func(f func(buf *Buffer) (M, error)) (M, error) {
	return func(f func(buf *Buffer) (M, error)) (M, error) {
		b := NewBuffer()
		result, err := f(b)
		if err != nil {
			return result, err
		}

		p := producer.ProviderImpl(l)(ctx)
		for t, ms := range b.GetAll() {
			if err = p(t)(model.FixedProvider(ms)); err != nil {
				return result, err
			}
		}
		return result, nil
	}
}
