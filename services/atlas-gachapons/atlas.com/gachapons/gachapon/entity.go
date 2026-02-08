package gachapon

import (
	"github.com/google/uuid"
	"github.com/lib/pq"
	"gorm.io/gorm"
)

func Migration(db *gorm.DB) error {
	return db.AutoMigrate(&entity{})
}

type entity struct {
	TenantId       uuid.UUID      `gorm:"not null"`
	ID             string         `gorm:"primaryKey;not null"`
	Name           string         `gorm:"not null"`
	NpcIds         pq.Int64Array  `gorm:"type:integer[];not null"`
	CommonWeight   uint32         `gorm:"not null;default:70"`
	UncommonWeight uint32         `gorm:"not null;default:25"`
	RareWeight     uint32         `gorm:"not null;default:5"`
}

func (e entity) TableName() string {
	return "gachapons"
}
