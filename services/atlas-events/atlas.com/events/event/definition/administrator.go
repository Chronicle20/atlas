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
// in the same statement.
func setEnabled(db *gorm.DB) func(id uuid.UUID) func(enabled bool) error {
	return func(id uuid.UUID) func(enabled bool) error {
		return func(enabled bool) error {
			result := db.Model(&Entity{}).Where("id = ?", id).Updates(map[string]interface{}{
				"enabled":    enabled,
				"updated_at": time.Now(),
			})
			return result.Error
		}
	}
}
