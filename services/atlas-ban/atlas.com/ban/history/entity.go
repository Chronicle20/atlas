package history

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

func Migration(db *gorm.DB) error {
	return db.AutoMigrate(&Entity{})
}

type Entity struct {
	TenantId      uuid.UUID `gorm:"not null"`
	ID            uint64    `gorm:"primaryKey;autoIncrement;not null"`
	AccountId     uint32    `gorm:"not null;index"`
	AccountName   string    `gorm:"not null"`
	IPAddress     string    `gorm:"not null;default='';index"`
	HWID          string    `gorm:"not null;default='';index"`
	Success       bool      `gorm:"not null;default=false"`
	FailureReason string    `gorm:"not null;default=''"`
	CreatedAt     time.Time `gorm:"index"`
}

func (e Entity) TableName() string {
	return "login_history"
}
