package definition

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// create persists a new event definition, stamping the tenant id and
// returning the written model.
func create(db *gorm.DB) func(tenantId uuid.UUID) func(m Model) (Model, error) {
	return func(tenantId uuid.UUID) func(m Model) (Model, error) {
		return func(m Model) (Model, error) {
			entity, err := ToEntity(m, tenantId)
			if err != nil {
				return Model{}, err
			}

			result := db.Create(&entity)
			if result.Error != nil {
				return Model{}, result.Error
			}

			return Make(entity)
		}
	}
}

// setEnabled updates the enabled flag for a definition, updating updated_at
// in the same statement. It checks existence first — matching the
// party-quests reference administrator's updateDefinition shape — so a
// nonexistent id surfaces gorm.ErrRecordNotFound rather than silently
// no-oping with a nil error.
func setEnabled(db *gorm.DB) func(id uuid.UUID) func(enabled bool) error {
	return func(id uuid.UUID) func(enabled bool) error {
		return func(enabled bool) error {
			var existing Entity
			if result := db.Where("id = ?", id).First(&existing); result.Error != nil {
				return result.Error
			}

			result := db.Model(&Entity{}).Where("id = ?", id).Updates(map[string]interface{}{
				"enabled":    enabled,
				"updated_at": time.Now(),
			})
			return result.Error
		}
	}
}
