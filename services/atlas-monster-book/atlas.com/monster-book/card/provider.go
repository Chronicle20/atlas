package card

import (
	database "github.com/Chronicle20/atlas/libs/atlas-database"
	"github.com/Chronicle20/atlas/libs/atlas-model/model"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

func byCharacterIdEntityProvider(tenantId uuid.UUID, characterId uint32) database.EntityProvider[[]entity] {
	return func(db *gorm.DB) model.Provider[[]entity] {
		return database.SliceQuery[entity](db.Where("tenant_id = ? AND character_id = ?", tenantId, characterId), &entity{})
	}
}

func byCharacterIdAndCardIdEntityProvider(tenantId uuid.UUID, characterId uint32, cardId uint32) database.EntityProvider[entity] {
	return func(db *gorm.DB) model.Provider[entity] {
		return database.Query[entity](db.Where("tenant_id = ? AND character_id = ? AND card_id = ?", tenantId, characterId, cardId), &entity{})
	}
}

func bySpecialEntityProvider(tenantId uuid.UUID, characterId uint32, isSpecial bool) database.EntityProvider[[]entity] {
	return func(db *gorm.DB) model.Provider[[]entity] {
		return database.SliceQuery[entity](db.Where("tenant_id = ? AND character_id = ? AND is_special = ?", tenantId, characterId, isSpecial), &entity{})
	}
}
