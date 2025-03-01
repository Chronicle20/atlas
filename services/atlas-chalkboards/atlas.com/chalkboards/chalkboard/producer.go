package chalkboard

import (
	"github.com/Chronicle20/atlas-kafka/producer"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/segmentio/kafka-go"
)

func setStatusEventProvider(worldId byte, channelId byte, mapId uint32, characterId uint32, message string) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(mapId))
	value := &statusEvent[setStatusEventBody]{
		WorldId:     worldId,
		ChannelId:   channelId,
		MapId:       mapId,
		CharacterId: characterId,
		Type:        EventTopicStatusTypeSet,
		Body: setStatusEventBody{
			Message: message,
		},
	}
	return producer.SingleMessageProvider(key, value)
}

func clearStatusEventProvider(worldId byte, channelId byte, mapId uint32, characterId uint32) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(mapId))
	value := &statusEvent[clearStatusEventBody]{
		WorldId:     worldId,
		ChannelId:   channelId,
		MapId:       mapId,
		CharacterId: characterId,
		Type:        EventTopicStatusTypeClear,
		Body:        clearStatusEventBody{},
	}
	return producer.SingleMessageProvider(key, value)
}
