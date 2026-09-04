package reagent

import (
	"github.com/google/uuid"

	"github.com/Chronicle20/atlas/libs/atlas-constants/item"
)

// Model is one gem/reagent's contribution to a crafted equip, as the client's
// CItemMakerInfo::Load_GemEffect reads it out of Item.wz/Etc/0425.img. Each gem
// declares exactly one field, so the model carries exactly one (stat, value)
// pair rather than a collection.
//
// Model is immutable: build it through Builder, read it through the accessors.
type Model struct {
	tenantId      uuid.UUID
	reagentItemId item.Id
	stat          string
	value         int16
}

func (m Model) TenantId() uuid.UUID {
	return m.tenantId
}

// ReagentItemId is the full item id of the gem (4250000 - 4251402), never a
// truncated form of it.
func (m Model) ReagentItemId() item.Id {
	return m.reagentItemId
}

// Stat is the affected equip stat, spelled exactly as the archive spells it
// (case-sensitive) — one of ValidStats.
func (m Model) Stat() string {
	return m.stat
}

// Value is the delta applied to Stat. It is signed: incReqLevel is negative for
// every one of its rows.
func (m Model) Value() int16 {
	return m.value
}
