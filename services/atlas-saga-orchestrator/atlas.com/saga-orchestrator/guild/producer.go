package guild

import (
	"atlas-saga-orchestrator/kafka/message/guild"
	"github.com/Chronicle20/atlas-kafka/producer"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/google/uuid"
	"github.com/segmentio/kafka-go"
)

func RequestNameProvider(transactionId uuid.UUID, worldId byte, channelId byte, characterId uint32) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(characterId))
	value := &guild.Command[guild.RequestNameBody]{
		TransactionId: transactionId,
		CharacterId:   characterId,
		Type:          guild.CommandTypeRequestName,
		Body: guild.RequestNameBody{
			WorldId:   worldId,
			ChannelId: channelId,
		},
	}
	return producer.SingleMessageProvider(key, value)
}

func RequestEmblemProvider(transactionId uuid.UUID, worldId byte, channelId byte, characterId uint32) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(characterId))
	value := &guild.Command[guild.RequestEmblemBody]{
		TransactionId: transactionId,
		CharacterId:   characterId,
		Type:          guild.CommandTypeRequestEmblem,
		Body: guild.RequestEmblemBody{
			WorldId:   worldId,
			ChannelId: channelId,
		},
	}
	return producer.SingleMessageProvider(key, value)
}

func RequestDisbandProvider(transactionId uuid.UUID, worldId byte, channelId byte, characterId uint32) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(characterId))
	value := &guild.Command[guild.RequestDisbandBody]{
		TransactionId: transactionId,
		CharacterId:   characterId,
		Type:          guild.CommandTypeRequestDisband,
		Body: guild.RequestDisbandBody{
			WorldId:   worldId,
			ChannelId: channelId,
		},
	}
	return producer.SingleMessageProvider(key, value)
}

func RequestCapacityIncreaseProvider(transactionId uuid.UUID, worldId byte, channelId byte, characterId uint32) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(characterId))
	value := &guild.Command[guild.RequestCapacityIncreaseBody]{
		TransactionId: transactionId,
		CharacterId:   characterId,
		Type:          guild.CommandTypeRequestCapacityIncrease,
		Body: guild.RequestCapacityIncreaseBody{
			WorldId:   worldId,
			ChannelId: channelId,
		},
	}
	return producer.SingleMessageProvider(key, value)
}
