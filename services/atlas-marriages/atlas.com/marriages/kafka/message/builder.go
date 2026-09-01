package message

import (
	kafka "github.com/segmentio/kafka-go"

	"github.com/Chronicle20/atlas/libs/atlas-kafka/producer"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/topic"
	"github.com/Chronicle20/atlas/libs/atlas-model/model"
)

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
func (bb *BufferBuilder) AddMessage(t topic.Token, provider model.Provider[[]kafka.Message]) *BufferBuilder {
	_ = bb.buffer.Put(t, provider)
	return bb
}

// AddConditionalMessage adds a message only if the condition is true
func (bb *BufferBuilder) AddConditionalMessage(condition bool, t topic.Token, provider model.Provider[[]kafka.Message]) *BufferBuilder {
	if condition {
		_ = bb.buffer.Put(t, provider)
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
