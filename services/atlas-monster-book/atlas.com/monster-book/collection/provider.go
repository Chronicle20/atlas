package collection

import (
	"github.com/Chronicle20/atlas/libs/atlas-constants/character"
	database "github.com/Chronicle20/atlas/libs/atlas-database"
	"github.com/Chronicle20/atlas/libs/atlas-model/model"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

func byCharacterIdEntityProvider(tenantId uuid.UUID, characterId character.Id) database.EntityProvider[entity] {
	return func(db *gorm.DB) model.Provider[entity] {
		return database.Query[entity](db.Where("tenant_id = ? AND character_id = ?", tenantId, uint32(characterId)), &entity{})
	}
}
