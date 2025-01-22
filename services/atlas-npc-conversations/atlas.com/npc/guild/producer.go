package guild

import (
	"github.com/Chronicle20/atlas-kafka/producer"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/segmentio/kafka-go"
)

func requestNameProvider(worldId byte, channelId byte, characterId uint32) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(characterId))
	value := &command[requestNameBody]{
		CharacterId: characterId,
		Type:        CommandTypeRequestName,
		Body: requestNameBody{
			WorldId:   worldId,
			ChannelId: channelId,
		},
	}
	return producer.SingleMessageProvider(key, value)
}

func requestEmblemProvider(worldId byte, channelId byte, characterId uint32) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(characterId))
	value := &command[requestEmblemBody]{
		CharacterId: characterId,
		Type:        CommandTypeRequestEmblem,
		Body: requestEmblemBody{
			WorldId:   worldId,
			ChannelId: channelId,
		},
	}
	return producer.SingleMessageProvider(key, value)
}

func requestDisbandProvider(worldId byte, channelId byte, characterId uint32) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(characterId))
	value := &command[requestDisbandBody]{
		CharacterId: characterId,
		Type:        CommandTypeRequestDisband,
		Body: requestDisbandBody{
			WorldId:   worldId,
			ChannelId: channelId,
		},
	}
	return producer.SingleMessageProvider(key, value)
}

func requestCapacityIncreaseProvider(worldId byte, channelId byte, characterId uint32) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(characterId))
	value := &command[requestCapacityIncreaseBody]{
		CharacterId: characterId,
		Type:        CommandTypeRequestCapacityIncrease,
		Body: requestCapacityIncreaseBody{
			WorldId:   worldId,
			ChannelId: channelId,
		},
	}
	return producer.SingleMessageProvider(key, value)
}
