package drop

import (
	"atlas-drops-information/database"
	"github.com/google/uuid"

	"github.com/Chronicle20/atlas-model/model"
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

func getByMonsterId(tenantId uuid.UUID, monsterId uint32) database.EntityProvider[[]entity] {
	return func(db *gorm.DB) model.Provider[[]entity] {
		return database.SliceQuery[entity](db, &entity{TenantId: tenantId, MonsterId: monsterId})
	}
}

func makeDrop(m entity) (Model, error) {
	r := NewMonsterDropBuilder(m.TenantId, m.ID).
		SetMonsterId(m.MonsterId).
		SetItemId(m.ItemId).
		SetMinimumQuantity(m.MinimumQuantity).
		SetMaximumQuantity(m.MaximumQuantity).
		SetQuestId(m.QuestId).
		SetChance(m.Chance).
		Build()
	return r, nil
}
