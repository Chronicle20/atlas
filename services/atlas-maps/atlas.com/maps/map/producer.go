package _map

import (
	mapKafka "atlas-maps/kafka/message/map"
	"github.com/Chronicle20/atlas-constants/channel"
	_map "github.com/Chronicle20/atlas-constants/map"
	"github.com/Chronicle20/atlas-constants/world"
	"github.com/Chronicle20/atlas-kafka/producer"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/google/uuid"
	"github.com/segmentio/kafka-go"
)

func enterMapProvider(transactionId uuid.UUID, worldId world.Id, channelId channel.Id, mapId _map.Id, characterId uint32) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(mapId))
	value := &mapKafka.StatusEvent[mapKafka.CharacterEnter]{
		TransactionId: transactionId,
		WorldId:       worldId,
		ChannelId:     channelId,
		MapId:         mapId,
		Type:          mapKafka.EventTopicMapStatusTypeCharacterEnter,
		Body: mapKafka.CharacterEnter{
			CharacterId: characterId,
		},
	}
	return producer.SingleMessageProvider(key, value)
}

func exitMapProvider(transactionId uuid.UUID, worldId world.Id, channelId channel.Id, mapId _map.Id, characterId uint32) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(mapId))
	value := &mapKafka.StatusEvent[mapKafka.CharacterExit]{
		TransactionId: transactionId,
		WorldId:       worldId,
		ChannelId:     channelId,
		MapId:         mapId,
		Type:          mapKafka.EventTopicMapStatusTypeCharacterExit,
		Body: mapKafka.CharacterExit{
			CharacterId: characterId,
		},
	}
	return producer.SingleMessageProvider(key, value)
}
