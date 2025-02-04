package login

import (
	"atlas-login/configuration/handler"
	"atlas-login/configuration/version"
	"atlas-login/configuration/writer"
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
