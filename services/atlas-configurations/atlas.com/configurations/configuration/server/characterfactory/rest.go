package characterfactory

import (
	"atlas-configurations/configuration/template"
	"github.com/google/uuid"
)

type RestModel struct {
	TenantId  uuid.UUID            `json:"tenantId"`
	Templates []template.RestModel `json:"templates"`
}
