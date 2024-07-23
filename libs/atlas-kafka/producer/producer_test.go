package producer

import (
	"context"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/segmentio/kafka-go"
	"github.com/sirupsen/logrus/hooks/test"
	"testing"
)

type testEvent struct {
	AccountId uint32 `json:"accountId"`
	Name      string `json:"name"`
	Status    string `json:"status"`
}

// MockWriter simulates the behavior of a kafka.Writer
type MockWriter struct {
	topic           string
	writtenMessages []kafka.Message
}

// WriteMessages simulates writing messages to Kafka
func (mw *MockWriter) WriteMessages(ctx context.Context, msgs ...kafka.Message) error {
	// Append messages to writtenMessages for later inspection
	mw.writtenMessages = append(mw.writtenMessages, msgs...)
	return nil
}

func (mw *MockWriter) Topic() string {
	return mw.topic
}

// Close simulates closing the writer
func (mw *MockWriter) Close() error {
	// Simulate closing resources if needed
	return nil
}

func (mw *MockWriter) WrittenMessages() []kafka.Message {
	return mw.writtenMessages
}

func TestProducer(t *testing.T) {
	l, _ := test.NewNullLogger()
	mw := &MockWriter{}

	key := []byte{0, 0, 0}
	e := &testEvent{}
	err := Produce(l)(model.FixedProvider[Writer](mw))()(SingleMessageProvider(key, e))
	if err != nil {
		t.Fatalf("Error producing event: %s", err)
	}

	if len(mw.WrittenMessages()) != 1 {
		t.Fatalf("Wrong number of messages: %d", len(mw.WrittenMessages()))
	}
}

func TestProducer2(t *testing.T) {
	l, _ := test.NewNullLogger()
	mw := &MockWriter{}

	key := []byte{0, 0, 0}
	e := &testEvent{}
	err := Produce(l)(model.FixedProvider[Writer](mw))()(MessageProvider(model.FixedSliceProvider([]RawMessage{{Key: key, Value: e}, {Key: key, Value: e}})))
	if err != nil {
		t.Fatalf("Error producing event: %s", err)
	}

	if len(mw.WrittenMessages()) != 2 {
		t.Fatalf("Wrong number of messages: %d", len(mw.WrittenMessages()))
	}
}
