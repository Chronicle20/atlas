package inventory

import (
	"github.com/google/uuid"
	"gorm.io/gorm"
)

func Migration(db *gorm.DB) error {
	return db.AutoMigrate(&Entity{})
}

type Entity struct {
	Id            uuid.UUID `gorm:"primaryKey;type:uuid;default:uuid_generate_v4()"`
	TenantId      uuid.UUID `gorm:"not null"`
	CharacterId   uint32    `gorm:"not null"`
	InventoryType byte      `gorm:"not null"`
	CashId        uint64    `gorm:"not null"`
}

func (e Entity) TableName() string {
	return "inventories"
}

func Make(e Entity) (Model, error) {
	return Model{
		id:            e.Id,
		characterId:   e.CharacterId,
		inventoryType: e.InventoryType,
		cashId:        e.CashId,
	}, nil
}
