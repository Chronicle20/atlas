package characterfactory

import (
	"atlas-character-factory/configuration/template"
	"github.com/google/uuid"
)

type RestModel struct {
	TenantId  uuid.UUID            `json:"tenantId"`
	Templates []template.RestModel `json:"templates"`
}
