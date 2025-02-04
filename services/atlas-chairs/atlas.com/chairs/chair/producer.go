package chair

import (
	"github.com/Chronicle20/atlas-kafka/producer"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/segmentio/kafka-go"
)

func statusEventUsedProvider(worldId byte, channelId byte, mapId uint32, chairType string, chairId uint32, characterId uint32) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(characterId))
	value := &statusEvent[statusEventUsedBody]{
		WorldId:   worldId,
		ChannelId: channelId,
		MapId:     mapId,
		ChairType: chairType,
		ChairId:   chairId,
		Type:      EventStatusTypeUsed,
		Body: statusEventUsedBody{
			CharacterId: characterId,
		},
	}
	return producer.SingleMessageProvider(key, value)
}

func statusEventCancelledProvider(worldId byte, channelId byte, mapId uint32, chairType string, chairId uint32, characterId uint32) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(characterId))
	value := &statusEvent[statusEventCancelledBody]{
		WorldId:   worldId,
		ChannelId: channelId,
		MapId:     mapId,
		ChairType: chairType,
		ChairId:   chairId,
		Type:      EventStatusTypeCancelled,
		Body: statusEventCancelledBody{
			CharacterId: characterId,
		},
	}
	return producer.SingleMessageProvider(key, value)
}
