package drop

import (
	"atlas-drops-information/database"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

func getAll(tenantId uuid.UUID) database.EntityProvider[[]entity] {
	return func(db *gorm.DB) model.Provider[[]entity] {
		var results []entity
		err := db.Where(&entity{TenantId: tenantId}).Find(&results).Error
		if err != nil {
			return model.ErrorProvider[[]entity](err)
		}
		return model.FixedProvider(results)
	}
}

func makeDrop(m entity) (Model, error) {
	r := Model{
		id:              m.ID,
		continentId:     m.ContinentId,
		itemId:          m.ItemId,
		minimumQuantity: m.MinimumQuantity,
		maximumQuantity: m.MaximumQuantity,
		questId:         m.QuestId,
		chance:          m.Chance,
	}
	return r, nil
}
