package guild

import (
	guild2 "atlas-npc-conversations/kafka/message/guild"
	"github.com/Chronicle20/atlas-kafka/producer"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/segmentio/kafka-go"
)

func requestNameProvider(worldId byte, channelId byte, characterId uint32) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(characterId))
	value := &guild2.Command[guild2.RequestNameBody]{
		CharacterId: characterId,
		Type:        guild2.CommandTypeRequestName,
		Body: guild2.RequestNameBody{
			WorldId:   worldId,
			ChannelId: channelId,
		},
	}
	return producer.SingleMessageProvider(key, value)
}

func requestEmblemProvider(worldId byte, channelId byte, characterId uint32) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(characterId))
	value := &guild2.Command[guild2.RequestEmblemBody]{
		CharacterId: characterId,
		Type:        guild2.CommandTypeRequestEmblem,
		Body: guild2.RequestEmblemBody{
			WorldId:   worldId,
			ChannelId: channelId,
		},
	}
	return producer.SingleMessageProvider(key, value)
}

func requestDisbandProvider(worldId byte, channelId byte, characterId uint32) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(characterId))
	value := &guild2.Command[guild2.RequestDisbandBody]{
		CharacterId: characterId,
		Type:        guild2.CommandTypeRequestDisband,
		Body: guild2.RequestDisbandBody{
			WorldId:   worldId,
			ChannelId: channelId,
		},
	}
	return producer.SingleMessageProvider(key, value)
}

func requestCapacityIncreaseProvider(worldId byte, channelId byte, characterId uint32) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(characterId))
	value := &guild2.Command[guild2.RequestCapacityIncreaseBody]{
		CharacterId: characterId,
		Type:        guild2.CommandTypeRequestCapacityIncrease,
		Body: guild2.RequestCapacityIncreaseBody{
			WorldId:   worldId,
			ChannelId: channelId,
		},
	}
	return producer.SingleMessageProvider(key, value)
}
