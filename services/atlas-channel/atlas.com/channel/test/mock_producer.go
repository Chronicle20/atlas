package test

import (
	"atlas-channel/kafka/producer"
	"sync"

	kafkaproducer "github.com/Chronicle20/atlas-kafka/producer"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/segmentio/kafka-go"
)

// MockMessage represents a captured Kafka message with its topic
type MockMessage struct {
	Topic    string
	Messages []kafka.Message
}

// MockProducer is a test double for Kafka message production
type MockProducer struct {
	mu       sync.Mutex
	messages []MockMessage
	// Error can be set to simulate producer failures
	Error error
}

// NewMockProducer creates a new MockProducer instance
func NewMockProducer() *MockProducer {
	return &MockProducer{
		messages: make([]MockMessage, 0),
	}
}

// Provider returns a producer.Provider that captures messages instead of sending them
func (m *MockProducer) Provider() producer.Provider {
	return func(token string) kafkaproducer.MessageProducer {
		return func(provider model.Provider[[]kafka.Message]) error {
			if m.Error != nil {
				return m.Error
			}

			msgs, err := provider()
			if err != nil {
				return err
			}

			m.mu.Lock()
			defer m.mu.Unlock()
			m.messages = append(m.messages, MockMessage{
				Topic:    token,
				Messages: msgs,
			})
			return nil
		}
	}
}

// Messages returns all captured messages
func (m *MockProducer) Messages() []MockMessage {
	m.mu.Lock()
	defer m.mu.Unlock()
	result := make([]MockMessage, len(m.messages))
	copy(result, m.messages)
	return result
}

// MessageCount returns the total number of captured messages
func (m *MockProducer) MessageCount() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return len(m.messages)
}

// MessagesForTopic returns all messages for a specific topic
func (m *MockProducer) MessagesForTopic(topic string) []MockMessage {
	m.mu.Lock()
	defer m.mu.Unlock()
	var result []MockMessage
	for _, msg := range m.messages {
		if msg.Topic == topic {
			result = append(result, msg)
		}
	}
	return result
}

// HasMessage checks if a message was sent to the specified topic
func (m *MockProducer) HasMessage(topic string) bool {
	return len(m.MessagesForTopic(topic)) > 0
}

// Reset clears all captured messages
func (m *MockProducer) Reset() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.messages = make([]MockMessage, 0)
	m.Error = nil
}

// SetError configures the producer to return an error on the next call
func (m *MockProducer) SetError(err error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.Error = err
}
