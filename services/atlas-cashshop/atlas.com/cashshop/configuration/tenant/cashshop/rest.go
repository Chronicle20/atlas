package cashshop

import (
	"atlas-cashshop/configuration/tenant/cashshop/commodities"
	"atlas-cashshop/configuration/tenant/cashshop/coupons"
)

type RestModel struct {
	Commodities commodities.RestModel `json:"commodities"`
	Coupons     coupons.RestModel     `json:"coupons"`
}
