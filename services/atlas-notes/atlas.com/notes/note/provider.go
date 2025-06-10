package note

import (
	"github.com/Chronicle20/atlas-model/model"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// getByIdProvider returns a provider for a note by its ID
func getByIdProvider(db *gorm.DB) func(tenantId uuid.UUID) func(id uint32) model.Provider[Model] {
	return func(tenantId uuid.UUID) func(id uint32) model.Provider[Model] {
		return func(id uint32) model.Provider[Model] {
			return func() (Model, error) {
				var entity Entity
				err := db.Where("tenant_id = ? AND id = ?", tenantId, id).First(&entity).Error
				if err != nil {
					return Model{}, err
				}
				return Make(entity)
			}
		}
	}
}

// getByCharacterIdProvider returns a provider for all notes belonging to a character
func getByCharacterIdProvider(db *gorm.DB) func(tenantId uuid.UUID) func(characterId uint32) model.Provider[[]Model] {
	return func(tenantId uuid.UUID) func(characterId uint32) model.Provider[[]Model] {
		return func(characterId uint32) model.Provider[[]Model] {
			return func() ([]Model, error) {
				var entities []Entity
				err := db.Where("tenant_id = ? AND character_id = ?", tenantId, characterId).Find(&entities).Error
				if err != nil {
					return nil, err
				}
				return model.FixedProvider(MakeNotes(entities))()
			}
		}
	}
}

// getAllProvider returns a provider for all notes in a tenant
func getAllProvider(db *gorm.DB) func(tenantId uuid.UUID) model.Provider[[]Model] {
	return func(tenantId uuid.UUID) model.Provider[[]Model] {
		return func() ([]Model, error) {
			var entities []Entity
			err := db.Where("tenant_id = ?", tenantId).Find(&entities).Error
			if err != nil {
				return nil, err
			}
			return model.FixedProvider(MakeNotes(entities))()
		}
	}
}
