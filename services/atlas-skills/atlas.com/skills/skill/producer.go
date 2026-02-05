package skill

import (
	skill2 "atlas-skills/kafka/message/skill"
	"github.com/Chronicle20/atlas-constants/world"
	"github.com/Chronicle20/atlas-kafka/producer"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/google/uuid"
	"github.com/segmentio/kafka-go"
	"time"
)

func createCommandProvider(transactionId uuid.UUID, worldId world.Id, characterId uint32, id uint32, level byte, masterLevel byte, expiration time.Time) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(characterId))
	value := &skill2.Command[skill2.RequestCreateBody]{
		TransactionId: transactionId,
		WorldId:       worldId,
		CharacterId:   characterId,
		Type:          skill2.CommandTypeRequestCreate,
		Body: skill2.RequestCreateBody{
			SkillId:     id,
			Level:       level,
			MasterLevel: masterLevel,
			Expiration:  expiration,
		},
	}
	return producer.SingleMessageProvider(key, value)
}

func updateCommandProvider(transactionId uuid.UUID, worldId world.Id, characterId uint32, id uint32, level byte, masterLevel byte, expiration time.Time) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(characterId))
	value := &skill2.Command[skill2.RequestUpdateBody]{
		TransactionId: transactionId,
		WorldId:       worldId,
		CharacterId:   characterId,
		Type:          skill2.CommandTypeRequestUpdate,
		Body: skill2.RequestUpdateBody{
			SkillId:     id,
			Level:       level,
			MasterLevel: masterLevel,
			Expiration:  expiration,
		},
	}
	return producer.SingleMessageProvider(key, value)
}

func statusEventCreatedProvider(transactionId uuid.UUID, worldId world.Id, characterId uint32, id uint32, level byte, masterLevel byte, expiration time.Time) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(characterId))
	value := &skill2.StatusEvent[skill2.StatusEventCreatedBody]{
		TransactionId: transactionId,
		WorldId:       worldId,
		CharacterId:   characterId,
		SkillId:       id,
		Type:          skill2.StatusEventTypeCreated,
		Body: skill2.StatusEventCreatedBody{
			Level:       level,
			MasterLevel: masterLevel,
			Expiration:  expiration,
		},
	}
	return producer.SingleMessageProvider(key, value)
}

func statusEventUpdatedProvider(transactionId uuid.UUID, worldId world.Id, characterId uint32, id uint32, level byte, masterLevel byte, expiration time.Time) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(characterId))
	value := &skill2.StatusEvent[skill2.StatusEventUpdatedBody]{
		TransactionId: transactionId,
		WorldId:       worldId,
		CharacterId:   characterId,
		SkillId:       id,
		Type:          skill2.StatusEventTypeUpdated,
		Body: skill2.StatusEventUpdatedBody{
			Level:       level,
			MasterLevel: masterLevel,
			Expiration:  expiration,
		},
	}
	return producer.SingleMessageProvider(key, value)
}

func statusEventCooldownAppliedProvider(transactionId uuid.UUID, worldId world.Id, characterId uint32, id uint32, cooldownExpiresAt time.Time) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(characterId))
	value := &skill2.StatusEvent[skill2.StatusEventCooldownAppliedBody]{
		TransactionId: transactionId,
		WorldId:       worldId,
		CharacterId:   characterId,
		SkillId:       id,
		Type:          skill2.StatusEventTypeCooldownApplied,
		Body: skill2.StatusEventCooldownAppliedBody{
			CooldownExpiresAt: cooldownExpiresAt,
		},
	}
	return producer.SingleMessageProvider(key, value)
}

func statusEventCooldownExpiredProvider(transactionId uuid.UUID, worldId world.Id, characterId uint32, id uint32) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(characterId))
	value := &skill2.StatusEvent[skill2.StatusEventCooldownExpiredBody]{
		TransactionId: transactionId,
		WorldId:       worldId,
		CharacterId:   characterId,
		SkillId:       id,
		Type:          skill2.StatusEventTypeCooldownExpired,
		Body:          skill2.StatusEventCooldownExpiredBody{},
	}
	return producer.SingleMessageProvider(key, value)
}
