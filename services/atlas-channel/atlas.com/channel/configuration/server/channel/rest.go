package channel

import (
	"atlas-channel/configuration/handler"
	"atlas-channel/configuration/version"
	"atlas-channel/configuration/world"
	"atlas-channel/configuration/writer"
	"github.com/google/uuid"
)

type RestModel struct {
	TenantId uuid.UUID           `json:"tenantId"`
	Region   string              `json:"region"`
	Version  version.RestModel   `json:"version"`
	Worlds   []world.RestModel   `json:"worlds"`
	Handlers []handler.RestModel `json:"handlers"`
	Writers  []writer.RestModel  `json:"writers"`
}
