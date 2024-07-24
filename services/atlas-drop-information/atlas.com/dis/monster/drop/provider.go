package drop

import (
	"atlas-dis/database"

	"github.com/Chronicle20/atlas-model/model"
	"gorm.io/gorm"
)

func getAll() database.EntityProvider[[]entity] {
	return func(db *gorm.DB) model.Provider[[]entity] {
		var results []entity
		err := db.Find(&results).Error
		if err != nil {
			return model.ErrorProvider[[]entity](err)
		}
		return model.FixedProvider(results)
	}
}

func getByMonsterId(monsterId uint32) database.EntityProvider[[]entity] {
	return func(db *gorm.DB) model.Provider[[]entity] {
		return database.SliceQuery[entity](db, &entity{MonsterId: monsterId})
	}
}

func makeDrop(m entity) (Model, error) {
	r := NewMonsterDropBuilder(m.ID).
		SetMonsterId(m.MonsterId).
		SetItemId(m.ItemId).
		SetMinimumQuantity(m.MinimumQuantity).
		SetMaximumQuantity(m.MaximumQuantity).
		SetQuestId(m.QuestId).
		SetChance(m.Chance).
		Build()
	return r, nil
}
