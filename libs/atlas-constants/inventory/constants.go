package inventory

import (
	"github.com/Chronicle20/atlas-constants/item"
	"math"
)

type Type int8

const (
	TypeValueEquip Type = 1
	TypeValueUse   Type = 2
	TypeValueSetup Type = 3
	TypeValueETC   Type = 4
	TypeValueCash  Type = 5
)

var Types = []Type{TypeValueEquip, TypeValueUse, TypeValueSetup, TypeValueETC, TypeValueCash}

func TypeFromItemId(itemId item.Id) (Type, bool) {
	t := int8(math.Floor(float64(itemId) / 1000000))
	if t >= 1 && t <= 5 {
		return Type(t), true
	}
	return TypeValueEquip, false
}
