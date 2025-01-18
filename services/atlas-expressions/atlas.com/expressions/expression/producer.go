package expression

import (
	"github.com/Chronicle20/atlas-kafka/producer"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/segmentio/kafka-go"
)

func expressionEventProvider(characterId uint32, worldId byte, channelId byte, mapId uint32, expression uint32) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(characterId))
	value := &expressionEvent{
		CharacterId: characterId,
		WorldId:     worldId,
		ChannelId:   channelId,
		MapId:       mapId,
		Expression:  expression,
	}
	return producer.SingleMessageProvider(key, value)
}
