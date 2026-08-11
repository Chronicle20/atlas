package cashshop

import (
	"atlas-configurations/tenants/cashshop/commodities"
	"atlas-configurations/tenants/cashshop/surprise"
)

type RestModel struct {
	Commodities commodities.RestModel `json:"commodities"`
	Surprise    surprise.RestModel    `json:"surprise"`
}
