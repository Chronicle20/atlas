package inventory

import (
	"github.com/google/uuid"
	"gorm.io/gorm"
)

func Migration(db *gorm.DB) error {
	return db.AutoMigrate(&entity{})
}

type entity struct {
	TenantId      uuid.UUID `gorm:"not null"`
	ID            uint32    `gorm:"primaryKey;autoIncrement;not null"`
	CharacterId   uint32    `gorm:"not null;UNIQUE_INDEX:composite_index;index"`
	InventoryType int8      `gorm:"not null;UNIQUE_INDEX:composite_index;"`
	Capacity      uint32    `gorm:"capacity"`
}

func (e entity) TableName() string {
	return "inventory"
}
