package pet

import (
	"atlas-pets/pet/exclude"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

func Migration(db *gorm.DB) error {
	return db.AutoMigrate(&Entity{})
}

type Entity struct {
	TenantId        uuid.UUID        `gorm:"not null;"`
	OwnerId         uint32           `gorm:"not null;"`
	Id              uint32           `gorm:"primary_key;auto_increment"`
	InventoryItemId uint32           `gorm:"not null"`
	TemplateId      uint32           `gorm:"not null"`
	Name            string           `gorm:"size:13;not null"`
	Level           byte             `gorm:"not null;default:1"`
	Closeness       uint16           `gorm:"not null;default:0"`
	Fullness        byte             `gorm:"not null;default:100"`
	Expiration      time.Time        `gorm:"not null;"`
	Slot            int8             `gorm:"not null;default:-1"`
	Excludes        []exclude.Entity `gorm:"foreignkey:PetId"`
}

func (e Entity) TableName() string {
	return "pets"
}
