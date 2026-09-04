package saga

import (
	"atlas-npc-conversations/validation"

	sharedsaga "github.com/Chronicle20/atlas/libs/atlas-saga"
)

// ValidateCharacterStatePayload uses the NPC service's validation.ConditionInput.
// This is NPC-specific and wraps the shared ValidationConditionInput with the local type.
type ValidateCharacterStatePayload struct {
	CharacterId uint32                      `json:"characterId"`
	Conditions  []validation.ConditionInput `json:"conditions"`
}

// ToSharedPayload converts to the shared saga payload type
func (p ValidateCharacterStatePayload) ToSharedPayload() sharedsaga.ValidateCharacterStatePayload {
	conditions := make([]sharedsaga.ValidationConditionInput, len(p.Conditions))
	for i, c := range p.Conditions {
		conditions[i] = sharedsaga.ValidationConditionInput{
			Type:            c.Type,
			Operator:        c.Operator,
			Value:           c.Value,
			ReferenceId:     c.ReferenceId,
			Step:            c.Step,
			WorldId:         c.WorldId,
			ChannelId:       c.ChannelId,
			IncludeEquipped: c.IncludeEquipped,
		}
	}
	return sharedsaga.ValidateCharacterStatePayload{
		CharacterId: p.CharacterId,
		Conditions:  conditions,
	}
}
