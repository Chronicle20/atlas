package item

import (
	"github.com/google/uuid"
	"gorm.io/gorm"
)

func Migration(db *gorm.DB) error {
	return db.AutoMigrate(&entity{})
}

type entity struct {
	TenantId   uuid.UUID `gorm:"not null"`
	ID         uint32    `gorm:"primaryKey;autoIncrement;not null"`
	GachaponId string    `gorm:"not null;index:idx_gachapon_items_tier"`
	ItemId     uint32    `gorm:"not null"`
	Quantity   uint32    `gorm:"not null;default:1"`
	Tier       string    `gorm:"not null;index:idx_gachapon_items_tier"`
}

func (e entity) TableName() string {
	return "gachapon_items"
}
