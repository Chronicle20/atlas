package collection

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

func Migration(db *gorm.DB) error {
	return db.AutoMigrate(&entity{})
}

type entity struct {
	TenantId         uuid.UUID  `gorm:"primaryKey;autoIncrement:false;not null"`
	CharacterId      uint32     `gorm:"primaryKey;autoIncrement:false;not null"`
	CoverCardId      uint32     `gorm:"not null;default:0"`
	CoverMobId       uint32     `gorm:"not null;default:0"`
	BookLevel        uint16     `gorm:"not null;default:1"`
	NormalCount      uint16     `gorm:"not null;default:0"`
	SpecialCount     uint16     `gorm:"not null;default:0"`
	ExpBonusPercent  uint16     `gorm:"not null;default:0"`
	LastCoverEventId *uuid.UUID `gorm:""`
	CreatedAt        time.Time  `gorm:"autoCreateTime"`
	UpdatedAt        time.Time  `gorm:"autoUpdateTime"`
}

func (entity) TableName() string { return "monster_book_collections" }
