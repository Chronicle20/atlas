package crystalband

import (
	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/Chronicle20/atlas/libs/atlas-constants/item"
)

func Migration(db *gorm.DB) error {
	return db.AutoMigrate(&entity{})
}

// entity is one row of the crystal_bands table: the client's
// CItemMakerInfo::Load_MonsterCrystalLevel record (lvMin, lvMax, itemId) from
// Item.wz/Etc/0426.img, keyed per tenant so the band table is retunable
// without a migration (design §4.2.4). The business identity is
// (tenant_id, min_level); the primary key is a surrogate uuid and the
// business identity is enforced by a unique index.
type entity struct {
	Id            uuid.UUID `gorm:"column:id;type:uuid;primaryKey"`
	TenantId      uuid.UUID `gorm:"not null;uniqueIndex:idx_crystal_bands_tenant_min,priority:1"`
	MinLevel      uint32    `gorm:"not null;uniqueIndex:idx_crystal_bands_tenant_min,priority:2"`
	MaxLevel      uint32    `gorm:"not null"`
	CrystalItemId uint32    `gorm:"not null"`
	// Count is an Atlas product decision, NOT a derived value: the client
	// record decompiled at CItemMakerInfo::Load_MonsterCrystalLevel is
	// exactly the 3-DWORD triple (lvMin, lvMax, itemId), and the loader reads
	// no quantity field. Seeded to 1 for all nine bands; retunable via seed
	// without a migration.
	Count uint32 `gorm:"not null"`
}

func (e entity) TableName() string {
	return "crystal_bands"
}

// Make builds a Model from e, the canonical entity-to-model conversion
// (DOM-02).
func Make(e entity) (Model, error) {
	return NewBuilder(e.TenantId).
		SetMinLevel(e.MinLevel).
		SetMaxLevel(e.MaxLevel).
		SetCrystalItemId(item.Id(e.CrystalItemId)).
		SetCount(e.Count).
		Build()
}

// ToEntity builds the entity for m, the canonical model-to-entity conversion
// (DOM-03).
func (m Model) ToEntity() entity {
	return entity{
		TenantId:      m.TenantId(),
		MinLevel:      m.MinLevel(),
		MaxLevel:      m.MaxLevel(),
		CrystalItemId: uint32(m.CrystalItemId()),
		Count:         m.Count(),
	}
}
