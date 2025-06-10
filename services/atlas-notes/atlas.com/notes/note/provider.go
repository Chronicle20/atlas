package note

import (
	"atlas-notes/database"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// getByIdProvider returns a provider for a note by its ID
func getByIdProvider(tenantId uuid.UUID) func(id uint32) database.EntityProvider[Entity] {
	return func(id uint32) database.EntityProvider[Entity] {
		return func(db *gorm.DB) model.Provider[Entity] {
			var entity Entity
			err := db.Where("tenant_id = ? AND id = ?", tenantId, id).First(&entity).Error
			if err != nil {
				return model.ErrorProvider[Entity](err)
			}
			return model.FixedProvider(entity)
		}
	}
}

// getByCharacterIdProvider returns a provider for all notes belonging to a character
func getByCharacterIdProvider(tenantId uuid.UUID) func(characterId uint32) database.EntityProvider[[]Entity] {
	return func(characterId uint32) database.EntityProvider[[]Entity] {
		return func(db *gorm.DB) model.Provider[[]Entity] {
			var entities []Entity
			err := db.Where("tenant_id = ? AND character_id = ?", tenantId, characterId).Find(&entities).Error
			if err != nil {
				return model.ErrorProvider[[]Entity](err)
			}
			return model.FixedProvider(entities)
		}
	}
}

// getAllProvider returns a provider for all notes in a tenant
func getAllProvider(tenantId uuid.UUID) database.EntityProvider[[]Entity] {
	return func(db *gorm.DB) model.Provider[[]Entity] {
		var entities []Entity
		err := db.Where("tenant_id = ?", tenantId).Find(&entities).Error
		if err != nil {
			return model.ErrorProvider[[]Entity](err)
		}
		return model.FixedProvider(entities)
	}
}
