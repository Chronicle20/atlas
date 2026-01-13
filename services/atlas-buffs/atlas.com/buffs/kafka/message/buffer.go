package message

import (
	"atlas-buffs/kafka/producer"
	"context"

	"github.com/Chronicle20/atlas-model/model"
	"github.com/segmentio/kafka-go"
	"github.com/sirupsen/logrus"
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
