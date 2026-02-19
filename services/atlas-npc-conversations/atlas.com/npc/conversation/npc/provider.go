package npc

import (
	"github.com/Chronicle20/atlas-model/model"
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

// getAllProvider returns a provider for retrieving all conversations
func getAllProvider(db *gorm.DB) model.Provider[[]Entity] {
	return func() ([]Entity, error) {
		var entities []Entity
		result := db.Find(&entities)
		return entities, result.Error
	}
}

// getAllByNpcIdProvider returns a provider for retrieving all conversations for a specific NPC ID
func getAllByNpcIdProvider(npcId uint32) func(db *gorm.DB) model.Provider[[]Entity] {
	return func(db *gorm.DB) model.Provider[[]Entity] {
		return func() ([]Entity, error) {
			var entities []Entity
			result := db.Where("npc_id = ?", npcId).Find(&entities)
			return entities, result.Error
		}
	}
}
