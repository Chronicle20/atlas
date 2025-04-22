package skill

import (
	"atlas-channel/kafka/message/skill"
	"github.com/Chronicle20/atlas-kafka/producer"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/segmentio/kafka-go"
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
