package asset

import (
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

// GetByStorageId retrieves all assets for a storage
func GetByStorageId(l logrus.FieldLogger, db *gorm.DB, tenantId uuid.UUID) func(storageId uuid.UUID) ([]Model[any], error) {
	return func(storageId uuid.UUID) ([]Model[any], error) {
		var entities []Entity
		err := db.Where("tenant_id = ? AND storage_id = ?", tenantId, storageId).
			Order("slot ASC").
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
