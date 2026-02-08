package asset

import (
	"atlas-cashshop/database"

	"github.com/Chronicle20/atlas-model/model"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

func getByIdProvider(tenantId uuid.UUID) func(id uint32) database.EntityProvider[Entity] {
	return func(id uint32) database.EntityProvider[Entity] {
		return func(db *gorm.DB) model.Provider[Entity] {
			return func() (Entity, error) {
				var entity Entity
				result := db.Where("tenant_id = ? AND id = ?", tenantId, id).First(&entity)
				return entity, result.Error
			}
		}
	}
}

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

func byCashIdProvider(tenantId uuid.UUID, cashId int64) database.EntityProvider[[]Entity] {
	return func(db *gorm.DB) model.Provider[[]Entity] {
		return func() ([]Entity, error) {
			var result []Entity
			err := db.Where(&Entity{TenantId: tenantId, CashId: cashId}).Find(&result).Error
			if err != nil {
				return nil, err
			}
			return result, nil
		}
	}
}
