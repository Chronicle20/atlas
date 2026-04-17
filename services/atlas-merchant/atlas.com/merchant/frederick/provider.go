package frederick

import (
	database "github.com/Chronicle20/atlas/libs/atlas-database"
	"github.com/Chronicle20/atlas/libs/atlas-model/model"
	"gorm.io/gorm"
)

func getItemsByCharacterId(characterId uint32) database.EntityProvider[[]ItemEntity] {
	return func(db *gorm.DB) model.Provider[[]ItemEntity] {
		var results []ItemEntity
		err := db.Where("character_id = ?", characterId).Find(&results).Error
		if err != nil {
			return model.ErrorProvider[[]ItemEntity](err)
		}
		return model.FixedProvider(results)
	}
}

func getMesosByCharacterId(characterId uint32) database.EntityProvider[[]MesoEntity] {
	return func(db *gorm.DB) model.Provider[[]MesoEntity] {
		var results []MesoEntity
		err := db.Where("character_id = ?", characterId).Find(&results).Error
		if err != nil {
			return model.ErrorProvider[[]MesoEntity](err)
		}
		return model.FixedProvider(results)
	}
}

// HasItemsOrMesos is the exported provider for use by other packages.
func HasItemsOrMesos(characterId uint32) database.EntityProvider[bool] {
	return hasItemsOrMesos(characterId)
}

func hasItemsOrMesos(characterId uint32) database.EntityProvider[bool] {
	return func(db *gorm.DB) model.Provider[bool] {
		var itemCount int64
		err := db.Model(&ItemEntity{}).Where("character_id = ?", characterId).Count(&itemCount).Error
		if err != nil {
			return model.ErrorProvider[bool](err)
		}
		if itemCount > 0 {
			return model.FixedProvider(true)
		}
		var mesoCount int64
		err = db.Model(&MesoEntity{}).Where("character_id = ?", characterId).Count(&mesoCount).Error
		if err != nil {
			return model.ErrorProvider[bool](err)
		}
		return model.FixedProvider(mesoCount > 0)
	}
}
