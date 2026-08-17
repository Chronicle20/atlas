package item

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// createItemConversation creates a new item conversation in the database
func createItemConversation(db *gorm.DB) func(tenantId uuid.UUID) func(m Model) (Model, error) {
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

// updateItemConversation updates an existing item conversation in the database
func updateItemConversation(db *gorm.DB) func(id uuid.UUID) func(m Model) (Model, error) {
	return func(id uuid.UUID) func(m Model) (Model, error) {
		return func(m Model) (Model, error) {
			// Check if conversation exists
			var existingEntity Entity
			result := db.Where("id = ?", id).First(&existingEntity)
			if result.Error != nil {
				return Model{}, result.Error
			}

			// Convert model to entity (use existing tenant ID from the found entity)
			entity, err := ToEntity(m, existingEntity.TenantID)
			if err != nil {
				return Model{}, err
			}

			// Ensure ID is preserved
			entity.ID = id

			// Update in database
			result = db.Model(&Entity{}).Where("id = ?", id).Updates(map[string]interface{}{
				"item_id":     entity.ItemID,
				"npc_id":      entity.NpcID,
				"script_name": entity.ScriptName,
				"data":        entity.Data,
				"updated_at":  time.Now(),
			})
			if result.Error != nil {
				return Model{}, result.Error
			}

			// Retrieve updated entity
			result = db.Where("id = ?", id).First(&entity)
			if result.Error != nil {
				return Model{}, result.Error
			}

			return Make(entity)
		}
	}
}

// deleteItemConversation deletes an item conversation from the database
func deleteItemConversation(db *gorm.DB) func(id uuid.UUID) error {
	return func(id uuid.UUID) error {
		result := db.Where("id = ?", id).Delete(&Entity{})
		return result.Error
	}
}

// deleteAllItemConversations deletes all item conversations for a tenant using hard delete
func deleteAllItemConversations(db *gorm.DB) (int64, error) {
	result := db.Unscoped().Where("1 = 1").Delete(&Entity{})
	if result.Error != nil {
		return 0, result.Error
	}
	return result.RowsAffected, nil
}
