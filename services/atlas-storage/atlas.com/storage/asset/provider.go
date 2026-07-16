package asset

import (
	"github.com/google/uuid"
	"gorm.io/gorm"
)

func GetByStorageId(db *gorm.DB) func(storageId uuid.UUID) ([]Model, error) {
	return func(storageId uuid.UUID) ([]Model, error) {
		var entities []Entity
		// id ASC is an explicit tiebreak after inventory_type/template_id so
		// the dynamic-slot assignment below (index i) is fully deterministic
		// across repeated calls, not just "mostly stable" via SQL's
		// otherwise-undefined tie order (task-117 pagination determinism
		// requirement -- this list is paginated via paginate.Slice in
		// resource.go, which requires a stable, reproducible ordering).
		err := db.Where("storage_id = ?", storageId).
			Order("inventory_type ASC, template_id ASC, id ASC").
			Find(&entities).Error
		if err != nil {
			return nil, err
		}

		models := make([]Model, 0, len(entities))
		for i, e := range entities {
			models = append(models, MakeWithDynamicSlot(e, int16(i)))
		}
		return models, nil
	}
}

func GetById(db *gorm.DB) func(id uint32) (Model, error) {
	return func(id uint32) (Model, error) {
		var e Entity
		err := db.Where("id = ?", id).First(&e).Error
		if err != nil {
			return Model{}, err
		}
		return Make(e), nil
	}
}

func GetByStorageIdAndTemplateId(db *gorm.DB) func(storageId uuid.UUID, templateId uint32) ([]Model, error) {
	return func(storageId uuid.UUID, templateId uint32) ([]Model, error) {
		var entities []Entity
		err := db.Where("storage_id = ? AND template_id = ?", storageId, templateId).
			Find(&entities).Error
		if err != nil {
			return nil, err
		}
		models := make([]Model, 0, len(entities))
		for _, e := range entities {
			models = append(models, Make(e))
		}
		return models, nil
	}
}

func GetByStorageIdAndInventoryType(db *gorm.DB) func(storageId uuid.UUID, inventoryType byte) ([]Model, error) {
	return func(storageId uuid.UUID, inventoryType byte) ([]Model, error) {
		var entities []Entity
		err := db.Where("storage_id = ? AND inventory_type = ?", storageId, inventoryType).
			Order("template_id ASC").
			Find(&entities).Error
		if err != nil {
			return nil, err
		}
		models := make([]Model, 0, len(entities))
		for i, e := range entities {
			models = append(models, MakeWithDynamicSlot(e, int16(i)))
		}
		return models, nil
	}
}
