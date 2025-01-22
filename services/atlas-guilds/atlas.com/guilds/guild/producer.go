package guild

import (
	"github.com/Chronicle20/atlas-kafka/producer"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/segmentio/kafka-go"
)

func statusEventRequestAgreementProvider(worldId byte, characterId uint32, proposedName string) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(characterId))
	value := &statusEvent[statusEventRequestAgreementBody]{
		WorldId: worldId,
		GuildId: 0,
		Type:    StatusEventTypeRequestAgreement,
		Body: statusEventRequestAgreementBody{
			ActorId:      characterId,
			ProposedName: proposedName,
		},
	}
	return producer.SingleMessageProvider(key, value)
}

func statusEventCreatedProvider(worldId byte, guildId uint32) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(guildId))
	value := &statusEvent[statusEventCreatedBody]{
		WorldId: worldId,
		GuildId: guildId,
		Type:    StatusEventTypeCreated,
		Body:    statusEventCreatedBody{},
	}
	return producer.SingleMessageProvider(key, value)
}

func statusEventDisbandedProvider(worldId byte, guildId uint32, members []uint32) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(guildId))
	value := &statusEvent[statusEventDisbandedBody]{
		WorldId: worldId,
		GuildId: guildId,
		Type:    StatusEventTypeDisbanded,
		Body: statusEventDisbandedBody{
			Members: members,
		},
	}
	return producer.SingleMessageProvider(key, value)
}

func statusEventEmblemUpdatedProvider(worldId byte, guildId uint32, logo uint16, logoColor byte, logoBackground uint16, logoBackgroundColor byte) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(guildId))
	value := &statusEvent[statusEventEmblemUpdatedBody]{
		WorldId: worldId,
		GuildId: guildId,
		Type:    StatusEventTypeEmblemUpdated,
		Body: statusEventEmblemUpdatedBody{
			Logo:                logo,
			LogoColor:           logoColor,
			LogoBackground:      logoBackground,
			LogoBackgroundColor: logoBackgroundColor,
		},
	}
	return producer.SingleMessageProvider(key, value)
}

func statusEventMemberStatusUpdatedProvider(worldId byte, guildId uint32, characterId uint32, online bool) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(guildId))
	value := &statusEvent[statusEventMemberStatusUpdatedBody]{
		WorldId: worldId,
		GuildId: guildId,
		Type:    StatusEventTypeMemberStatusUpdated,
		Body: statusEventMemberStatusUpdatedBody{
			CharacterId: characterId,
			Online:      online,
		},
	}
	return producer.SingleMessageProvider(key, value)
}

func statusEventMemberTitleUpdatedProvider(worldId byte, guildId uint32, characterId uint32, title byte) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(guildId))
	value := &statusEvent[statusEventMemberTitleUpdatedBody]{
		WorldId: worldId,
		GuildId: guildId,
		Type:    StatusEventTypeMemberTitleUpdated,
		Body: statusEventMemberTitleUpdatedBody{
			CharacterId: characterId,
			Title:       title,
		},
	}
	return producer.SingleMessageProvider(key, value)
}

func statusEventNoticeUpdatedProvider(worldId byte, guildId uint32, notice string) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(guildId))
	value := &statusEvent[statusEventNoticeUpdatedBody]{
		WorldId: worldId,
		GuildId: guildId,
		Type:    StatusEventTypeNoticeUpdated,
		Body: statusEventNoticeUpdatedBody{
			Notice: notice,
		},
	}
	return producer.SingleMessageProvider(key, value)
}

func statusEventCapacityUpdatedProvider(worldId byte, guildId uint32, capacity uint32) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(guildId))
	value := &statusEvent[statusEventCapacityUpdatedBody]{
		WorldId: worldId,
		GuildId: guildId,
		Type:    StatusEventTypeCapacityUpdated,
		Body: statusEventCapacityUpdatedBody{
			Capacity: capacity,
		},
	}
	return producer.SingleMessageProvider(key, value)
}

func statusEventMemberLeftProvider(worldId byte, guildId uint32, characterId uint32, force bool) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(guildId))
	value := &statusEvent[statusEventMemberLeftBody]{
		WorldId: worldId,
		GuildId: guildId,
		Type:    StatusEventTypeMemberLeft,
		Body: statusEventMemberLeftBody{
			CharacterId: characterId,
			Force:       force,
		},
	}
	return producer.SingleMessageProvider(key, value)
}

func statusEventMemberJoinedProvider(worldId byte, guildId uint32, characterId uint32, name string, jobId uint16, level byte, title byte, allianceTitle byte) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(guildId))
	value := &statusEvent[statusEventMemberJoinedBody]{
		WorldId: worldId,
		GuildId: guildId,
		Type:    StatusEventTypeMemberJoined,
		Body: statusEventMemberJoinedBody{
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

func statusEventTitlesUpdatedProvider(worldId byte, guildId uint32, titles []string) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(guildId))
	value := &statusEvent[statusEventTitlesUpdatedBody]{
		WorldId: worldId,
		GuildId: guildId,
		Type:    StatusEventTypeTitlesUpdated,
		Body: statusEventTitlesUpdatedBody{
			Titles: titles,
		},
	}
	return producer.SingleMessageProvider(key, value)
}

func statusEventErrorProvider(worldId byte, characterId uint32, error string) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(characterId))
	value := &statusEvent[statusEventErrorBody]{
		WorldId: worldId,
		GuildId: 0,
		Type:    StatusEventTypeError,
		Body: statusEventErrorBody{
			ActorId: characterId,
			Error:   error,
		},
	}
	return producer.SingleMessageProvider(key, value)
}
