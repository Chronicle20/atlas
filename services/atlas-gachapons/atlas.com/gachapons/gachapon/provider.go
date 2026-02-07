package gachapon

import (
	"atlas-gachapons/database"

	"github.com/Chronicle20/atlas-model/model"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

func getAll(tenantId uuid.UUID) database.EntityProvider[[]entity] {
	return func(db *gorm.DB) model.Provider[[]entity] {
		return database.SliceQuery[entity](db, &entity{TenantId: tenantId})
	}
}

func getById(tenantId uuid.UUID, id string) database.EntityProvider[entity] {
	return func(db *gorm.DB) model.Provider[entity] {
		return database.Query[entity](db, &entity{TenantId: tenantId, ID: id})
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
