package _map

import (
	"github.com/Chronicle20/atlas-kafka/producer"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/segmentio/kafka-go"
)

func enterMapProvider(worldId byte, channelId byte, mapId uint32, characterId uint32) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(mapId))
	value := &statusEvent[characterEnter]{
		WorldId:   worldId,
		ChannelId: channelId,
		MapId:     mapId,
		Type:      EventTopicMapStatusTypeCharacterEnter,
		Body: characterEnter{
			CharacterId: characterId,
		},
	}
	return producer.SingleMessageProvider(key, value)
}

func exitMapProvider(worldId byte, channelId byte, mapId uint32, characterId uint32) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(mapId))
	value := &statusEvent[characterExit]{
		WorldId:   worldId,
		ChannelId: channelId,
		MapId:     mapId,
		Type:      EventTopicMapStatusTypeCharacterExit,
		Body: characterExit{
			CharacterId: characterId,
		},
	}
	return producer.SingleMessageProvider(key, value)
}
