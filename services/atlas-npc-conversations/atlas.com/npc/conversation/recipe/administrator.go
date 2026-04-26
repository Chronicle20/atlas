package recipe

import (
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// createRecipe inserts a single recipe row. Tenant-scoping is enforced via the
// entity's tenant_id column; pass a tx (e.g. obtained from db.Transaction) so
// the insert participates in the parent NPC-conversation write.
func createRecipe(db *gorm.DB) func(tenantId uuid.UUID) func(m Model) (Model, error) {
	return func(tenantId uuid.UUID) func(m Model) (Model, error) {
		return func(m Model) (Model, error) {
			entity, err := ToEntity(m, tenantId)
			if err != nil {
				return Model{}, err
			}
			if entity.ID == uuid.Nil {
				entity.ID = ComputeRecipeId(tenantId, m.ConversationId(), m.StateId())
			}
			if result := db.Create(&entity); result.Error != nil {
				return Model{}, result.Error
			}
			return Make(entity)
		}
	}
}

// deleteRecipesByConversation hard-deletes every recipe row attached to the
// given conversation id. Returns the number of rows removed.
func deleteRecipesByConversation(db *gorm.DB) func(conversationId uuid.UUID) (int64, error) {
	return func(conversationId uuid.UUID) (int64, error) {
		result := db.Where("conversation_id = ?", conversationId).Delete(&Entity{})
		return result.RowsAffected, result.Error
	}
}

// deleteAllRecipes hard-deletes every recipe row for the active tenant. Tenant
// scoping is enforced by the registered tenant callback on the GORM context.
func deleteAllRecipes(db *gorm.DB) (int64, error) {
	result := db.Where("1 = 1").Delete(&Entity{})
	return result.RowsAffected, result.Error
}
