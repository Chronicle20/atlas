package npcconversation

import (
	"atlas-npc-conversations/configuration/script/npc"
	"atlas-npc-conversations/configuration/version"
	"github.com/google/uuid"
)

type RestModel struct {
	TenantId uuid.UUID         `json:"tenantId"`
	Region   string            `json:"region"`
	Version  version.RestModel `json:"version"`
	Scripts  []npc.RestModel   `json:"scripts"`
}
