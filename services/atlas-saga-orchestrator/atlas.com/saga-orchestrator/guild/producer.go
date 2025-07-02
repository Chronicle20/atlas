package guild

import (
	"atlas-saga-orchestrator/kafka/message/guild"
	"github.com/Chronicle20/atlas-kafka/producer"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/segmentio/kafka-go"
)

func RequestNameProvider(worldId byte, channelId byte, characterId uint32) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(characterId))
	value := &guild.Command[guild.RequestNameBody]{
		CharacterId: characterId,
		Type:        guild.CommandTypeRequestName,
		Body: guild.RequestNameBody{
			WorldId:   worldId,
			ChannelId: channelId,
		},
	}
	return producer.SingleMessageProvider(key, value)
}

func RequestEmblemProvider(worldId byte, channelId byte, characterId uint32) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(characterId))
	value := &guild.Command[guild.RequestEmblemBody]{
		CharacterId: characterId,
		Type:        guild.CommandTypeRequestEmblem,
		Body: guild.RequestEmblemBody{
			WorldId:   worldId,
			ChannelId: channelId,
		},
	}
	return producer.SingleMessageProvider(key, value)
}

func RequestDisbandProvider(worldId byte, channelId byte, characterId uint32) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(characterId))
	value := &guild.Command[guild.RequestDisbandBody]{
		CharacterId: characterId,
		Type:        guild.CommandTypeRequestDisband,
		Body: guild.RequestDisbandBody{
			WorldId:   worldId,
			ChannelId: channelId,
		},
	}
	return producer.SingleMessageProvider(key, value)
}

func RequestCapacityIncreaseProvider(worldId byte, channelId byte, characterId uint32) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(characterId))
	value := &guild.Command[guild.RequestCapacityIncreaseBody]{
		CharacterId: characterId,
		Type:        guild.CommandTypeRequestCapacityIncrease,
		Body: guild.RequestCapacityIncreaseBody{
			WorldId:   worldId,
			ChannelId: channelId,
		},
	}
	return producer.SingleMessageProvider(key, value)
}
