package character

import (
	"github.com/google/uuid"
	"gorm.io/gorm"
)

func Migration(db *gorm.DB) error {
	return db.AutoMigrate(&entity{})
}

type entity struct {
	TenantId    uuid.UUID `gorm:"not null"`
	CharacterId uint32    `gorm:"primaryKey;autoIncrement:false;not null"`
	Capacity    uint32    `gorm:"not null"`
}

func (e entity) TableName() string {
	return "characters"
}
