package asset

import (
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

// GetByStorageId retrieves all assets for a storage
// Slots are computed dynamically based on ordering by inventory_type, then template_id
func GetByStorageId(l logrus.FieldLogger, db *gorm.DB, tenantId uuid.UUID) func(storageId uuid.UUID) ([]Model[any], error) {
	return func(storageId uuid.UUID) ([]Model[any], error) {
		var entities []Entity
		err := db.Where("tenant_id = ? AND storage_id = ?", tenantId, storageId).
			Order("inventory_type ASC, template_id ASC").
			Find(&entities).Error
		if err != nil {
			return nil, err
		}

		models := make([]Model[any], 0, len(entities))
		for i, e := range entities {
			models = append(models, MakeWithDynamicSlot(e, int16(i)))
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

// GetByStorageIdAndInventoryType retrieves assets with a specific inventory type in a storage
// Slots are computed dynamically based on ordering by template_id
func GetByStorageIdAndInventoryType(l logrus.FieldLogger, db *gorm.DB, tenantId uuid.UUID) func(storageId uuid.UUID, inventoryType InventoryType) ([]Model[any], error) {
	return func(storageId uuid.UUID, inventoryType InventoryType) ([]Model[any], error) {
		var entities []Entity
		err := db.Where("tenant_id = ? AND storage_id = ? AND inventory_type = ?", tenantId, storageId, inventoryType).
			Order("template_id ASC").
			Find(&entities).Error
		if err != nil {
			return nil, err
		}
		models := make([]Model[any], 0, len(entities))
		for i, e := range entities {
			models = append(models, MakeWithDynamicSlot(e, int16(i)))
		}
		return models, nil
	}
}
