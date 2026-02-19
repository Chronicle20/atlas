package reactor_test

import (
	consumer2 "atlas-reactors/kafka/consumer/reactor"
	"testing"

	"github.com/Chronicle20/atlas-kafka/consumer"
	"github.com/Chronicle20/atlas-kafka/handler"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/sirupsen/logrus"
	"github.com/sirupsen/logrus/hooks/test"
)

func testLogger() logrus.FieldLogger {
	l, _ := test.NewNullLogger()
	return l
}

func TestInitConsumers(t *testing.T) {
	l := testLogger()
	consumerCount := 0
	rf := func(config consumer.Config, decorators ...model.Decorator[consumer.Config]) {
		consumerCount++
	}

	consumer2.InitConsumers(l)(rf)("test-consumer-group")
	if consumerCount != 1 {
		t.Fatalf("Expected 1 consumer to be registered, got %d", consumerCount)
	}
}

func TestInitHandlers(t *testing.T) {
	l := testLogger()
	handlerCount := 0
	rf := func(topic string, h handler.Handler) (string, error) {
		handlerCount++
		return topic, nil
	}

	consumer2.InitHandlers(l)(rf)
	if handlerCount != 3 {
		t.Fatalf("Expected 3 handlers to be registered, got %d", handlerCount)
	}
}
