package login

import (
	"atlas-configurations/configuration/handler"
	"atlas-configurations/configuration/version"
	"atlas-configurations/configuration/writer"
	"github.com/google/uuid"
)

type RestModel struct {
	TenantId uuid.UUID           `json:"tenantId"`
	Region   string              `json:"region"`
	Port     string              `json:"port"`
	Version  version.RestModel   `json:"version"`
	UsesPIN  bool                `json:"usesPin"`
	Handlers []handler.RestModel `json:"handlers"`
	Writers  []writer.RestModel  `json:"writers"`
}
