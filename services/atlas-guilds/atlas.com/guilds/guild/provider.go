package guild

import (
	"strings"

	"gorm.io/gorm"

	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
	database "github.com/Chronicle20/atlas/libs/atlas-database"
	"github.com/Chronicle20/atlas/libs/atlas-model/model"
)

func getAll(page model.Page) database.EntityProvider[model.Paged[Entity]] {
	return func(db *gorm.DB) model.Provider[model.Paged[Entity]] {
		return database.PagedQuery[Entity](db.Preload("Members").Preload("Titles"), page)
	}
}

// escapeLike escapes a substring so it can be embedded in a LIKE pattern
// without its own literal `%`/`_`/`\` characters being interpreted as
// wildcards or an escape character. Must be applied before wrapping the
// value in `%...%`.
func escapeLike(s string) string {
	s = strings.ReplaceAll(s, `\`, `\\`)
	s = strings.ReplaceAll(s, `%`, `\%`)
	s = strings.ReplaceAll(s, `_`, `\_`)
	return s
}

func getByNameLike(name string, page model.Page) database.EntityProvider[model.Paged[Entity]] {
	return func(db *gorm.DB) model.Provider[model.Paged[Entity]] {
		pattern := "%" + escapeLike(name) + "%"
		return database.PagedQuery[Entity](
			db.Preload("Members").Preload("Titles").Where(`LOWER(name) LIKE LOWER(?) ESCAPE '\'`, pattern), page)
	}
}

func getById(id uint32) database.EntityProvider[Entity] {
	return func(db *gorm.DB) model.Provider[Entity] {
		var result Entity
		err := db.Where("id = ?", id).Preload("Members").Preload("Titles").First(&result).Error
		if err != nil {
			return model.ErrorProvider[Entity](err)
		}
		return model.FixedProvider[Entity](result)
	}
}

func getForName(worldId world.Id, name string) database.EntityProvider[[]Entity] {
	return func(db *gorm.DB) model.Provider[[]Entity] {
		var results []Entity
		err := db.Where("world_id = ? AND LOWER(name) = LOWER(?)", worldId, name).Find(&results).Error
		if err != nil {
			return model.ErrorProvider[[]Entity](err)
		}
		return model.FixedProvider(results)
	}
}
