package slot

import (
	"atlas-query-aggregator/asset"

	"github.com/Chronicle20/atlas/libs/atlas-constants/inventory/slot"
)

type Model struct {
	Position      slot.Position
	Equipable     *asset.Model
	CashEquipable *asset.Model
}
