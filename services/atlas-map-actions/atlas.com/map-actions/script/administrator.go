package script

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// createMapScript creates a new map script in the database
func createMapScript(db *gorm.DB) func(tenantId uuid.UUID) func(m MapScript) (MapScript, error) {
	return func(tenantId uuid.UUID) func(m MapScript) (MapScript, error) {
		return func(m MapScript) (MapScript, error) {
			entity, err := ToEntity(m, tenantId)
			if err != nil {
				return MapScript{}, err
			}

			entity.ID = uuid.New()

			result := db.Create(&entity)
			if result.Error != nil {
				return MapScript{}, result.Error
			}

			return Make(entity)
		}
	}
}

// updateMapScript updates an existing map script in the database
func updateMapScript(db *gorm.DB) func(tenantId uuid.UUID) func(id uuid.UUID) func(m MapScript) (MapScript, error) {
	return func(tenantId uuid.UUID) func(id uuid.UUID) func(m MapScript) (MapScript, error) {
		return func(id uuid.UUID) func(m MapScript) (MapScript, error) {
			return func(m MapScript) (MapScript, error) {
				var existingEntity Entity
				result := db.Where("tenant_id = ? AND id = ?", tenantId, id).First(&existingEntity)
				if result.Error != nil {
					return MapScript{}, result.Error
				}

				entity, err := ToEntity(m, tenantId)
				if err != nil {
					return MapScript{}, err
				}

				entity.ID = id

				result = db.Model(&Entity{}).Where("tenant_id = ? AND id = ?", tenantId, id).Updates(map[string]interface{}{
					"script_name": entity.ScriptName,
					"script_type": entity.ScriptType,
					"data":        entity.Data,
					"updated_at":  time.Now(),
				})
				if result.Error != nil {
					return MapScript{}, result.Error
				}

				result = db.Where("tenant_id = ? AND id = ?", tenantId, id).First(&entity)
				if result.Error != nil {
					return MapScript{}, result.Error
				}

				return Make(entity)
			}
		}
	}
}

// deleteMapScript deletes a map script from the database (soft delete)
func deleteMapScript(db *gorm.DB) func(tenantId uuid.UUID) func(id uuid.UUID) error {
	return func(tenantId uuid.UUID) func(id uuid.UUID) error {
		return func(id uuid.UUID) error {
			result := db.Where("tenant_id = ? AND id = ?", tenantId, id).Delete(&Entity{})
			return result.Error
		}
	}
}

// deleteAllMapScripts deletes all map scripts for a tenant using hard delete
func deleteAllMapScripts(db *gorm.DB) func(tenantId uuid.UUID) (int64, error) {
	return func(tenantId uuid.UUID) (int64, error) {
		result := db.Unscoped().Where("tenant_id = ?", tenantId).Delete(&Entity{})
		if result.Error != nil {
			return 0, result.Error
		}
		return result.RowsAffected, nil
	}
}
