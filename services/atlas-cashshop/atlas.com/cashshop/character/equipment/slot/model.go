package slot

import (
	"atlas-cashshop/asset"
	"github.com/Chronicle20/atlas-constants/inventory/slot"
)

type Model struct {
	Position      slot.Position
	Equipable     *asset.Model[asset.EquipableReferenceData]
	CashEquipable *asset.Model[asset.CashEquipableReferenceData]
}
