package cashshop

import (
	"atlas-configurations/templates/cashshop/commodities"
)

type RestModel struct {
	Commodities commodities.RestModel `json:"commodities"`
}
