package card

import (
	"github.com/Chronicle20/atlas/libs/atlas-constants/character"
	"github.com/Chronicle20/atlas/libs/atlas-constants/item"
	database "github.com/Chronicle20/atlas/libs/atlas-database"
	"github.com/Chronicle20/atlas/libs/atlas-model/model"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

func byCharacterIdEntityProvider(tenantId uuid.UUID, characterId character.Id) database.EntityProvider[[]entity] {
	return func(db *gorm.DB) model.Provider[[]entity] {
		return database.SliceQuery[entity](db.Where("tenant_id = ? AND character_id = ?", tenantId, uint32(characterId)), &entity{})
	}
}

func byCharacterIdAndCardIdEntityProvider(tenantId uuid.UUID, characterId character.Id, cardId item.Id) database.EntityProvider[entity] {
	return func(db *gorm.DB) model.Provider[entity] {
		return database.Query[entity](db.Where("tenant_id = ? AND character_id = ? AND card_id = ?", tenantId, uint32(characterId), uint32(cardId)), &entity{})
	}
}

func bySpecialEntityProvider(tenantId uuid.UUID, characterId character.Id, isSpecial bool) database.EntityProvider[[]entity] {
	return func(db *gorm.DB) model.Provider[[]entity] {
		return database.SliceQuery[entity](db.Where("tenant_id = ? AND character_id = ? AND is_special = ?", tenantId, uint32(characterId), isSpecial), &entity{})
	}
}
