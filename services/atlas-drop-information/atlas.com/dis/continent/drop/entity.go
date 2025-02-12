package drop

import (
	"github.com/google/uuid"
	"gorm.io/gorm"
)

func Migration(db *gorm.DB) error {
	return db.AutoMigrate(&entity{})
}

type entity struct {
	TenantId        uuid.UUID `gorm:"not null"`
	ID              uint32    `gorm:"primaryKey;autoIncrement;not null"`
	ContinentId     int32     `gorm:"not null;default=-1"`
	ItemId          uint32    `gorm:"not null;default=0"`
	MinimumQuantity uint32    `gorm:"not null;default=0"`
	MaximumQuantity uint32    `gorm:"not null;default=0"`
	QuestId         uint32    `gorm:"not null;default=0"`
	Chance          uint32    `gorm:"not null;default=0"`
}

func (e entity) TableName() string {
	return "continent_drops"
}
