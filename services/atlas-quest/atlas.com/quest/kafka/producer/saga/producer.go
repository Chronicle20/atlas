package saga

import (
	"atlas-quest/kafka/message/saga"
	"context"
	"fmt"

	kafka "github.com/segmentio/kafka-go"
	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-kafka/producer"
	"github.com/Chronicle20/atlas/libs/atlas-model/model"
)

// SagaCommandProvider creates a kafka message for a saga command
func SagaCommandProvider(s saga.Saga) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(s.TransactionId.ID()))
	return producer.SingleMessageProvider(key, s)
}

// EmitSaga emits a saga command to the saga orchestrator
func EmitSaga(l logrus.FieldLogger, ctx context.Context, s saga.Saga) error {
	topicToken := saga.EnvCommandTopic
	sd := producer.SpanHeaderDecorator(ctx)
	td := producer.TenantHeaderDecorator(ctx)
	ed := producer.EnvHeaderDecorator(ctx)
	return producer.Produce(l)(producer.ManagerWriterProvider(l)(topicToken))(sd, td, ed)(SagaCommandProvider(s))
}

func stepId(n int) string {
	return fmt.Sprintf("step_%d", n)
}
