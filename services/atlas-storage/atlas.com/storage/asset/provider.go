package asset

import (
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

// GetByStorageId retrieves all assets for a storage with computed slots.
// Slots are computed dynamically based on inventory_type and template_id ordering.
// This is necessary because the client reorders slots dynamically.
func GetByStorageId(l logrus.FieldLogger, db *gorm.DB, tenantId uuid.UUID) func(storageId uuid.UUID) ([]Model[any], error) {
	return func(storageId uuid.UUID) ([]Model[any], error) {
		var entities []Entity
		err := db.Where("tenant_id = ? AND storage_id = ?", tenantId, storageId).
			Order("inventory_type ASC, template_id ASC").
			Find(&entities).Error
		if err != nil {
			return nil, err
		}

		// Compute slots dynamically based on position in sorted order
		models := make([]Model[any], 0, len(entities))
		for i, e := range entities {
			// Create model with computed slot
			m := NewModelBuilder[any]().
				SetId(e.Id).
				SetStorageId(e.StorageId).
				SetInventoryType(e.InventoryType).
				SetSlot(int16(i)).
				SetTemplateId(e.TemplateId).
				SetExpiration(e.Expiration).
				SetReferenceId(e.ReferenceId).
				SetReferenceType(e.ReferenceType).
				MustBuild()
			models = append(models, m)
		}
		return models, nil
	}
}

// GetById retrieves an asset by ID
func GetById(l logrus.FieldLogger, db *gorm.DB, tenantId uuid.UUID) func(id uint32) (Model[any], error) {
	return func(id uint32) (Model[any], error) {
		var e Entity
		err := db.Where("tenant_id = ? AND id = ?", tenantId, id).First(&e).Error
		if err != nil {
			return Model[any]{}, err
		}
		return Make(e), nil
	}
}

// GetByStorageIdAndTemplateId retrieves assets with a specific templateId in a storage
func GetByStorageIdAndTemplateId(l logrus.FieldLogger, db *gorm.DB, tenantId uuid.UUID) func(storageId uuid.UUID, templateId uint32) ([]Model[any], error) {
	return func(storageId uuid.UUID, templateId uint32) ([]Model[any], error) {
		var entities []Entity
		err := db.Where("tenant_id = ? AND storage_id = ? AND template_id = ?", tenantId, storageId, templateId).
			Find(&entities).Error
		if err != nil {
			return nil, err
		}
		models := make([]Model[any], 0, len(entities))
		for _, e := range entities {
			models = append(models, Make(e))
		}
		return models, nil
	}
}
