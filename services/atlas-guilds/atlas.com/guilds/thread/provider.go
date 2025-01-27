package thread

import (
	"atlas-guilds/database"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

func getAll(tenantId uuid.UUID, guildId uint32) database.EntityProvider[[]Entity] {
	return func(db *gorm.DB) model.Provider[[]Entity] {
		var result []Entity
		err := db.Order("notice desc").Order("created_at desc").Where(&Entity{TenantId: tenantId, GuildId: guildId}).Preload("Replies").Find(&result).Error
		if err != nil {
			return model.ErrorProvider[[]Entity](err)
		}
		return model.FixedProvider[[]Entity](result)
	}
}

func getById(tenantId uuid.UUID, guildId uint32, threadId uint32) database.EntityProvider[Entity] {
	return func(db *gorm.DB) model.Provider[Entity] {
		var result Entity
		err := db.Where(&Entity{TenantId: tenantId, GuildId: guildId, Id: threadId}).Preload("Replies").First(&result).Error
		if err != nil {
			return model.ErrorProvider[Entity](err)
		}
		return model.FixedProvider[Entity](result)
	}
}
