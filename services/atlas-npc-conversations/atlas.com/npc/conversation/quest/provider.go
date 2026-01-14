package quest

import (
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// getByIdProvider returns a provider for retrieving a quest conversation by ID
func getByIdProvider(tenantId uuid.UUID) func(id uuid.UUID) func(db *gorm.DB) func() (Entity, error) {
	return func(id uuid.UUID) func(db *gorm.DB) func() (Entity, error) {
		return func(db *gorm.DB) func() (Entity, error) {
			return func() (Entity, error) {
				var entity Entity
				result := db.Where("tenant_id = ? AND id = ?", tenantId, id).First(&entity)
				return entity, result.Error
			}
		}
	}
}

// getByQuestIdProvider returns a provider for retrieving a quest conversation by quest ID
func getByQuestIdProvider(tenantId uuid.UUID) func(questId uint32) func(db *gorm.DB) func() (Entity, error) {
	return func(questId uint32) func(db *gorm.DB) func() (Entity, error) {
		return func(db *gorm.DB) func() (Entity, error) {
			return func() (Entity, error) {
				var entity Entity
				result := db.Where("tenant_id = ? AND quest_id = ?", tenantId, questId).First(&entity)
				return entity, result.Error
			}
		}
	}
}

// getAllProvider returns a provider for retrieving all quest conversations
func getAllProvider(tenantId uuid.UUID) func(db *gorm.DB) func() ([]Entity, error) {
	return func(db *gorm.DB) func() ([]Entity, error) {
		return func() ([]Entity, error) {
			var entities []Entity
			result := db.Where("tenant_id = ?", tenantId).Find(&entities)
			return entities, result.Error
		}
	}
}
