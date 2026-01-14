package npc

import (
	"github.com/Chronicle20/atlas-model/model"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// getByIdProvider returns a provider for retrieving a conversation by ID
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

// getByNpcIdProvider returns a provider for retrieving a conversation by NPC ID
func getByNpcIdProvider(tenantId uuid.UUID) func(npcId uint32) func(db *gorm.DB) model.Provider[Entity] {
	return func(npcId uint32) func(db *gorm.DB) model.Provider[Entity] {
		return func(db *gorm.DB) model.Provider[Entity] {
			return func() (Entity, error) {
				var entity Entity
				result := db.Where("tenant_id = ? AND npc_id = ?", tenantId, npcId).First(&entity)
				return entity, result.Error
			}
		}
	}
}

// getAllProvider returns a provider for retrieving all conversations
func getAllProvider(tenantId uuid.UUID) func(db *gorm.DB) model.Provider[[]Entity] {
	return func(db *gorm.DB) model.Provider[[]Entity] {
		return func() ([]Entity, error) {
			var entities []Entity
			result := db.Where("tenant_id = ?", tenantId).Find(&entities)
			return entities, result.Error
		}
	}
}

// getAllByNpcIdProvider returns a provider for retrieving all conversations for a specific NPC ID
func getAllByNpcIdProvider(tenantId uuid.UUID) func(npcId uint32) func(db *gorm.DB) model.Provider[[]Entity] {
	return func(npcId uint32) func(db *gorm.DB) model.Provider[[]Entity] {
		return func(db *gorm.DB) model.Provider[[]Entity] {
			return func() ([]Entity, error) {
				var entities []Entity
				result := db.Where("tenant_id = ? AND npc_id = ?", tenantId, npcId).Find(&entities)
				return entities, result.Error
			}
		}
	}
}
