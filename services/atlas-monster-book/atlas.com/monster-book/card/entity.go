package card

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

func Migration(db *gorm.DB) error {
	return db.AutoMigrate(&entity{})
}

type entity struct {
	TenantId        uuid.UUID  `gorm:"primaryKey;autoIncrement:false;not null"`
	CharacterId     uint32     `gorm:"primaryKey;autoIncrement:false;not null"`
	CardId          uint32     `gorm:"primaryKey;autoIncrement:false;not null"`
	Level           uint8      `gorm:"not null"`
	IsSpecial       bool       `gorm:"not null;default:false;index"`
	LastEventId     *uuid.UUID `gorm:""`
	FirstAcquiredAt time.Time  `gorm:"autoCreateTime"`
	UpdatedAt       time.Time  `gorm:"autoUpdateTime"`
}

func (entity) TableName() string { return "monster_book_cards" }
