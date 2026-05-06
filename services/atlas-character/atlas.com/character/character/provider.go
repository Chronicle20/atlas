package character

import (
	database "github.com/Chronicle20/atlas/libs/atlas-database"

	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
	"github.com/Chronicle20/atlas/libs/atlas-model/model"
	"gorm.io/gorm"
)

func getById(characterId uint32) database.EntityProvider[entity] {
	return func(db *gorm.DB) model.Provider[entity] {
		return database.Query[entity](db.Where("id = ?", characterId), &entity{})
	}
}

func getForAccountInWorld(accountId uint32, worldId world.Id) database.EntityProvider[[]entity] {
	return func(db *gorm.DB) model.Provider[[]entity] {
		return database.SliceQuery[entity](db.Where("account_id = ? AND world = ?", accountId, worldId), &entity{})
	}
}

func getForAccount(accountId uint32) database.EntityProvider[[]entity] {
	return func(db *gorm.DB) model.Provider[[]entity] {
		return database.SliceQuery[entity](db.Where("account_id = ?", accountId), &entity{})
	}
}

func getForName(name string) database.EntityProvider[[]entity] {
	return func(db *gorm.DB) model.Provider[[]entity] {
		var results []entity
		err := db.Where("LOWER(name) = LOWER(?)", name).Find(&results).Error
		if err != nil {
			return model.ErrorProvider[[]entity](err)
		}
		return model.FixedProvider(results)
	}
}

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
		SetSpawnPoint(e.SpawnPoint).
		SetGm(e.GM).
		Build()
	return r, nil
}
