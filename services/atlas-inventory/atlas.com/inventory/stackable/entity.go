package stackable

import (
	"github.com/google/uuid"
	"gorm.io/gorm"
)

func Migration(db *gorm.DB) error {
	return db.AutoMigrate(&Entity{})
}

type Entity struct {
	TenantId      uuid.UUID `gorm:"not null"`
	Id            uint32    `gorm:"primaryKey;autoIncrement;not null"`
	CompartmentId uuid.UUID `gorm:"not null"`
	Quantity      uint32    `gorm:"not null"`
	OwnerId       uint32    `gorm:"not null"`
	Flag          uint16    `gorm:"not null"`
	Rechargeable  uint64    `gorm:"not null;default=0"`
}

func (e Entity) TableName() string {
	return "stackables"
}

func Make(e Entity) (Model, error) {
	return Model{
		id:           e.Id,
		quantity:     e.Quantity,
		ownerId:      e.OwnerId,
		flag:         e.Flag,
		rechargeable: e.Rechargeable,
	}, nil
}
