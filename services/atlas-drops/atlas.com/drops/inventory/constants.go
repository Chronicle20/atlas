package inventory

const (
	TypeValueEquip int8 = 1
	TypeValueUse   int8 = 2
	TypeValueSetup int8 = 3
	TypeValueETC   int8 = 4
	TypeValueCash  int8 = 5
)

func GetInventoryType(itemId uint32) (int8, bool) {
	t := int8(itemId / 1000000)
	if t >= 1 && t <= 5 {
		return t, true
	}
	return 0, false
}
