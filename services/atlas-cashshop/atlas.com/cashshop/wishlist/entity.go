package wishlist

import (
	"github.com/google/uuid"
	"gorm.io/gorm"
)

func Migration(db *gorm.DB) error {
	return db.AutoMigrate(&Entity{})
}

type Entity struct {
	Id           uuid.UUID `gorm:"primaryKey;type:uuid;default:uuid_generate_v4()"`
	TenantId     uuid.UUID `gorm:"not null"`
	CharacterId  uint32    `gorm:"not null"`
	SerialNumber uint32    `gorm:"not null"`
}

func (e Entity) TableName() string {
	return "wishlist_items"
}

func Make(e Entity) (Model, error) {
	return Model{
		id:           e.Id,
		characterId:  e.CharacterId,
		serialNumber: e.SerialNumber,
	}, nil
}
