package key

import (
	"atlas-keys/database"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

func byCharacterKeyEntityProvider(tenantId uuid.UUID, characterId uint32, key int32) database.EntityProvider[entity] {
	return func(db *gorm.DB) model.Provider[entity] {
		return database.Query[entity](db, &entity{TenantId: tenantId, CharacterId: characterId, Key: key})
	}
}

func byCharacterIdEntityProvider(tenantId uuid.UUID, characterId uint32) database.EntityProvider[[]entity] {
	return func(db *gorm.DB) model.Provider[[]entity] {
		return database.SliceQuery[entity](db, &entity{TenantId: tenantId, CharacterId: characterId})
	}
}
