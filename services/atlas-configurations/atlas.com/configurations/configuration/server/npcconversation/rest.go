package npcconversation

import (
	"atlas-configurations/configuration/script/npc"
	"atlas-configurations/configuration/version"
	"github.com/google/uuid"
)

type RestModel struct {
	TenantId uuid.UUID         `json:"tenantId"`
	Region   string            `json:"region"`
	Version  version.RestModel `json:"version"`
	Scripts  []npc.RestModel   `json:"scripts"`
}
