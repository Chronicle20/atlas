package macro

import (
	macro2 "atlas-skills/kafka/message/macro"

	"github.com/Chronicle20/atlas-constants/world"
	"github.com/Chronicle20/atlas-kafka/producer"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/google/uuid"
	"github.com/segmentio/kafka-go"
)

// statusEventUpdatedProvider creates a provider for a macro updated status event
func statusEventUpdatedProvider(transactionId uuid.UUID, worldId world.Id, characterId uint32, macros []Model) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(characterId))

	// Convert domain models to MacroBody structs
	macroBodies := make([]macro2.MacroBody, 0, len(macros))
	for _, m := range macros {
		macroBodies = append(macroBodies, macro2.MacroBody{
			Id:       m.Id(),
			Name:     m.Name(),
			Shout:    m.Shout(),
			SkillId1: uint32(m.SkillId1()),
			SkillId2: uint32(m.SkillId2()),
			SkillId3: uint32(m.SkillId3()),
		})
	}

	value := &macro2.StatusEvent[macro2.StatusEventUpdatedBody]{
		TransactionId: transactionId,
		WorldId:       worldId,
		CharacterId:   characterId,
		Type:          macro2.StatusEventTypeUpdated,
		Body: macro2.StatusEventUpdatedBody{
			Macros: macroBodies,
		},
	}

	return producer.SingleMessageProvider(key, value)
}
