package cashshop

import (
	"atlas-cashshop/configuration/tenant/cashshop/commodities"
)

type RestModel struct {
	Commodities commodities.RestModel `json:"commodities"`
}
