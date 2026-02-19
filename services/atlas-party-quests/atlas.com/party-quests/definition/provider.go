package definition

import (
	"github.com/Chronicle20/atlas-model/model"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

func getByIdProvider(id uuid.UUID) func(db *gorm.DB) model.Provider[Entity] {
	return func(db *gorm.DB) model.Provider[Entity] {
		return func() (Entity, error) {
			var entity Entity
			result := db.Where("id = ?", id).First(&entity)
			return entity, result.Error
		}
	}
}

func getByQuestIdProvider(questId string) func(db *gorm.DB) model.Provider[Entity] {
	return func(db *gorm.DB) model.Provider[Entity] {
		return func() (Entity, error) {
			var entity Entity
			result := db.Where("quest_id = ?", questId).First(&entity)
			return entity, result.Error
		}
	}
}

func getAllProvider(db *gorm.DB) model.Provider[[]Entity] {
	return func() ([]Entity, error) {
		var entities []Entity
		result := db.Find(&entities)
		return entities, result.Error
	}
}
