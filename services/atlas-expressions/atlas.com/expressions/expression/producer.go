package expression

import (
	"atlas-expressions/kafka/message/expression"

	"github.com/Chronicle20/atlas-constants/field"
	"github.com/Chronicle20/atlas-kafka/producer"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/google/uuid"
	"github.com/segmentio/kafka-go"
)

func expressionEventProvider(transactionId uuid.UUID, characterId uint32, field field.Model, expressionId uint32) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(characterId))
	value := &expression.StatusEvent{
		TransactionId: transactionId,
		CharacterId:   characterId,
		WorldId:       field.WorldId(),
		ChannelId:     field.ChannelId(),
		MapId:         field.MapId(),
		Instance:      field.Instance(),
		Expression:    expressionId,
	}
	return producer.SingleMessageProvider(key, value)
}
