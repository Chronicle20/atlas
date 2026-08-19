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
	// TransactionId is the idempotency key of the LAST Extend call applied to
	// this row (task-240 task 24c): atlas-cashshop's purchase transaction
	// mints an EXTEND_EQUIP_SLOT outbox command carrying its own transaction
	// id, and the outbox is at-least-once, so a redelivery of that command
	// must not add days a second time. A repeat call carrying the SAME
	// TransactionId already stored here is a no-op that returns the current
	// ExpiresAt unchanged. The zero UUID means "no dedupe key recorded" and
	// never matches a real transaction id, preserving the original
	// (pre-task-24c) behavior for any caller that does not supply one.
	TransactionId uuid.UUID `gorm:"not null;default:'00000000-0000-0000-0000-000000000000'"`
	CreatedAt     time.Time `gorm:"not null"`
	UpdatedAt     time.Time `gorm:"not null"`
}

func (e Entity) TableName() string { return "character_equip_slot_extensions" }

func Migration(db *gorm.DB) error {
	return db.AutoMigrate(&Entity{})
}
