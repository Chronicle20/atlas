package asset

import (
	"atlas-cashshop/database"
	"atlas-cashshop/item"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// getByIdProvider retrieves an asset by ID
func getByIdProvider(tenantId uuid.UUID) func(id uuid.UUID) database.EntityProvider[Entity] {
	return func(id uuid.UUID) database.EntityProvider[Entity] {
		return func(db *gorm.DB) model.Provider[Entity] {
			return func() (Entity, error) {
				var entity Entity
				result := db.Where("id = ? AND tenant_id = ?", id, tenantId).First(&entity)
				return entity, result.Error
			}
		}
	}
}

// getByCompartmentIdProvider retrieves all assets for a compartment
func getByCompartmentIdProvider(tenantId uuid.UUID) func(compartmentId uuid.UUID) database.EntityProvider[[]Entity] {
	return func(compartmentId uuid.UUID) database.EntityProvider[[]Entity] {
		return func(db *gorm.DB) model.Provider[[]Entity] {
			return func() ([]Entity, error) {
				var entities []Entity
				result := db.Where("compartment_id = ? AND tenant_id = ?", compartmentId, tenantId).Find(&entities)
				return entities, result.Error
			}
		}
	}
}

// ByIdProvider retrieves an asset by ID
func ByIdProvider(tenantId uuid.UUID) func(id uuid.UUID) func(itemProvider func(itemId uint32) model.Provider[item.Model]) func(db *gorm.DB) model.Provider[Model] {
	return func(id uuid.UUID) func(itemProvider func(itemId uint32) model.Provider[item.Model]) func(db *gorm.DB) model.Provider[Model] {
		return func(itemProvider func(itemId uint32) model.Provider[item.Model]) func(db *gorm.DB) model.Provider[Model] {
			return func(db *gorm.DB) model.Provider[Model] {
				return func() (Model, error) {
					entity, err := getByIdProvider(tenantId)(id)(db)()
					if err != nil {
						return Model{}, err
					}
					
					item, err := itemProvider(entity.ItemId)()
					if err != nil {
						return Model{}, err
					}
					
					return Make(entity, item)
				}
			}
		}
	}
}

// ByCompartmentIdProvider retrieves all assets for a compartment
func ByCompartmentIdProvider(tenantId uuid.UUID) func(compartmentId uuid.UUID) func(itemProvider func(itemId uint32) model.Provider[item.Model]) func(db *gorm.DB) model.Provider[[]Model] {
	return func(compartmentId uuid.UUID) func(itemProvider func(itemId uint32) model.Provider[item.Model]) func(db *gorm.DB) model.Provider[[]Model] {
		return func(itemProvider func(itemId uint32) model.Provider[item.Model]) func(db *gorm.DB) model.Provider[[]Model] {
			return func(db *gorm.DB) model.Provider[[]Model] {
				return func() ([]Model, error) {
					entities, err := getByCompartmentIdProvider(tenantId)(compartmentId)(db)()
					if err != nil {
						return []Model{}, err
					}
					
					models := make([]Model, 0, len(entities))
					for _, entity := range entities {
						item, err := itemProvider(entity.ItemId)()
						if err != nil {
							return []Model{}, err
						}
						
						model, err := Make(entity, item)
						if err != nil {
							return []Model{}, err
						}
						
						models = append(models, model)
					}
					
					return models, nil
				}
			}
		}
	}
}