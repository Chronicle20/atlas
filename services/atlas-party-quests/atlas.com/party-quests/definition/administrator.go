package definition

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

func createDefinition(db *gorm.DB) func(tenantId uuid.UUID) func(m Model) (Model, error) {
	return func(tenantId uuid.UUID) func(m Model) (Model, error) {
		return func(m Model) (Model, error) {
			entity, err := ToEntity(m, tenantId)
			if err != nil {
				return Model{}, err
			}

			entity.ID = uuid.New()

			result := db.Create(&entity)
			if result.Error != nil {
				return Model{}, result.Error
			}

			return Make(entity)
		}
	}
}

func updateDefinition(db *gorm.DB) func(tenantId uuid.UUID) func(id uuid.UUID) func(m Model) (Model, error) {
	return func(tenantId uuid.UUID) func(id uuid.UUID) func(m Model) (Model, error) {
		return func(id uuid.UUID) func(m Model) (Model, error) {
			return func(m Model) (Model, error) {
				var existingEntity Entity
				result := db.Where("tenant_id = ? AND id = ?", tenantId, id).First(&existingEntity)
				if result.Error != nil {
					return Model{}, result.Error
				}

				entity, err := ToEntity(m, tenantId)
				if err != nil {
					return Model{}, err
				}

				entity.ID = id

				result = db.Model(&Entity{}).Where("tenant_id = ? AND id = ?", tenantId, id).Updates(map[string]interface{}{
					"quest_id":   entity.QuestID,
					"data":       entity.Data,
					"updated_at": time.Now(),
				})
				if result.Error != nil {
					return Model{}, result.Error
				}

				result = db.Where("tenant_id = ? AND id = ?", tenantId, id).First(&entity)
				if result.Error != nil {
					return Model{}, result.Error
				}

				return Make(entity)
			}
		}
	}
}

func deleteDefinition(db *gorm.DB) func(tenantId uuid.UUID) func(id uuid.UUID) error {
	return func(tenantId uuid.UUID) func(id uuid.UUID) error {
		return func(id uuid.UUID) error {
			result := db.Where("tenant_id = ? AND id = ?", tenantId, id).Delete(&Entity{})
			return result.Error
		}
	}
}

func deleteAllDefinitions(db *gorm.DB) func(tenantId uuid.UUID) (int64, error) {
	return func(tenantId uuid.UUID) (int64, error) {
		result := db.Unscoped().Where("tenant_id = ?", tenantId).Delete(&Entity{})
		if result.Error != nil {
			return 0, result.Error
		}
		return result.RowsAffected, nil
	}
}
