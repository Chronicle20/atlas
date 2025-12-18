package guild

import (
	guild2 "atlas-guilds/kafka/message/guild"
	"github.com/Chronicle20/atlas-constants/world"
	"github.com/Chronicle20/atlas-kafka/producer"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/google/uuid"
	"github.com/segmentio/kafka-go"
)

func statusEventRequestAgreementProvider(worldId world.Id, characterId uint32, proposedName string, transactionId uuid.UUID) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(characterId))
	value := &guild2.StatusEvent[guild2.StatusEventRequestAgreementBody]{
		WorldId:       byte(worldId),
		GuildId:       0,
		Type:          guild2.StatusEventTypeRequestAgreement,
		TransactionId: transactionId,
		Body: guild2.StatusEventRequestAgreementBody{
			ActorId:      characterId,
			ProposedName: proposedName,
		},
	}
	return producer.SingleMessageProvider(key, value)
}

func statusEventCreatedProvider(worldId byte, guildId uint32, transactionId uuid.UUID) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(guildId))
	value := &guild2.StatusEvent[guild2.StatusEventCreatedBody]{
		WorldId:       worldId,
		GuildId:       guildId,
		Type:          guild2.StatusEventTypeCreated,
		TransactionId: transactionId,
		Body:          guild2.StatusEventCreatedBody{},
	}
	return producer.SingleMessageProvider(key, value)
}

func statusEventDisbandedProvider(worldId byte, guildId uint32, members []uint32, transactionId uuid.UUID) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(guildId))
	value := &guild2.StatusEvent[guild2.StatusEventDisbandedBody]{
		WorldId:       worldId,
		GuildId:       guildId,
		Type:          guild2.StatusEventTypeDisbanded,
		TransactionId: transactionId,
		Body: guild2.StatusEventDisbandedBody{
			Members: members,
		},
	}
	return producer.SingleMessageProvider(key, value)
}

func statusEventEmblemUpdatedProvider(worldId byte, guildId uint32, logo uint16, logoColor byte, logoBackground uint16, logoBackgroundColor byte, transactionId uuid.UUID) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(guildId))
	value := &guild2.StatusEvent[guild2.StatusEventEmblemUpdatedBody]{
		WorldId:       worldId,
		GuildId:       guildId,
		Type:          guild2.StatusEventTypeEmblemUpdated,
		TransactionId: transactionId,
		Body: guild2.StatusEventEmblemUpdatedBody{
			Logo:                logo,
			LogoColor:           logoColor,
			LogoBackground:      logoBackground,
			LogoBackgroundColor: logoBackgroundColor,
		},
	}
	return producer.SingleMessageProvider(key, value)
}

func statusEventMemberStatusUpdatedProvider(worldId byte, guildId uint32, characterId uint32, online bool, transactionId uuid.UUID) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(guildId))
	value := &guild2.StatusEvent[guild2.StatusEventMemberStatusUpdatedBody]{
		WorldId:       worldId,
		GuildId:       guildId,
		Type:          guild2.StatusEventTypeMemberStatusUpdated,
		TransactionId: transactionId,
		Body: guild2.StatusEventMemberStatusUpdatedBody{
			CharacterId: characterId,
			Online:      online,
		},
	}
	return producer.SingleMessageProvider(key, value)
}

func statusEventMemberTitleUpdatedProvider(worldId byte, guildId uint32, characterId uint32, title byte, transactionId uuid.UUID) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(guildId))
	value := &guild2.StatusEvent[guild2.StatusEventMemberTitleUpdatedBody]{
		WorldId:       worldId,
		GuildId:       guildId,
		Type:          guild2.StatusEventTypeMemberTitleUpdated,
		TransactionId: transactionId,
		Body: guild2.StatusEventMemberTitleUpdatedBody{
			CharacterId: characterId,
			Title:       title,
		},
	}
	return producer.SingleMessageProvider(key, value)
}

func statusEventNoticeUpdatedProvider(worldId byte, guildId uint32, notice string, transactionId uuid.UUID) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(guildId))
	value := &guild2.StatusEvent[guild2.StatusEventNoticeUpdatedBody]{
		WorldId:       worldId,
		GuildId:       guildId,
		Type:          guild2.StatusEventTypeNoticeUpdated,
		TransactionId: transactionId,
		Body: guild2.StatusEventNoticeUpdatedBody{
			Notice: notice,
		},
	}
	return producer.SingleMessageProvider(key, value)
}

func statusEventCapacityUpdatedProvider(worldId byte, guildId uint32, capacity uint32, transactionId uuid.UUID) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(guildId))
	value := &guild2.StatusEvent[guild2.StatusEventCapacityUpdatedBody]{
		WorldId:       worldId,
		GuildId:       guildId,
		Type:          guild2.StatusEventTypeCapacityUpdated,
		TransactionId: transactionId,
		Body: guild2.StatusEventCapacityUpdatedBody{
			Capacity: capacity,
		},
	}
	return producer.SingleMessageProvider(key, value)
}

func statusEventMemberLeftProvider(worldId byte, guildId uint32, characterId uint32, force bool, transactionId uuid.UUID) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(guildId))
	value := &guild2.StatusEvent[guild2.StatusEventMemberLeftBody]{
		WorldId:       worldId,
		GuildId:       guildId,
		Type:          guild2.StatusEventTypeMemberLeft,
		TransactionId: transactionId,
		Body: guild2.StatusEventMemberLeftBody{
			CharacterId: characterId,
			Force:       force,
		},
	}
	return producer.SingleMessageProvider(key, value)
}

func statusEventMemberJoinedProvider(worldId byte, guildId uint32, characterId uint32, name string, jobId uint16, level byte, title byte, allianceTitle byte, transactionId uuid.UUID) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(guildId))
	value := &guild2.StatusEvent[guild2.StatusEventMemberJoinedBody]{
		WorldId:       worldId,
		GuildId:       guildId,
		Type:          guild2.StatusEventTypeMemberJoined,
		TransactionId: transactionId,
		Body: guild2.StatusEventMemberJoinedBody{
			CharacterId:   characterId,
			Name:          name,
			JobId:         jobId,
			Level:         level,
			Title:         title,
			Online:        true,
			AllianceTitle: allianceTitle,
		},
	}
	return producer.SingleMessageProvider(key, value)
}

func statusEventTitlesUpdatedProvider(worldId byte, guildId uint32, titles []string, transactionId uuid.UUID) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(guildId))
	value := &guild2.StatusEvent[guild2.StatusEventTitlesUpdatedBody]{
		WorldId:       worldId,
		GuildId:       guildId,
		Type:          guild2.StatusEventTypeTitlesUpdated,
		TransactionId: transactionId,
		Body: guild2.StatusEventTitlesUpdatedBody{
			Titles: titles,
		},
	}
	return producer.SingleMessageProvider(key, value)
}

func statusEventErrorProvider(worldId world.Id, characterId uint32, error string, transactionId uuid.UUID) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(characterId))
	value := &guild2.StatusEvent[guild2.StatusEventErrorBody]{
		WorldId:       byte(worldId),
		GuildId:       0,
		Type:          guild2.StatusEventTypeError,
		TransactionId: transactionId,
		Body: guild2.StatusEventErrorBody{
			ActorId: characterId,
			Error:   error,
		},
	}
	return producer.SingleMessageProvider(key, value)
}
