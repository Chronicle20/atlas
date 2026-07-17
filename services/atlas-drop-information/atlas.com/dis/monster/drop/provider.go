package drop

import (
	database "github.com/Chronicle20/atlas/libs/atlas-database"

	"gorm.io/gorm"

	"github.com/Chronicle20/atlas/libs/atlas-model/model"
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

func getByMonsterIdPagedProvider(monsterId uint32, page model.Page) database.EntityProvider[model.Paged[entity]] {
	return func(db *gorm.DB) model.Provider[model.Paged[entity]] {
		return database.PagedQuery[entity](db.Where(&entity{MonsterId: monsterId}), page)
	}
}

func getByItemIdPagedProvider(itemId uint32, page model.Page) database.EntityProvider[model.Paged[entity]] {
	return func(db *gorm.DB) model.Provider[model.Paged[entity]] {
		return database.PagedQuery[entity](db.Where(&entity{ItemId: itemId}), page)
	}
}

func modelFromEntity(m entity) (Model, error) {
	return NewMonsterDropBuilder(m.TenantId, m.ID).
		SetMonsterId(m.MonsterId).
		SetItemId(m.ItemId).
		SetMinimumQuantity(m.MinimumQuantity).
		SetMaximumQuantity(m.MaximumQuantity).
		SetQuestId(m.QuestId).
		SetChance(m.Chance).
		Build()
}
