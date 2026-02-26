package frederick

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type NotificationEntity struct {
	gorm.Model
	Id           uuid.UUID `gorm:"type:uuid;primaryKey"`
	TenantId     uuid.UUID `gorm:"type:uuid;not null"`
	TenantRegion string    `gorm:"not null"`
	TenantMajor  uint16    `gorm:"not null"`
	TenantMinor  uint16    `gorm:"not null"`
	CharacterId  uint32    `gorm:"not null;index"`
	StoredAt     time.Time `gorm:"not null"`
	NextDay      uint16    `gorm:"not null"`
}

func (e *NotificationEntity) TableName() string {
	return "frederick_notifications"
}
