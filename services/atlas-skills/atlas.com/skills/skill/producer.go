package skill

import (
	skill2 "atlas-skills/kafka/message/skill"
	"github.com/Chronicle20/atlas-kafka/producer"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/segmentio/kafka-go"
	"time"
)

func createCommandProvider(characterId uint32, id uint32, level byte, masterLevel byte, expiration time.Time) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(characterId))
	value := &skill2.Command[skill2.RequestCreateBody]{
		CharacterId: characterId,
		Type:        skill2.CommandTypeRequestCreate,
		Body: skill2.RequestCreateBody{
			SkillId:     id,
			Level:       level,
			MasterLevel: masterLevel,
			Expiration:  expiration,
		},
	}
	return producer.SingleMessageProvider(key, value)
}

func updateCommandProvider(characterId uint32, id uint32, level byte, masterLevel byte, expiration time.Time) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(characterId))
	value := &skill2.Command[skill2.RequestUpdateBody]{
		CharacterId: characterId,
		Type:        skill2.CommandTypeRequestUpdate,
		Body: skill2.RequestUpdateBody{
			SkillId:     id,
			Level:       level,
			MasterLevel: masterLevel,
			Expiration:  expiration,
		},
	}
	return producer.SingleMessageProvider(key, value)
}

func statusEventCreatedProvider(characterId uint32, id uint32, level byte, masterLevel byte, expiration time.Time) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(characterId))
	value := &skill2.StatusEvent[skill2.StatusEventCreatedBody]{
		CharacterId: characterId,
		SkillId:     id,
		Type:        skill2.StatusEventTypeCreated,
		Body: skill2.StatusEventCreatedBody{
			Level:       level,
			MasterLevel: masterLevel,
			Expiration:  expiration,
		},
	}
	return producer.SingleMessageProvider(key, value)
}

func statusEventUpdatedProvider(characterId uint32, id uint32, level byte, masterLevel byte, expiration time.Time) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(characterId))
	value := &skill2.StatusEvent[skill2.StatusEventUpdatedBody]{
		CharacterId: characterId,
		SkillId:     id,
		Type:        skill2.StatusEventTypeUpdated,
		Body: skill2.StatusEventUpdatedBody{
			Level:       level,
			MasterLevel: masterLevel,
			Expiration:  expiration,
		},
	}
	return producer.SingleMessageProvider(key, value)
}

func statusEventCooldownAppliedProvider(characterId uint32, id uint32, cooldownExpiresAt time.Time) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(characterId))
	value := &skill2.StatusEvent[skill2.StatusEventCooldownAppliedBody]{
		CharacterId: characterId,
		SkillId:     id,
		Type:        skill2.StatusEventTypeCooldownApplied,
		Body: skill2.StatusEventCooldownAppliedBody{
			CooldownExpiresAt: cooldownExpiresAt,
		},
	}
	return producer.SingleMessageProvider(key, value)
}

func statusEventCooldownExpiredProvider(characterId uint32, id uint32) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(characterId))
	value := &skill2.StatusEvent[skill2.StatusEventCooldownExpiredBody]{
		CharacterId: characterId,
		SkillId:     id,
		Type:        skill2.StatusEventTypeCooldownExpired,
		Body:        skill2.StatusEventCooldownExpiredBody{},
	}
	return producer.SingleMessageProvider(key, value)
}
