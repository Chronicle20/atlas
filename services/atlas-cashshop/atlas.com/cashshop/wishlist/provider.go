package wishlist

import (
	database "github.com/Chronicle20/atlas/libs/atlas-database"

	"github.com/Chronicle20/atlas/libs/atlas-model/model"
	"gorm.io/gorm"
)

// byCharacterIdPagedEntityProvider backs the REST list handler (GET
// /characters/{characterId}/cash-shop/wishlist, task-117).
func byCharacterIdPagedEntityProvider(characterId uint32, page model.Page) database.EntityProvider[model.Paged[Entity]] {
	return func(db *gorm.DB) model.Provider[model.Paged[Entity]] {
		return database.PagedQuery[Entity](db.Where("character_id = ?", characterId), page)
	}
}
