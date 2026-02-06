package account

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

func Migration(db *gorm.DB) error {
	return db.AutoMigrate(&Entity{})
}

type Entity struct {
	TenantId  uuid.UUID `gorm:"not null"`
	ID        uint32    `gorm:"primaryKey;autoIncrement;not null"`
	Name      string    `gorm:"not null"`
	Password  string    `gorm:"not null"`
	PIN       string
	PIC       string
	Gender    byte `gorm:"not null;default=0"`
	TOS       bool `gorm:"not null;default=false"`
	LastLogin int64
	CreatedAt time.Time // Automatically managed by GORM for creation time
	UpdatedAt time.Time // Automatically managed by GORM for update time
}

func (e Entity) TableName() string {
	return "accounts"
}
