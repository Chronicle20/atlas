package guild

import (
	"atlas-saga-orchestrator/kafka/message/guild"

	"github.com/google/uuid"
	"github.com/segmentio/kafka-go"

	"github.com/Chronicle20/atlas/libs/atlas-constants/channel"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/producer"
	"github.com/Chronicle20/atlas/libs/atlas-model/model"
)

func RequestNameProvider(transactionId uuid.UUID, ch channel.Model, characterId uint32) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(characterId))
	value := &guild.Command[guild.RequestNameBody]{
		TransactionId: transactionId,
		CharacterId:   characterId,
		Type:          guild.CommandTypeRequestName,
		Body: guild.RequestNameBody{
			WorldId:   ch.WorldId(),
			ChannelId: ch.Id(),
		},
	}
	return producer.SingleMessageProvider(key, value)
}

func RequestEmblemProvider(transactionId uuid.UUID, ch channel.Model, characterId uint32) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(characterId))
	value := &guild.Command[guild.RequestEmblemBody]{
		TransactionId: transactionId,
		CharacterId:   characterId,
		Type:          guild.CommandTypeRequestEmblem,
		Body: guild.RequestEmblemBody{
			WorldId:   ch.WorldId(),
			ChannelId: ch.Id(),
		},
	}
	return producer.SingleMessageProvider(key, value)
}

func RequestDisbandProvider(transactionId uuid.UUID, ch channel.Model, characterId uint32) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(characterId))
	value := &guild.Command[guild.RequestDisbandBody]{
		TransactionId: transactionId,
		CharacterId:   characterId,
		Type:          guild.CommandTypeRequestDisband,
		Body: guild.RequestDisbandBody{
			WorldId:   ch.WorldId(),
			ChannelId: ch.Id(),
		},
	}
	return producer.SingleMessageProvider(key, value)
}

func RequestLeaveProvider(transactionId uuid.UUID, characterId uint32, guildId uint32, force bool) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(characterId))
	value := &guild.Command[guild.LeaveBody]{
		TransactionId: transactionId,
		CharacterId:   characterId,
		Type:          guild.CommandTypeLeave,
		Body: guild.LeaveBody{
			GuildId: guildId,
			Force:   force,
		},
	}
	return producer.SingleMessageProvider(key, value)
}

// RequestRejoinProvider builds the REJOIN command that undoes a forced LEAVE,
// restoring characterId to guildId at the rank recorded in the saga step's
// payload (task-227 FR-4.8).
func RequestRejoinProvider(transactionId uuid.UUID, characterId uint32, guildId uint32, title byte) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(characterId))
	value := &guild.Command[guild.RejoinBody]{
		TransactionId: transactionId,
		CharacterId:   characterId,
		Type:          guild.CommandTypeRejoin,
		Body: guild.RejoinBody{
			GuildId: guildId,
			Title:   title,
		},
	}
	return producer.SingleMessageProvider(key, value)
}

func RequestCapacityIncreaseProvider(transactionId uuid.UUID, ch channel.Model, characterId uint32) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(characterId))
	value := &guild.Command[guild.RequestCapacityIncreaseBody]{
		TransactionId: transactionId,
		CharacterId:   characterId,
		Type:          guild.CommandTypeRequestCapacityIncrease,
		Body: guild.RequestCapacityIncreaseBody{
			WorldId:   ch.WorldId(),
			ChannelId: ch.Id(),
		},
	}
	return producer.SingleMessageProvider(key, value)
}
