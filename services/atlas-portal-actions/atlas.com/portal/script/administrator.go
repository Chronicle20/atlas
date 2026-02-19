package script

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// createPortalScript creates a new portal script in the database
func createPortalScript(db *gorm.DB) func(tenantId uuid.UUID) func(m PortalScript) (PortalScript, error) {
	return func(tenantId uuid.UUID) func(m PortalScript) (PortalScript, error) {
		return func(m PortalScript) (PortalScript, error) {
			entity, err := ToEntity(m, tenantId)
			if err != nil {
				return PortalScript{}, err
			}

			entity.ID = uuid.New()

			result := db.Create(&entity)
			if result.Error != nil {
				return PortalScript{}, result.Error
			}

			return Make(entity)
		}
	}
}

// updatePortalScript updates an existing portal script in the database
func updatePortalScript(db *gorm.DB) func(id uuid.UUID) func(m PortalScript, tenantId uuid.UUID) (PortalScript, error) {
	return func(id uuid.UUID) func(m PortalScript, tenantId uuid.UUID) (PortalScript, error) {
		return func(m PortalScript, tenantId uuid.UUID) (PortalScript, error) {
			// Check if script exists
			var existingEntity Entity
			result := db.Where("id = ?", id).First(&existingEntity)
			if result.Error != nil {
				return PortalScript{}, result.Error
			}

			// Convert model to entity
			entity, err := ToEntity(m, tenantId)
			if err != nil {
				return PortalScript{}, err
			}

			// Ensure ID is preserved
			entity.ID = id

			// Update in database
			result = db.Model(&Entity{}).Where("id = ?", id).Updates(map[string]interface{}{
				"portal_id":  entity.PortalID,
				"map_id":     entity.MapID,
				"data":       entity.Data,
				"updated_at": time.Now(),
			})
			if result.Error != nil {
				return PortalScript{}, result.Error
			}

			// Retrieve updated entity
			result = db.Where("id = ?", id).First(&entity)
			if result.Error != nil {
				return PortalScript{}, result.Error
			}

			return Make(entity)
		}
	}
}

// deletePortalScript deletes a portal script from the database (soft delete)
func deletePortalScript(db *gorm.DB) func(id uuid.UUID) error {
	return func(id uuid.UUID) error {
		result := db.Where("id = ?", id).Delete(&Entity{})
		return result.Error
	}
}

// deleteAllPortalScripts deletes all portal scripts for a tenant using hard delete
func deleteAllPortalScripts(db *gorm.DB) (int64, error) {
	result := db.Unscoped().Delete(&Entity{})
	if result.Error != nil {
		return 0, result.Error
	}
	return result.RowsAffected, nil
}
