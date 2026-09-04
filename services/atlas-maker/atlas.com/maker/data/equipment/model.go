package equipment

import "github.com/Chronicle20/atlas/libs/atlas-constants/item"

// Model is an equip template's crafting-relevant attribute, as resolved
// from atlas-data. reqLevel is the minimum equip level a "req equip"
// ingredient (design C-3) must meet before the material is consumable.
type Model struct {
	id       item.Id
	reqLevel uint32
}

func (m Model) Id() item.Id {
	return m.id
}

// ReqLevel is info/reqLevel — the equip's own required level, read here so a
// craft's "req equip" ingredient (an equip supplied as a material) can be
// checked against the recipe's own reqLevel expectation.
func (m Model) ReqLevel() uint32 {
	return m.reqLevel
}
