package craft

import (
	"context"

	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-kafka/producer"
	saga "github.com/Chronicle20/atlas/libs/atlas-saga"
)

// EnvCommandTopic is the saga command topic env var name
// (deploy/k8s/base/env-configmap.yaml), shared across every
// saga-emitting service.
const EnvCommandTopic = "COMMAND_TOPIC_SAGA"

// kafkaEmitter is the concrete, Kafka-backed SagaEmitter (Task 23's
// interface) that Task 24 wires into main.go. Every other package in this
// module tests against SagaEmitter directly, so this type has no exported
// surface beyond NewKafkaEmitter.
type kafkaEmitter struct {
	l   logrus.FieldLogger
	ctx context.Context
}

// NewKafkaEmitter builds the SagaEmitter Processor.Create submits an
// accepted craft's saga through.
func NewKafkaEmitter(l logrus.FieldLogger, ctx context.Context) SagaEmitter {
	return kafkaEmitter{l: l, ctx: ctx}
}

func (e kafkaEmitter) Emit(s saga.Saga) error {
	key := []byte(s.TransactionId.String())
	provider := producer.SingleMessageProvider(key, &s)
	return producer.ProviderImpl(e.l)(e.ctx)(EnvCommandTopic)(provider)
}
