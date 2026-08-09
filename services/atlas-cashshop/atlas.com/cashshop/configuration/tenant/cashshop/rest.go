package cashshop

import (
	"atlas-cashshop/configuration/tenant/cashshop/commodities"
	"atlas-cashshop/configuration/tenant/cashshop/surprise"
)

type RestModel struct {
	Commodities commodities.RestModel `json:"commodities"`
	Surprise    surprise.RestModel    `json:"surprise"`
}
