package thread

import (
	"gorm.io/gorm"

	database "github.com/Chronicle20/atlas/libs/atlas-database"
	"github.com/Chronicle20/atlas/libs/atlas-model/model"
)

// getAll returns a paged provider of threads for a guild, preserving the
// notice-desc / created_at-desc ordering the unpaginated query used (the
// guild BBS notice thread must stay first-in-list -- see
// socket/writer/guild_bbs.go's GuildBBSThreadsBody in atlas-channel, which
// treats ts[0] as the candidate notice thread). database.PagedQuery appends
// a primary-key tie-break after these explicit orderings for stable paging.
func getAll(guildId uint32, page model.Page) database.EntityProvider[model.Paged[Entity]] {
	return func(db *gorm.DB) model.Provider[model.Paged[Entity]] {
		return database.PagedQuery[Entity](db.Order("notice desc").Order("created_at desc").Where("guild_id = ?", guildId).Preload("Replies"), page)
	}
}

func getById(guildId uint32, threadId uint32) database.EntityProvider[Entity] {
	return func(db *gorm.DB) model.Provider[Entity] {
		var result Entity
		err := db.Where("guild_id = ? AND id = ?", guildId, threadId).Preload("Replies").First(&result).Error
		if err != nil {
			return model.ErrorProvider[Entity](err)
		}
		return model.FixedProvider[Entity](result)
	}
}
