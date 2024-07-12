package account

import (
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type State byte

const (
	NotLoggedIn       State = 0
	ServerTransistion State = 1
	LoggedIn          State = 2
)

func Migration(db *gorm.DB) error {
	return db.AutoMigrate(&entity{})
}

type entity struct {
	TenantId  uuid.UUID `gorm:"not null"`
	ID        uint32    `gorm:"primaryKey;autoIncrement;not null"`
	Name      string    `gorm:"not null"`
	Password  string    `gorm:"not null"`
	PIN       string
	PIC       string
	Gender    byte
	State     byte `gorm:"not null;default=0"`
	LastLogin int64
}

func (e entity) TableName() string {
	return "accounts"
}
