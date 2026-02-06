package message

import (
	"atlas-marriages/kafka/producer"

	"github.com/Chronicle20/atlas-model/model"
	"github.com/segmentio/kafka-go"
)

type Buffer struct {
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
	b.buffer[t] = append(b.buffer[t], ms...)
	return nil
}

func (b *Buffer) GetAll() map[string][]kafka.Message {
	return b.buffer
}

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

func EmitWithResult[M any, B any](p producer.Provider) func(func(*Buffer) func(B) (M, error)) func(B) (M, error) {
	return func(f func(*Buffer) func(B) (M, error)) func(B) (M, error) {
		return func(input B) (M, error) {
			var buf = NewBuffer()
			result, err := f(buf)(input)
			if err != nil {
				return result, err
			}
			for t, ms := range buf.GetAll() {
				if err = p(t)(model.FixedProvider(ms)); err != nil {
					return result, err
				}
			}
			return result, nil
		}
	}
}

// EmitMultiple accumulates multiple messages across multiple operations and emits them all at once
// This ensures transactional consistency across complex operations involving multiple events
func EmitMultiple(p producer.Provider) func(...func(*Buffer) error) error {
	return func(operations ...func(*Buffer) error) error {
		buf := NewBuffer()

		// Execute all operations and accumulate messages in the buffer
		for _, operation := range operations {
			if err := operation(buf); err != nil {
				return err
			}
		}

		// Emit all buffered messages in a single transaction
		for t, ms := range buf.GetAll() {
			if err := p(t)(model.FixedProvider(ms)); err != nil {
				return err
			}
		}

		return nil
	}
}

// BufferMessages provides a fluent interface for building complex message buffers
// Use this for operations that need to conditionally add different types of messages
type BufferBuilder struct {
	buffer *Buffer
}

// NewBufferBuilder creates a new buffer builder for constructing complex message operations
func NewBufferBuilder() *BufferBuilder {
	return &BufferBuilder{
		buffer: NewBuffer(),
	}
}

// AddMessage adds a message provider to the buffer
func (bb *BufferBuilder) AddMessage(topic string, provider model.Provider[[]kafka.Message]) *BufferBuilder {
	bb.buffer.Put(topic, provider)
	return bb
}

// AddConditionalMessage adds a message only if the condition is true
func (bb *BufferBuilder) AddConditionalMessage(condition bool, topic string, provider model.Provider[[]kafka.Message]) *BufferBuilder {
	if condition {
		bb.buffer.Put(topic, provider)
	}
	return bb
}

// Build returns the accumulated buffer
func (bb *BufferBuilder) Build() *Buffer {
	return bb.buffer
}

// EmitAll emits all messages in the buffer using the provided producer
func (bb *BufferBuilder) EmitAll(p producer.Provider) error {
	for t, ms := range bb.buffer.GetAll() {
		if err := p(t)(model.FixedProvider(ms)); err != nil {
			return err
		}
	}
	return nil
}
