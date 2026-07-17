package npc

import (
	database "github.com/Chronicle20/atlas/libs/atlas-database"
	"github.com/Chronicle20/atlas/libs/atlas-model/model"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// getByIdProvider returns a provider for retrieving a conversation by ID
func getByIdProvider(id uuid.UUID) func(db *gorm.DB) model.Provider[Entity] {
	return func(db *gorm.DB) model.Provider[Entity] {
		return func() (Entity, error) {
			var entity Entity
			result := db.Where("id = ?", id).First(&entity)
			return entity, result.Error
		}
	}
}

// getByNpcIdProvider returns a provider for retrieving a conversation by NPC ID
func getByNpcIdProvider(npcId uint32) func(db *gorm.DB) model.Provider[Entity] {
	return func(db *gorm.DB) model.Provider[Entity] {
		return func() (Entity, error) {
			var entity Entity
			result := db.Where("npc_id = ?", npcId).First(&entity)
			return entity, result.Error
		}
	}
}

// getAllPagedProvider returns a provider for retrieving one page of
// conversations for a tenant
func getAllPagedProvider(page model.Page) database.EntityProvider[model.Paged[Entity]] {
	return func(db *gorm.DB) model.Provider[model.Paged[Entity]] {
		return database.PagedQuery[Entity](db, page)
	}
}

// getAllByNpcIdPagedProvider returns a provider for retrieving one page of
// conversations for a specific NPC ID
func getAllByNpcIdPagedProvider(npcId uint32, page model.Page) database.EntityProvider[model.Paged[Entity]] {
	return func(db *gorm.DB) model.Provider[model.Paged[Entity]] {
		return database.PagedQuery[Entity](db.Where("npc_id = ?", npcId), page)
	}
}
