package asset

import (
	"github.com/google/uuid"
	"gorm.io/gorm"
	"time"
)

type Entity struct {
	TenantId      uuid.UUID     `gorm:"not null;index:idx_asset_tenant_storage"`
	Id            uint32        `gorm:"primaryKey;autoIncrement"`
	StorageId     uuid.UUID     `gorm:"not null;index:idx_asset_tenant_storage"`
	Slot          int16         `gorm:"not null"`
	TemplateId    uint32        `gorm:"not null"`
	Expiration    time.Time     `gorm:"not null"`
	ReferenceId   uint32        `gorm:"not null"`
	ReferenceType ReferenceType `gorm:"not null;type:varchar(20)"`
}

func (e Entity) TableName() string {
	return "storage_assets"
}

func Migration(db *gorm.DB) error {
	return db.AutoMigrate(&Entity{})
}

// Make converts an Entity to a Model with empty reference data
func Make(e Entity) Model[any] {
	return NewModelBuilder[any]().
		SetId(e.Id).
		SetStorageId(e.StorageId).
		SetSlot(e.Slot).
		SetTemplateId(e.TemplateId).
		SetExpiration(e.Expiration).
		SetReferenceId(e.ReferenceId).
		SetReferenceType(e.ReferenceType).
		Build()
}
