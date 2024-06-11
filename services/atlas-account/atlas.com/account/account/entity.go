package account

import (
	"github.com/google/uuid"
	"gorm.io/gorm"
)

const (
	StateNotLoggedIn       byte = 0
	StateServerTransistion byte = 1
	StateLoggedIn          byte = 2
)

func Migration(db *gorm.DB) error {
	return db.AutoMigrate(&entity{})
}

type entity struct {
	TenantId  uuid.UUID `gorm:"not null"`
	ID        uint32    `gorm:"primaryKey;autoIncrement;not null"`
	Name      string    `gorm:"not null"`
	Password  string    `gorm:"not null"`
	State     byte      `gorm:"not null;default=0"`
	LastLogin int64
}

func (e entity) TableName() string {
	return "accounts"
}
