package definition

import (
	"github.com/google/uuid"
	"gorm.io/gorm"

	database "github.com/Chronicle20/atlas/libs/atlas-database"
	"github.com/Chronicle20/atlas/libs/atlas-model/model"
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

func getAllPagedProvider(page model.Page) database.EntityProvider[model.Paged[Entity]] {
	return func(db *gorm.DB) model.Provider[model.Paged[Entity]] {
		return database.PagedQuery[Entity](db, page)
	}
}
