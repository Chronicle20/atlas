package character

import (
	"atlas-character/database"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

func getById(tenantId uuid.UUID, characterId uint32) database.EntityProvider[entity] {
	return func(db *gorm.DB) model.Provider[entity] {
		return database.Query[entity](db, &entity{TenantId: tenantId, ID: characterId})
	}
}

func getForAccountInWorld(tenantId uuid.UUID, accountId uint32, worldId byte) database.EntityProvider[[]entity] {
	return func(db *gorm.DB) model.Provider[[]entity] {
		where := map[string]interface{}{"tenant_id": tenantId, "account_id": accountId, "world": worldId}
		return database.SliceQuery[entity](db, where)
	}
}

func getForMapInWorld(tenantId uuid.UUID, worldId byte, mapId uint32) database.EntityProvider[[]entity] {
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

func makeCharacter(e entity) (Model, error) {
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
		SetHp(e.HP).
		SetMp(e.MP).
		SetMaxHp(e.MaxHP).
		SetMaxMp(e.MaxMP).
		SetMeso(e.Meso).
		SetHpMpUsed(e.HPMPUsed).
		SetJobId(e.JobId).
		SetSkinColor(e.SkinColor).
		SetGender(e.Gender).
		SetFame(e.Fame).
		SetHair(e.Hair).
		SetFace(e.Face).
		SetAp(e.AP).
		SetSp(e.SP).
		SetMapId(e.MapId).
		SetSpawnPoint(e.SpawnPoint).
		SetGm(e.GM).
		Build()
	return r, nil
}
