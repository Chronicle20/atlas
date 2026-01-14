package npc

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// createNpcConversation creates a new NPC conversation in the database
func createNpcConversation(db *gorm.DB) func(tenantId uuid.UUID) func(m Model) (Model, error) {
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

// updateNpcConversation updates an existing NPC conversation in the database
func updateNpcConversation(db *gorm.DB) func(tenantId uuid.UUID) func(id uuid.UUID) func(m Model) (Model, error) {
	return func(tenantId uuid.UUID) func(id uuid.UUID) func(m Model) (Model, error) {
		return func(id uuid.UUID) func(m Model) (Model, error) {
			return func(m Model) (Model, error) {
				// Check if conversation exists
				var existingEntity Entity
				result := db.Where("tenant_id = ? AND id = ?", tenantId, id).First(&existingEntity)
				if result.Error != nil {
					return Model{}, result.Error
				}

				// Convert model to entity
				entity, err := ToEntity(m, tenantId)
				if err != nil {
					return Model{}, err
				}

				// Ensure ID is preserved
				entity.ID = id

				// Update in database
				result = db.Model(&Entity{}).Where("tenant_id = ? AND id = ?", tenantId, id).Updates(map[string]interface{}{
					"npc_id":     entity.NpcID,
					"data":       entity.Data,
					"updated_at": time.Now(),
				})
				if result.Error != nil {
					return Model{}, result.Error
				}

				// Retrieve updated entity
				result = db.Where("tenant_id = ? AND id = ?", tenantId, id).First(&entity)
				if result.Error != nil {
					return Model{}, result.Error
				}

				return Make(entity)
			}
		}
	}
}

// deleteNpcConversation deletes an NPC conversation from the database
func deleteNpcConversation(db *gorm.DB) func(tenantId uuid.UUID) func(id uuid.UUID) error {
	return func(tenantId uuid.UUID) func(id uuid.UUID) error {
		return func(id uuid.UUID) error {
			result := db.Where("tenant_id = ? AND id = ?", tenantId, id).Delete(&Entity{})
			return result.Error
		}
	}
}

// deleteAllNpcConversations deletes all NPC conversations for a tenant using hard delete
func deleteAllNpcConversations(db *gorm.DB) func(tenantId uuid.UUID) (int64, error) {
	return func(tenantId uuid.UUID) (int64, error) {
		result := db.Unscoped().Where("tenant_id = ?", tenantId).Delete(&Entity{})
		if result.Error != nil {
			return 0, result.Error
		}
		return result.RowsAffected, nil
	}
}
