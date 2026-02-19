package quest

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// createQuestConversation creates a new quest conversation in the database
func createQuestConversation(db *gorm.DB) func(tenantId uuid.UUID) func(m Model) (Model, error) {
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

// updateQuestConversation updates an existing quest conversation in the database
func updateQuestConversation(db *gorm.DB) func(id uuid.UUID) func(m Model) (Model, error) {
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
				"quest_id":   entity.QuestID,
				"npc_id":     entity.NpcID,
				"data":       entity.Data,
				"updated_at": time.Now(),
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

// deleteQuestConversation deletes a quest conversation from the database
func deleteQuestConversation(db *gorm.DB) func(id uuid.UUID) error {
	return func(id uuid.UUID) error {
		result := db.Where("id = ?", id).Delete(&Entity{})
		return result.Error
	}
}

// deleteAllQuestConversations deletes all quest conversations for a tenant using hard delete
func deleteAllQuestConversations(db *gorm.DB) (int64, error) {
	result := db.Unscoped().Where("1 = 1").Delete(&Entity{})
	if result.Error != nil {
		return 0, result.Error
	}
	return result.RowsAffected, nil
}
