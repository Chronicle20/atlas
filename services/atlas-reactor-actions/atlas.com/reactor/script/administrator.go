package script

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// createReactorScript creates a new reactor script in the database
func createReactorScript(db *gorm.DB) func(tenantId uuid.UUID) func(m ReactorScript) (ReactorScript, error) {
	return func(tenantId uuid.UUID) func(m ReactorScript) (ReactorScript, error) {
		return func(m ReactorScript) (ReactorScript, error) {
			entity, err := ToEntity(m, tenantId)
			if err != nil {
				return ReactorScript{}, err
			}

			entity.ID = uuid.New()

			result := db.Create(&entity)
			if result.Error != nil {
				return ReactorScript{}, result.Error
			}

			return Make(entity)
		}
	}
}

// updateReactorScript updates an existing reactor script in the database
func updateReactorScript(db *gorm.DB) func(tenantId uuid.UUID) func(id uuid.UUID) func(m ReactorScript) (ReactorScript, error) {
	return func(tenantId uuid.UUID) func(id uuid.UUID) func(m ReactorScript) (ReactorScript, error) {
		return func(id uuid.UUID) func(m ReactorScript) (ReactorScript, error) {
			return func(m ReactorScript) (ReactorScript, error) {
				// Check if script exists
				var existingEntity Entity
				result := db.Where("tenant_id = ? AND id = ?", tenantId, id).First(&existingEntity)
				if result.Error != nil {
					return ReactorScript{}, result.Error
				}

				// Convert model to entity
				entity, err := ToEntity(m, tenantId)
				if err != nil {
					return ReactorScript{}, err
				}

				// Ensure ID is preserved
				entity.ID = id

				// Update in database
				result = db.Model(&Entity{}).Where("tenant_id = ? AND id = ?", tenantId, id).Updates(map[string]interface{}{
					"reactor_id": entity.ReactorID,
					"data":       entity.Data,
					"updated_at": time.Now(),
				})
				if result.Error != nil {
					return ReactorScript{}, result.Error
				}

				// Retrieve updated entity
				result = db.Where("tenant_id = ? AND id = ?", tenantId, id).First(&entity)
				if result.Error != nil {
					return ReactorScript{}, result.Error
				}

				return Make(entity)
			}
		}
	}
}

// deleteReactorScript deletes a reactor script from the database (soft delete)
func deleteReactorScript(db *gorm.DB) func(tenantId uuid.UUID) func(id uuid.UUID) error {
	return func(tenantId uuid.UUID) func(id uuid.UUID) error {
		return func(id uuid.UUID) error {
			result := db.Where("tenant_id = ? AND id = ?", tenantId, id).Delete(&Entity{})
			return result.Error
		}
	}
}

// deleteAllReactorScripts deletes all reactor scripts for a tenant using hard delete
func deleteAllReactorScripts(db *gorm.DB) func(tenantId uuid.UUID) (int64, error) {
	return func(tenantId uuid.UUID) (int64, error) {
		result := db.Unscoped().Where("tenant_id = ?", tenantId).Delete(&Entity{})
		if result.Error != nil {
			return 0, result.Error
		}
		return result.RowsAffected, nil
	}
}
