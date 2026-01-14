package asset

import (
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
	"time"
)

// Create creates a new asset in storage
func Create(l logrus.FieldLogger, db *gorm.DB, tenantId uuid.UUID) func(storageId uuid.UUID, slot int16, templateId uint32, expiration time.Time, referenceId uint32, referenceType ReferenceType) (Model[any], error) {
	return func(storageId uuid.UUID, slot int16, templateId uint32, expiration time.Time, referenceId uint32, referenceType ReferenceType) (Model[any], error) {
		e := Entity{
			TenantId:      tenantId,
			StorageId:     storageId,
			InventoryType: InventoryTypeFromTemplateId(templateId),
			Slot:          slot,
			TemplateId:    templateId,
			Expiration:    expiration,
			ReferenceId:   referenceId,
			ReferenceType: referenceType,
		}
		err := db.Create(&e).Error
		if err != nil {
			return Model[any]{}, err
		}
		return Make(e), nil
	}
}

// Delete removes an asset from storage
func Delete(l logrus.FieldLogger, db *gorm.DB, tenantId uuid.UUID) func(id uint32) error {
	return func(id uint32) error {
		return db.Where("tenant_id = ? AND id = ?", tenantId, id).Delete(&Entity{}).Error
	}
}

// DeleteByStorageId removes all assets from a storage
func DeleteByStorageId(l logrus.FieldLogger, db *gorm.DB, tenantId uuid.UUID) func(storageId uuid.UUID) error {
	return func(storageId uuid.UUID) error {
		return db.Where("tenant_id = ? AND storage_id = ?", tenantId, storageId).Delete(&Entity{}).Error
	}
}

// UpdateSlot updates the slot of an asset
func UpdateSlot(l logrus.FieldLogger, db *gorm.DB, tenantId uuid.UUID) func(id uint32, slot int16) error {
	return func(id uint32, slot int16) error {
		return db.Model(&Entity{}).
			Where("tenant_id = ? AND id = ?", tenantId, id).
			Update("slot", slot).Error
	}
}
