package information

import (
	"atlas-configurations/configuration/continent"
	"atlas-configurations/configuration/monster"
	"github.com/google/uuid"
)

type RestModel struct {
	TenantId   uuid.UUID             `json:"tenantId"`
	Continents []continent.RestModel `json:"continents"`
	Monsters   []monster.RestModel   `json:"monsters"`
}
