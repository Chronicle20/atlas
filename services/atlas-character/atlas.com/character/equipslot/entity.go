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

// Make converts a persisted Entity into its domain Model. TransactionId is a
// write-path idempotency key (see the field doc above) and is not part of
// the domain Model -- it never leaves the persistence layer.
func Make(e Entity) (Model, error) {
	return NewBuilder().
		SetId(e.Id).
		SetTenantId(e.TenantId).
		SetCharacterId(e.CharacterId).
		SetSlotIndex(e.SlotIndex).
		SetExpiresAt(e.ExpiresAt).
		Build()
}

// ToEntity is the inverse of Make for the fields the Model owns. It leaves
// TransactionId at its zero value: Model does not carry that write-path
// dedupe key, and Extend (administrator.go) sets it directly when persisting
// a purchase, so this method is not on that path.
func (m Model) ToEntity() Entity {
	return Entity{
		Id:          m.id,
		TenantId:    m.tenantId,
		CharacterId: m.characterId,
		SlotIndex:   m.slotIndex,
		ExpiresAt:   m.expiresAt,
	}
}
