package definition

import (
	"github.com/Chronicle20/atlas-model/model"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

func getByIdProvider(tenantId uuid.UUID) func(id uuid.UUID) func(db *gorm.DB) model.Provider[Entity] {
	return func(id uuid.UUID) func(db *gorm.DB) model.Provider[Entity] {
		return func(db *gorm.DB) model.Provider[Entity] {
			return func() (Entity, error) {
				var entity Entity
				result := db.Where("tenant_id = ? AND id = ?", tenantId, id).First(&entity)
				return entity, result.Error
			}
		}
	}
}

func getByQuestIdProvider(tenantId uuid.UUID) func(questId string) func(db *gorm.DB) model.Provider[Entity] {
	return func(questId string) func(db *gorm.DB) model.Provider[Entity] {
		return func(db *gorm.DB) model.Provider[Entity] {
			return func() (Entity, error) {
				var entity Entity
				result := db.Where("tenant_id = ? AND quest_id = ?", tenantId, questId).First(&entity)
				return entity, result.Error
			}
		}
	}
}

func getAllProvider(tenantId uuid.UUID) func(db *gorm.DB) model.Provider[[]Entity] {
	return func(db *gorm.DB) model.Provider[[]Entity] {
		return func() ([]Entity, error) {
			var entities []Entity
			result := db.Where("tenant_id = ?", tenantId).Find(&entities)
			return entities, result.Error
		}
	}
}
