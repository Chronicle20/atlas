package account

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

func Migration(db *gorm.DB) error {
	return db.AutoMigrate(&Entity{}, &CharacterSlotEntity{})
}

type Entity struct {
	TenantId    uuid.UUID `gorm:"not null"`
	ID          uint32    `gorm:"primaryKey;autoIncrement;not null"`
	Name        string    `gorm:"not null"`
	Password    string    `gorm:"not null"`
	PIN         string
	PIC         string
	BirthDate   uint32
	PinAttempts int  `gorm:"not null;default=0"`
	PicAttempts int  `gorm:"not null;default=0"`
	Gender      byte `gorm:"not null;default=0"`
	TOS         bool `gorm:"not null;default=false"`
	LastLogin   int64
	CreatedAt   time.Time // Automatically managed by GORM for creation time
	UpdatedAt   time.Time // Automatically managed by GORM for update time
}

func (e Entity) TableName() string {
	return "accounts"
}

// CharacterSlotEntity is the per-(tenant, account, world) character-slot
// count (task-246 bug-b-type-must-add-a-slot.md). It cannot live as a column
// on Entity: slots are world-scoped, not account-scoped, so an account with
// characters in several worlds needs one row per world. The absence of a row
// for a given (account, world) means the caller has never had a slot
// increment there yet -- the processor treats that as
// DefaultCharacterSlotsPerWorld, so no backfill migration is needed.
type CharacterSlotEntity struct {
	TenantId  uuid.UUID `gorm:"not null;uniqueIndex:idx_character_slots_tenant_account_world"`
	ID        uint32    `gorm:"primaryKey;autoIncrement;not null"`
	AccountId uint32    `gorm:"not null;uniqueIndex:idx_character_slots_tenant_account_world"`
	WorldId   byte      `gorm:"not null;uniqueIndex:idx_character_slots_tenant_account_world"`
	Slots     int16     `gorm:"not null;default=4"`
	CreatedAt time.Time
	UpdatedAt time.Time
}

func (e CharacterSlotEntity) TableName() string {
	return "account_character_slots"
}
