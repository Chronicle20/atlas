package drop

import (
	"atlas-dis/database"
	"atlas-dis/model"
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

func getByMonsterId(monsterId uint32) database.EntitySliceProvider[entity] {
	return func(db *gorm.DB) model.SliceProvider[entity] {
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
