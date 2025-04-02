package wallet

import (
	"github.com/google/uuid"
	"gorm.io/gorm"
)

func Migration(db *gorm.DB) error {
	return db.AutoMigrate(&Entity{})
}

type Entity struct {
	Id          uuid.UUID `gorm:"primaryKey;type:uuid;default:uuid_generate_v4()"`
	TenantId    uuid.UUID `gorm:"not null"`
	CharacterId uint32    `gorm:"not null"`
	Credit      uint32    `gorm:"not null;default=0"`
	Points      uint32    `gorm:"not null;default=0"`
	Prepaid     uint32    `gorm:"not null;default=0"`
}

func (e Entity) TableName() string {
	return "characters"
}

func Make(e Entity) (Model, error) {
	return Model{
		id:          e.Id,
		characterId: e.CharacterId,
		credit:      e.Credit,
		points:      e.Points,
		prepaid:     e.Prepaid,
	}, nil
}
