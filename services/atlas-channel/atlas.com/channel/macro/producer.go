package macro

import (
	"github.com/Chronicle20/atlas-kafka/producer"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/segmentio/kafka-go"
)

func updateCommandProvider(characterId uint32, macros []Model) model.Provider[[]kafka.Message] {
	ms := make([]MacroBody, 0)
	for _, macro := range macros {
		ms = append(ms, MacroBody{
			Id:       macro.Id(),
			Name:     macro.Name(),
			Shout:    macro.Shout(),
			SkillId1: uint32(macro.SkillId1()),
			SkillId2: uint32(macro.SkillId2()),
			SkillId3: uint32(macro.SkillId3()),
		})
	}

	key := producer.CreateKey(int(characterId))
	value := &Command[UpdateCommandBody]{
		CharacterId: characterId,
		Type:        CommandTypeUpdate,
		Body: UpdateCommandBody{
			Macros: ms,
		},
	}
	return producer.SingleMessageProvider(key, value)
}
