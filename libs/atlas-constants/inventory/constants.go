package inventory

import (
	"math"

	"github.com/Chronicle20/atlas/libs/atlas-constants/item"
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

// Token returns the lowercase API token for an inventory type, suitable for
// JSON responses and query-param values. Unrecognised types return "unknown".
func (t Type) Token() string {
	switch t {
	case TypeValueEquip:
		return "equipment"
	case TypeValueUse:
		return "use"
	case TypeValueSetup:
		return "setup"
	case TypeValueETC:
		return "etc"
	case TypeValueCash:
		return "cash"
	default:
		return "unknown"
	}
}

func TypeFromItemId(itemId item.Id) (Type, bool) {
	t := int8(math.Floor(float64(itemId) / 1000000))
	if t >= 1 && t <= 5 {
		return Type(t), true
	}
	return TypeValueEquip, false
}
