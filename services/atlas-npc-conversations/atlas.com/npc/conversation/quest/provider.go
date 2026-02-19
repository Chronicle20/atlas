package quest

import (
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// getByIdProvider returns a provider for retrieving a quest conversation by ID
func getByIdProvider(id uuid.UUID) func(db *gorm.DB) func() (Entity, error) {
	return func(db *gorm.DB) func() (Entity, error) {
		return func() (Entity, error) {
			var entity Entity
			result := db.Where("id = ?", id).First(&entity)
			return entity, result.Error
		}
	}
}

// getByQuestIdProvider returns a provider for retrieving a quest conversation by quest ID
func getByQuestIdProvider(questId uint32) func(db *gorm.DB) func() (Entity, error) {
	return func(db *gorm.DB) func() (Entity, error) {
		return func() (Entity, error) {
			var entity Entity
			result := db.Where("quest_id = ?", questId).First(&entity)
			return entity, result.Error
		}
	}
}

// getAllProvider returns a provider for retrieving all quest conversations
func getAllProvider(db *gorm.DB) func() ([]Entity, error) {
	return func() ([]Entity, error) {
		var entities []Entity
		result := db.Find(&entities)
		return entities, result.Error
	}
}
