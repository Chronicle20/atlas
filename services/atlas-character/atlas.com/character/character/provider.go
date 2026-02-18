package character

import (
	"atlas-character/database"

	_map "github.com/Chronicle20/atlas-constants/map"
	"github.com/Chronicle20/atlas-constants/world"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

func getById(tenantId uuid.UUID, characterId uint32) database.EntityProvider[entity] {
	return func(db *gorm.DB) model.Provider[entity] {
		where := map[string]interface{}{"tenant_id": tenantId, "id": characterId}
		return database.Query[entity](db, where)
	}
}

func getForAccountInWorld(tenantId uuid.UUID, accountId uint32, worldId world.Id) database.EntityProvider[[]entity] {
	return func(db *gorm.DB) model.Provider[[]entity] {
		where := map[string]interface{}{"tenant_id": tenantId, "account_id": accountId, "world": worldId}
		return database.SliceQuery[entity](db, where)
	}
}

func getForAccount(tenantId uuid.UUID, accountId uint32) database.EntityProvider[[]entity] {
	return func(db *gorm.DB) model.Provider[[]entity] {
		where := map[string]interface{}{"tenant_id": tenantId, "account_id": accountId}
		return database.SliceQuery[entity](db, where)
	}
}

func getForMapInWorld(tenantId uuid.UUID, worldId world.Id, mapId _map.Id) database.EntityProvider[[]entity] {
	return func(db *gorm.DB) model.Provider[[]entity] {
		return database.SliceQuery[entity](db, &entity{TenantId: tenantId, World: worldId, MapId: mapId})
	}
}

func getForName(tenantId uuid.UUID, name string) database.EntityProvider[[]entity] {
	return func(db *gorm.DB) model.Provider[[]entity] {
		var results []entity
		err := db.Where("tenant_id = ? AND LOWER(name) = LOWER(?)", tenantId, name).Find(&results).Error
		if err != nil {
			return model.ErrorProvider[[]entity](err)
		}
		return model.FixedProvider(results)
	}
}

func getAll(tenantId uuid.UUID) database.EntityProvider[[]entity] {
	return func(db *gorm.DB) model.Provider[[]entity] {
		var results []entity
		err := db.Where("tenant_id = ?", tenantId).Find(&results).Error
		if err != nil {
			return model.ErrorProvider[[]entity](err)
		}
		return model.FixedProvider(results)
	}
}

func modelFromEntity(e entity) (Model, error) {
	r := NewModelBuilder().
		SetId(e.ID).
		SetAccountId(e.AccountId).
		SetWorldId(e.World).
		SetName(e.Name).
		SetLevel(e.Level).
		SetExperience(e.Experience).
		SetGachaponExperience(e.GachaponExperience).
		SetStrength(e.Strength).
		SetDexterity(e.Dexterity).
		SetLuck(e.Luck).
		SetIntelligence(e.Intelligence).
		SetHp(e.Hp).
		SetMp(e.Mp).
		SetMaxHp(e.MaxHp).
		SetMaxMp(e.MaxMp).
		SetMeso(e.Meso).
		SetHpMpUsed(e.HpMpUsed).
		SetJobId(e.JobId).
		SetSkinColor(e.SkinColor).
		SetGender(e.Gender).
		SetFame(e.Fame).
		SetHair(e.Hair).
		SetFace(e.Face).
		SetAp(e.AP).
		SetSp(e.SP).
		SetMapId(e.MapId).
		SetInstance(e.Instance).
		SetSpawnPoint(e.SpawnPoint).
		SetGm(e.GM).
		Build()
	return r, nil
}
