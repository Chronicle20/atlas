package skill

import (
	"atlas-channel/kafka/message/skill"

	"github.com/segmentio/kafka-go"

	"github.com/google/uuid"

	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/producer"
	"github.com/Chronicle20/atlas/libs/atlas-model/model"
)

func SetCooldownCommandProvider(characterId uint32, id uint32, cooldown uint32) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(characterId))
	value := &skill.Command[skill.SetCooldownBody]{
		CharacterId: characterId,
		Type:        skill.CommandTypeSetCooldown,
		Body: skill.SetCooldownBody{
			SkillId:  id,
			Cooldown: cooldown,
		},
	}
	return producer.SingleMessageProvider(key, value)
}

func ResetCooldownsCommandProvider(transactionId uuid.UUID, worldId world.Id, characterId uint32, exceptSkillIds []uint32, sourceSkillId uint32) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(characterId))
	value := &skill.Command[skill.ResetCooldownsBody]{
		TransactionId: transactionId,
		WorldId:       worldId,
		CharacterId:   characterId,
		Type:          skill.CommandTypeResetCooldowns,
		Body: skill.ResetCooldownsBody{
			ExceptSkillIds: exceptSkillIds,
			SourceSkillId:  sourceSkillId,
		},
	}
	return producer.SingleMessageProvider(key, value)
}
