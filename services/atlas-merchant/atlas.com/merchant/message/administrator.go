package message

import (
	"time"

	database "github.com/Chronicle20/atlas/libs/atlas-database"
	"github.com/Chronicle20/atlas/libs/atlas-model/model"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

func create(tenantId uuid.UUID, shopId uuid.UUID, characterId uint32, content string) database.EntityProvider[Entity] {
	return func(db *gorm.DB) model.Provider[Entity] {
		entity := &Entity{
			Id:          uuid.New(),
			TenantId:    tenantId,
			ShopId:      shopId,
			CharacterId: characterId,
			Content:     content,
			SentAt:      time.Now(),
		}
		err := db.Create(entity).Error
		if err != nil {
			return model.ErrorProvider[Entity](err)
		}
		return model.FixedProvider(*entity)
	}
}
