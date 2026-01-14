package stackable

import (
	"gorm.io/gorm"
)

// Entity stores stackable item data locally
// This is used for consumable, setup, and etc items
type Entity struct {
	AssetId  uint32 `gorm:"primaryKey"`
	Quantity uint32 `gorm:"not null;default:1"`
	OwnerId  uint32 `gorm:"not null;default:0"`
	Flag     uint16 `gorm:"not null;default:0"`
}

func (e Entity) TableName() string {
	return "storage_stackables"
}

func Migration(db *gorm.DB) error {
	return db.AutoMigrate(&Entity{})
}

// Make converts an Entity to a Model.
// Uses MustBuild since entities from database are trusted.
func Make(e Entity) Model {
	return NewModelBuilder().
		SetAssetId(e.AssetId).
		SetQuantity(e.Quantity).
		SetOwnerId(e.OwnerId).
		SetFlag(e.Flag).
		MustBuild()
}
