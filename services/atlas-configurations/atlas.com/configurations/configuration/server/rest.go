package server

import (
	"atlas-configurations/configuration/handler"
	"atlas-configurations/configuration/version"
	"atlas-configurations/configuration/world"
	"atlas-configurations/configuration/writer"
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
