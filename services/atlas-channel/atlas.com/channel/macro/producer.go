package macro

import (
	macro2 "atlas-channel/kafka/message/macro"
	"github.com/Chronicle20/atlas-kafka/producer"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/segmentio/kafka-go"
)

func UpdateCommandProvider(characterId uint32, macros []Model) model.Provider[[]kafka.Message] {
	ms := make([]macro2.MacroBody, 0)
	for _, macro := range macros {
		ms = append(ms, macro2.MacroBody{
			Id:       macro.Id(),
			Name:     macro.Name(),
			Shout:    macro.Shout(),
			SkillId1: uint32(macro.SkillId1()),
			SkillId2: uint32(macro.SkillId2()),
			SkillId3: uint32(macro.SkillId3()),
		})
	}

	key := producer.CreateKey(int(characterId))
	value := &macro2.Command[macro2.UpdateCommandBody]{
		CharacterId: characterId,
		Type:        macro2.CommandTypeUpdate,
		Body: macro2.UpdateCommandBody{
			Macros: ms,
		},
	}
	return producer.SingleMessageProvider(key, value)
}
