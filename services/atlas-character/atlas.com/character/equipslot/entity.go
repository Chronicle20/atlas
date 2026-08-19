package equipslot

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// Entity records one purchased equipped-inventory slot extension.
//
// SlotIndex is the client-facing slot the extension unlocks; its value comes
// from the GMS v95.1 IDB (see docs/tasks/task-240-cash-shop-stub-operations/
// derivation-equip-slot.md E1) and is NOT invented. Persistence stores the
// version-agnostic Atlas canonical position, not the version-dependent wire
// or client-body-part encodings.
type Entity struct {
	Id          uuid.UUID `gorm:"primaryKey;not null"`
	TenantId    uuid.UUID `gorm:"not null;uniqueIndex:idx_equipslot_unique"`
	CharacterId uint32    `gorm:"not null;uniqueIndex:idx_equipslot_unique"`
	SlotIndex   int16     `gorm:"not null;uniqueIndex:idx_equipslot_unique"`
	ExpiresAt   time.Time `gorm:"not null"`
	CreatedAt   time.Time `gorm:"not null"`
	UpdatedAt   time.Time `gorm:"not null"`
}

func (e Entity) TableName() string { return "character_equip_slot_extensions" }

func Migration(db *gorm.DB) error {
	return db.AutoMigrate(&Entity{})
}
