package skill

import (
	"github.com/Chronicle20/atlas-kafka/producer"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/segmentio/kafka-go"
	"time"
)

func createCommandProvider(characterId uint32, id uint32, level byte, masterLevel byte, expiration time.Time) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(characterId))
	value := &command[requestCreateBody]{
		CharacterId: characterId,
		Type:        CommandTypeRequestCreate,
		Body: requestCreateBody{
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
	value := &command[requestUpdateBody]{
		CharacterId: characterId,
		Type:        CommandTypeRequestUpdate,
		Body: requestUpdateBody{
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
	value := &statusEvent[statusEventCreatedBody]{
		CharacterId: characterId,
		SkillId:     id,
		Type:        StatusEventTypeCreated,
		Body: statusEventCreatedBody{
			Level:       level,
			MasterLevel: masterLevel,
			Expiration:  expiration,
		},
	}
	return producer.SingleMessageProvider(key, value)
}

func statusEventUpdatedProvider(characterId uint32, id uint32, level byte, masterLevel byte, expiration time.Time) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(characterId))
	value := &statusEvent[statusEventUpdatedBody]{
		CharacterId: characterId,
		SkillId:     id,
		Type:        StatusEventTypeUpdated,
		Body: statusEventUpdatedBody{
			Level:       level,
			MasterLevel: masterLevel,
			Expiration:  expiration,
		},
	}
	return producer.SingleMessageProvider(key, value)
}
