package drop

import (
	database "github.com/Chronicle20/atlas-database"

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

func getByReactorId(reactorId uint32) database.EntityProvider[[]entity] {
	return func(db *gorm.DB) model.Provider[[]entity] {
		return database.SliceQuery[entity](db, &entity{ReactorId: reactorId})
	}
}

func modelFromEntity(m entity) (Model, error) {
	return NewReactorDropBuilder(m.TenantId, m.ID).
		SetReactorId(m.ReactorId).
		SetItemId(m.ItemId).
		SetQuestId(m.QuestId).
		SetChance(m.Chance).
		Build()
}
