package ranking

import (
	"gorm.io/gorm"

	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
	database "github.com/Chronicle20/atlas/libs/atlas-database"
	"github.com/Chronicle20/atlas/libs/atlas-model/model"
)

// byCharacterIdEntityProvider reads the ranking row for one character.
// Tenant scoping comes from the GORM query callback on the context-bearing
// db handle passed by the caller.
func byCharacterIdEntityProvider(characterId uint32) database.EntityProvider[Entity] {
	return func(db *gorm.DB) model.Provider[Entity] {
		var result Entity
		err := db.Where("character_id = ?", characterId).First(&result).Error
		if err != nil {
			return model.ErrorProvider[Entity](err)
		}
		return model.FixedProvider(result)
	}
}

// byCharacterIdsEntityProvider reads ranking rows for a set of characters.
func byCharacterIdsEntityProvider(characterIds []uint32) database.EntityProvider[[]Entity] {
	return func(db *gorm.DB) model.Provider[[]Entity] {
		var result []Entity
		err := db.Where("character_id IN ?", characterIds).Find(&result).Error
		if err != nil {
			return model.ErrorProvider[[]Entity](err)
		}
		return model.FixedProvider(result)
	}
}

// allEntityProvider reads every ranking row for the calling tenant.
func allEntityProvider() database.EntityProvider[[]Entity] {
	return func(db *gorm.DB) model.Provider[[]Entity] {
		var result []Entity
		err := db.Find(&result).Error
		if err != nil {
			return model.ErrorProvider[[]Entity](err)
		}
		return model.FixedProvider(result)
	}
}

// cycleEntityProvider reads the single ranking_cycles row for the calling
// tenant. Returns an error (gorm.ErrRecordNotFound) when no cycle has run
// yet.
func cycleEntityProvider() database.EntityProvider[CycleEntity] {
	return func(db *gorm.DB) model.Provider[CycleEntity] {
		var result CycleEntity
		err := db.First(&result).Error
		if err != nil {
			return model.ErrorProvider[CycleEntity](err)
		}
		return model.FixedProvider(result)
	}
}

// byWorldPagedEntityProvider reads one page of ranking rows for a world,
// ordered for a leaderboard: overall_rank ASC for the overall view, or
// job_rank ASC when a job category filter is supplied. Tenant scoping comes
// from the GORM query callback on the context-bearing db handle.
func byWorldPagedEntityProvider(worldId world.Id, jobCategory *uint16, page model.Page) database.EntityProvider[model.Paged[Entity]] {
	return func(db *gorm.DB) model.Provider[model.Paged[Entity]] {
		q := db.Where("world_id = ?", worldId)
		if jobCategory != nil {
			q = q.Where("job_category = ?", *jobCategory).Order("job_rank ASC")
		} else {
			q = q.Order("overall_rank ASC")
		}
		return database.PagedQuery[Entity](q, page)
	}
}
