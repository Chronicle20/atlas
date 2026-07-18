package quest

import (
	"github.com/google/uuid"
	"gorm.io/gorm"

	database "github.com/Chronicle20/atlas/libs/atlas-database"
	"github.com/Chronicle20/atlas/libs/atlas-model/model"
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

// getAllPagedProvider returns a provider for retrieving one page of quest
// conversations for a tenant
func getAllPagedProvider(page model.Page) database.EntityProvider[model.Paged[Entity]] {
	return func(db *gorm.DB) model.Provider[model.Paged[Entity]] {
		return database.PagedQuery[Entity](db, page)
	}
}
