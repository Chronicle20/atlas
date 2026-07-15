package searchcount

import (
	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
	database "github.com/Chronicle20/atlas/libs/atlas-database"
	"github.com/Chronicle20/atlas/libs/atlas-model/model"
	"gorm.io/gorm"
)

// getTopByWorld returns the highest-count entities for a world. Uses a
// schema-bound Find so the automatic tenant callback scopes the query.
func getTopByWorld(worldId world.Id, limit int) database.EntityProvider[[]Entity] {
	return func(db *gorm.DB) model.Provider[[]Entity] {
		var results []Entity
		err := db.Where("world_id = ?", worldId).Order("count DESC").Limit(limit).Find(&results).Error
		if err != nil {
			return model.ErrorProvider[[]Entity](err)
		}
		return model.FixedProvider(results)
	}
}
