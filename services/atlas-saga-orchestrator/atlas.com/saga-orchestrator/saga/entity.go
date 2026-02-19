package saga

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

func Migration(db *gorm.DB) error {
	return db.AutoMigrate(&Entity{})
}

type Entity struct {
	TransactionId uuid.UUID `gorm:"type:uuid;primaryKey;not null"`
	TenantId      uuid.UUID `gorm:"type:uuid;not null;index:idx_sagas_tenant"`
	TenantRegion  string    `gorm:"type:varchar(32);not null;default:''"`
	TenantMajor   uint16    `gorm:"not null;default:0"`
	TenantMinor   uint16    `gorm:"not null;default:0"`
	SagaType      string    `gorm:"type:varchar(64);not null"`
	InitiatedBy   string    `gorm:"type:varchar(255);not null"`
	Status        string    `gorm:"type:varchar(16);not null;default:active;index:idx_sagas_status"`
	SagaData      []byte    `gorm:"type:jsonb;not null"`
	Version       int       `gorm:"not null;default:1"`
	CreatedAt     time.Time
	UpdatedAt     time.Time
	TimeoutAt     *time.Time `gorm:"index:idx_sagas_timeout"`
}

func (e Entity) TableName() string {
	return "sagas"
}
