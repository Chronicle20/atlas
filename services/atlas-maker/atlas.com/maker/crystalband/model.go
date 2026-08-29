package crystalband

import (
	"github.com/google/uuid"

	"github.com/Chronicle20/atlas/libs/atlas-constants/item"
)

// Model is one level band of the client's monster-crystal disassembly table,
// as CItemMakerInfo::Load_MonsterCrystalLevel reads it from
// Item.wz/Etc/0426.img/<crystalItemId>/info (lvMin, lvMax). Count carries no
// client-data backing — see the entity field comment.
//
// Model is immutable: build it through Builder, read it through the
// accessors.
type Model struct {
	tenantId      uuid.UUID
	minLevel      uint32
	maxLevel      uint32
	crystalItemId item.Id
	count         uint32
}

func (m Model) TenantId() uuid.UUID {
	return m.tenantId
}

// MinLevel is the band's inclusive lower reqLevel bound.
func (m Model) MinLevel() uint32 {
	return m.minLevel
}

// MaxLevel is the band's inclusive upper reqLevel bound.
func (m Model) MaxLevel() uint32 {
	return m.maxLevel
}

// CrystalItemId is the full item id of the monster crystal this band yields.
func (m Model) CrystalItemId() item.Id {
	return m.crystalItemId
}

// Count is the quantity awarded for this band. It is an Atlas product
// decision, not client-derived data — see the entity field comment.
func (m Model) Count() uint32 {
	return m.count
}

// Contains reports whether reqLevel falls within [MinLevel, MaxLevel], both
// ends inclusive, matching the derived table's contiguous band boundaries.
func (m Model) Contains(reqLevel uint32) bool {
	return reqLevel >= m.minLevel && reqLevel <= m.maxLevel
}
