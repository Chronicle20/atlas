package information

import (
	"atlas-drops-information/configuration/continent"
	"atlas-drops-information/configuration/monster"
	"github.com/google/uuid"
)

type RestModel struct {
	TenantId   uuid.UUID             `json:"tenantId"`
	Continents []continent.RestModel `json:"continents"`
	Monsters   []monster.RestModel   `json:"monsters"`
}
