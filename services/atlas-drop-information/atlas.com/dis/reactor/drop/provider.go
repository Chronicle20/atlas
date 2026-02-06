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

func getByReactorId(tenantId uuid.UUID, reactorId uint32) database.EntityProvider[[]entity] {
	return func(db *gorm.DB) model.Provider[[]entity] {
		return database.SliceQuery[entity](db, &entity{TenantId: tenantId, ReactorId: reactorId})
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
