package reagent

import (
	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/Chronicle20/atlas/libs/atlas-constants/item"
)

func Migration(db *gorm.DB) error {
	return db.AutoMigrate(&entity{})
}

// entity is one row of the reagents table. The business identity of a reagent
// is (tenant_id, reagent_item_id) — the same gem is retunable per tenant — so
// the primary key is a surrogate uuid and the business identity is enforced by
// a unique index.
type entity struct {
	Id            uuid.UUID `gorm:"column:id;type:uuid;primaryKey"`
	TenantId      uuid.UUID `gorm:"not null;uniqueIndex:idx_reagents_tenant_item,priority:1"`
	ReagentItemId uint32    `gorm:"not null;uniqueIndex:idx_reagents_tenant_item,priority:2"`
	Stat          string    `gorm:"not null"`
	// Value is a signed delta; incReqLevel rows are negative.
	Value int16 `gorm:"not null"`
}

func (e entity) TableName() string {
	return "reagents"
}

// Make builds a Model from e, the canonical entity-to-model conversion
// (DOM-02).
func Make(e entity) (Model, error) {
	return NewBuilder(e.TenantId, item.Id(e.ReagentItemId)).
		SetStat(e.Stat).
		SetValue(e.Value).
		Build()
}

// ToEntity builds the entity for m, the canonical model-to-entity conversion
// (DOM-03).
func (m Model) ToEntity() entity {
	return entity{
		TenantId:      m.TenantId(),
		ReagentItemId: uint32(m.ReagentItemId()),
		Stat:          m.Stat(),
		Value:         m.Value(),
	}
}
