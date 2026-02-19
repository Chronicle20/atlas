package gachapon

import (
	database "github.com/Chronicle20/atlas-database"

	"github.com/Chronicle20/atlas-model/model"
	"gorm.io/gorm"
)

func getAll() database.EntityProvider[[]entity] {
	return func(db *gorm.DB) model.Provider[[]entity] {
		return database.SliceQuery[entity](db, &entity{})
	}
}

func getById(id string) database.EntityProvider[entity] {
	return func(db *gorm.DB) model.Provider[entity] {
		return database.Query[entity](db, &entity{ID: id})
	}
}

func modelFromEntity(e entity) (Model, error) {
	npcIds := make([]uint32, len(e.NpcIds))
	for i, id := range e.NpcIds {
		npcIds[i] = uint32(id)
	}
	return NewBuilder(e.TenantId, e.ID).
		SetName(e.Name).
		SetNpcIds(npcIds).
		SetCommonWeight(e.CommonWeight).
		SetUncommonWeight(e.UncommonWeight).
		SetRareWeight(e.RareWeight).
		Build()
}
