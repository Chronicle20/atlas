package equipable

import (
	"github.com/google/uuid"
	"gorm.io/gorm"
)

func Migration(db *gorm.DB) error {
	return db.AutoMigrate(&entity{})
}

type entity struct {
	TenantId    uuid.UUID `gorm:"not null"`
	ID          uint32    `gorm:"primaryKey;autoIncrement;not null"`
	InventoryId uint32    `gorm:"not null"`
	ItemId      uint32    `gorm:"not null"`
	Slot        int16     `gorm:"not null"`
	ReferenceId uint32    `gorm:"not null"`
}

func (e entity) TableName() string {
	return "equipables"
}
