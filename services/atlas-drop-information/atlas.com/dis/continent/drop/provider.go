package drop

import (
	"atlas-dis/database"
	"github.com/Chronicle20/atlas-model/model"
	"gorm.io/gorm"
)

func getAll() database.EntitySliceProvider[entity] {
	return func(db *gorm.DB) model.SliceProvider[entity] {
		var results []entity
		err := db.Find(&results).Error
		if err != nil {
			return model.ErrorSliceProvider[entity](err)
		}
		return model.FixedSliceProvider(results)
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
