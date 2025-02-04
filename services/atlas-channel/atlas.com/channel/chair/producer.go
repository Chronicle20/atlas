package chair

import (
	"github.com/Chronicle20/atlas-kafka/producer"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/segmentio/kafka-go"
)

func commandUseProvider(worldId byte, channelId byte, mapId uint32, chairType string, chairId uint32, characterId uint32) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(characterId))
	value := &command[useChairCommandBody]{
		WorldId:   worldId,
		ChannelId: channelId,
		MapId:     mapId,
		Type:      CommandUseChair,
		Body: useChairCommandBody{
			CharacterId: characterId,
			ChairType:   chairType,
			ChairId:     chairId,
		},
	}
	return producer.SingleMessageProvider(key, value)
}

func commandCancelProvider(worldId byte, channelId byte, mapId uint32, characterId uint32) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(characterId))
	value := &command[cancelChairCommandBody]{
		WorldId:   worldId,
		ChannelId: channelId,
		MapId:     mapId,
		Type:      CommandCancelChair,
		Body: cancelChairCommandBody{
			CharacterId: characterId,
		},
	}
	return producer.SingleMessageProvider(key, value)
}
