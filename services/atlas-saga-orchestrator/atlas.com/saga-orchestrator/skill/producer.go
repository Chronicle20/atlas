package skill

import (
	skill2 "atlas-saga-orchestrator/kafka/message/skill"
	"github.com/Chronicle20/atlas-constants/world"
	"github.com/Chronicle20/atlas-kafka/producer"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/google/uuid"
	"github.com/segmentio/kafka-go"
	"time"
)

func RequestCreateProvider(transactionId uuid.UUID, worldId world.Id, characterId uint32, id uint32, level byte, masterLevel byte, expiration time.Time) model.Provider[[]kafka.Message] {
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

func RequestUpdateProvider(transactionId uuid.UUID, worldId world.Id, characterId uint32, id uint32, level byte, masterLevel byte, expiration time.Time) model.Provider[[]kafka.Message] {
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
