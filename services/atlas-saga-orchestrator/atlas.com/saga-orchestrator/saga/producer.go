package saga

import (
	"atlas-saga-orchestrator/kafka/message/saga"
	"github.com/Chronicle20/atlas-kafka/producer"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/google/uuid"
	"github.com/segmentio/kafka-go"
)

func CompletedStatusEventProvider(sagaId uuid.UUID, transactionId uuid.UUID) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(sagaId.ID()))
	value := &saga.StatusEvent[saga.StatusEventCompletedBody]{
		SagaId:        sagaId,
		Type:          saga.StatusEventTypeCompleted,
		TransactionId: transactionId,
		Body: saga.StatusEventCompletedBody{
			TransactionId: transactionId,
		},
	}
	return producer.SingleMessageProvider(key, value)
}