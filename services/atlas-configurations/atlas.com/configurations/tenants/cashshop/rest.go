package cashshop

import (
	"atlas-configurations/tenants/cashshop/commodities"
)

type RestModel struct {
	Commodities commodities.RestModel `json:"commodities"`
}
