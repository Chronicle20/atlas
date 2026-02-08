package ban

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

func Migration(db *gorm.DB) error {
	return db.AutoMigrate(&Entity{})
}

type Entity struct {
	TenantId   uuid.UUID `gorm:"not null"`
	ID         uint32    `gorm:"primaryKey;autoIncrement;not null"`
	BanType    byte      `gorm:"not null"`
	Value      string    `gorm:"not null"`
	Reason     string    `gorm:"not null;default=''"`
	ReasonCode byte      `gorm:"not null;default=0"`
	Permanent  bool      `gorm:"not null;default=false"`
	ExpiresAt  int64     `gorm:"not null;default=0"`
	IssuedBy   string    `gorm:"not null;default=''"`
	CreatedAt  time.Time
	UpdatedAt  time.Time
}

func (e Entity) TableName() string {
	return "bans"
}
