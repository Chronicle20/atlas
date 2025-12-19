package expression

import (
	"atlas-expressions/kafka/message/expression"
	"github.com/Chronicle20/atlas-constants/channel"
	_map "github.com/Chronicle20/atlas-constants/map"
	"github.com/Chronicle20/atlas-constants/world"
	"github.com/Chronicle20/atlas-kafka/producer"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/google/uuid"
	"github.com/segmentio/kafka-go"
)

func expressionEventProvider(transactionId uuid.UUID, characterId uint32, worldId world.Id, channelId channel.Id, mapId _map.Id, expressionId uint32) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(characterId))
	value := &expression.StatusEvent{
		TransactionId: transactionId,
		CharacterId:   characterId,
		WorldId:       worldId,
		ChannelId:     channelId,
		MapId:         mapId,
		Expression:    expressionId,
	}
	return producer.SingleMessageProvider(key, value)
}
