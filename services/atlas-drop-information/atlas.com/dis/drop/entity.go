package drop

import "gorm.io/gorm"

func Migration(db *gorm.DB) error {
	return db.AutoMigrate(&entity{})
}

type entity struct {
	ID              uint32 `gorm:"primaryKey;autoIncrement;not null"`
	MonsterId       uint32 `gorm:"not null;default=0"`
	ItemId          uint32 `gorm:"not null;default=0"`
	MinimumQuantity uint32 `gorm:"not null;default=0"`
	MaximumQuantity uint32 `gorm:"not null;default=0"`
	Chance          uint32 `gorm:"not null;default=0"`
}

func (e entity) TableName() string {
	return "monster_drops"
}
