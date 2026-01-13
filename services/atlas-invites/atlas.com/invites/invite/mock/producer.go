package mock

import (
	"sync"

	"github.com/Chronicle20/atlas-kafka/producer"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/segmentio/kafka-go"
)

// ProducerMock is a mock implementation of producer.Provider for testing.
// It captures all messages that would be sent to Kafka topics.
type ProducerMock struct {
	mu       sync.Mutex
	messages map[string][]kafka.Message
	err      error
}

// NewProducerMock creates a new mock producer.
func NewProducerMock() *ProducerMock {
	return &ProducerMock{
		messages: make(map[string][]kafka.Message),
	}
}

// SetError configures the mock to return an error on produce calls.
func (m *ProducerMock) SetError(err error) {
	m.err = err
}

// Provider returns a producer.Provider function for use in tests.
func (m *ProducerMock) Provider() func(token string) producer.MessageProducer {
	return func(token string) producer.MessageProducer {
		return func(p model.Provider[[]kafka.Message]) error {
			if m.err != nil {
				return m.err
			}
			msgs, err := p()
			if err != nil {
				return err
			}
			m.mu.Lock()
			defer m.mu.Unlock()
			m.messages[token] = append(m.messages[token], msgs...)
			return nil
		}
	}
}

// GetMessages returns all messages captured for a given topic.
func (m *ProducerMock) GetMessages(topic string) []kafka.Message {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.messages[topic]
}

// GetAllMessages returns all captured messages by topic.
func (m *ProducerMock) GetAllMessages() map[string][]kafka.Message {
	m.mu.Lock()
	defer m.mu.Unlock()
	result := make(map[string][]kafka.Message)
	for k, v := range m.messages {
		result[k] = append([]kafka.Message(nil), v...)
	}
	return result
}

// MessageCount returns the total number of messages captured.
func (m *ProducerMock) MessageCount() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	count := 0
	for _, msgs := range m.messages {
		count += len(msgs)
	}
	return count
}

// Reset clears all captured messages.
func (m *ProducerMock) Reset() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.messages = make(map[string][]kafka.Message)
	m.err = nil
}
