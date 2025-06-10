package compartment

import (
	"github.com/Chronicle20/atlas-constants/inventory"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

func Migration(db *gorm.DB) error {
	return db.AutoMigrate(&Entity{})
}

type Entity struct {
	TenantId      uuid.UUID      `gorm:"not null"`
	Id            uuid.UUID      `gorm:"primaryKey;type:uuid;"`
	CharacterId   uint32         `gorm:"not null"`
	InventoryType inventory.Type `gorm:"not null"`
	Capacity      uint32         `gorm:"capacity"`
}

func (e Entity) TableName() string {
	return "compartments"
}

func (e *Entity) BeforeCreate(_ *gorm.DB) (err error) {
	if e.Id == uuid.Nil {
		e.Id = uuid.New()
	}
	return
}

func Make(e Entity) (Model, error) {
	return Model{
		id:            e.Id,
		characterId:   e.CharacterId,
		inventoryType: e.InventoryType,
		capacity:      e.Capacity,
	}, nil
}
