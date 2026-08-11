package cashshop

import (
	"atlas-configurations/templates/cashshop/commodities"
	"atlas-configurations/templates/cashshop/surprise"
)

type RestModel struct {
	Commodities commodities.RestModel `json:"commodities"`
	Surprise    surprise.RestModel    `json:"surprise"`
}
