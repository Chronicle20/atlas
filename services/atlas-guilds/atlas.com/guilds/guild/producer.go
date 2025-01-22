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
