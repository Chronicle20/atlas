package purchaserecord

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

func Migration(db *gorm.DB) error {
	return db.AutoMigrate(&entity{})
}

// entity is the durable answer to "has this account ever bought serial X".
// It exists because cash_assets is soft-deleted on withdrawal and on rebate,
// so live locker contents cannot answer the question -- and FR-REC-2 is
// explicit that a consumed or discarded item still counts as purchased.
// A rebate does NOT remove a record: "purchased" is a historical fact.
type entity struct {
	Id           uuid.UUID `gorm:"primaryKey;not null"`
	TenantId     uuid.UUID `gorm:"not null;uniqueIndex:idx_purchase_record_unique"`
	AccountId    uint32    `gorm:"not null;uniqueIndex:idx_purchase_record_unique"`
	SerialNumber uint32    `gorm:"not null;uniqueIndex:idx_purchase_record_unique"`
	Count        uint32    `gorm:"not null"`
	FirstAt      time.Time `gorm:"not null"`
	LastAt       time.Time `gorm:"not null"`
}

func (e entity) TableName() string { return "cash_purchase_records" }
