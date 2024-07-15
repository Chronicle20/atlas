package account

import (
	"github.com/google/uuid"
	"gorm.io/gorm"
	"time"
)

type State byte

const (
	NotLoggedIn       State = 0
	InLogin           State = 1
	ServerTransistion State = 2
	LoggedIn          State = 3
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
	Gender    byte `gorm:"not null;default=0"`
	State     byte `gorm:"not null;default=0"`
	TOS       bool `gorm:"not null;default=false"`
	LastLogin int64
	CreatedAt time.Time // Automatically managed by GORM for creation time
	UpdatedAt time.Time // Automatically managed by GORM for update time
}

func (e entity) TableName() string {
	return "accounts"
}
