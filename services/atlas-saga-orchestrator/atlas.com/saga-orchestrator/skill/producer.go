package skill

import (
	skill2 "atlas-saga-orchestrator/kafka/message/skill"
	"time"

	"github.com/google/uuid"
	"github.com/segmentio/kafka-go"

	"github.com/Chronicle20/atlas/libs/atlas-constants/job"
	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/producer"
	"github.com/Chronicle20/atlas/libs/atlas-model/model"
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

// RequestDeleteProvider emits the saga-correlated REQUEST_DELETE command on
// COMMAND_TOPIC_SKILL (plan Phase 5 / Phase 6).
func RequestDeleteProvider(transactionId uuid.UUID, worldId world.Id, characterId uint32, skillId uint32) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(characterId))
	value := &skill2.Command[skill2.RequestDeleteBody]{
		TransactionId: transactionId,
		WorldId:       worldId,
		CharacterId:   characterId,
		Type:          skill2.CommandTypeRequestDelete,
		Body: skill2.RequestDeleteBody{
			SkillId: skillId,
		},
	}
	return producer.SingleMessageProvider(key, value)
}

// TransferSpProvider emits the saga-correlated TRANSFER_SP command consumed by
// atlas-skills (SP Reset items 5050001-5050004, task-126).
func TransferSpProvider(transactionId uuid.UUID, worldId world.Id, characterId uint32, jobId job.Id, fromSkillId uint32, toSkillId uint32, itemTier byte, targetMaxLevel byte) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(characterId))
	value := &skill2.Command[skill2.TransferSpBody]{
		TransactionId: transactionId,
		WorldId:       worldId,
		CharacterId:   characterId,
		Type:          skill2.CommandTypeTransferSp,
		Body: skill2.TransferSpBody{
			JobId:          jobId,
			FromSkillId:    fromSkillId,
			ToSkillId:      toSkillId,
			ItemTier:       itemTier,
			TargetMaxLevel: targetMaxLevel,
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
