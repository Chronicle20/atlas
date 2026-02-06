package saga

import (
	"atlas-saga-orchestrator/kafka/message/saga"

	"github.com/Chronicle20/atlas-kafka/producer"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/google/uuid"
	"github.com/segmentio/kafka-go"
)

func CompletedStatusEventProvider(transactionId uuid.UUID) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(transactionId.ID()))
	value := &saga.StatusEvent[saga.StatusEventCompletedBody]{
		TransactionId: transactionId,
		Type:          saga.StatusEventTypeCompleted,
		Body:          saga.StatusEventCompletedBody{},
	}
	return producer.SingleMessageProvider(key, value)
}

func FailedStatusEventProvider(transactionId uuid.UUID, characterId uint32, sagaType string, errorCode string, reason string, failedStep string) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(transactionId.ID()))
	value := &saga.StatusEvent[saga.StatusEventFailedBody]{
		TransactionId: transactionId,
		Type:          saga.StatusEventTypeFailed,
		Body: saga.StatusEventFailedBody{
			Reason:      reason,
			FailedStep:  failedStep,
			CharacterId: characterId,
			SagaType:    sagaType,
			ErrorCode:   errorCode,
		},
	}
	return producer.SingleMessageProvider(key, value)
}
