package visit

import (
	"github.com/google/uuid"
	"gorm.io/gorm"

	database "github.com/Chronicle20/atlas/libs/atlas-database"
	"github.com/Chronicle20/atlas/libs/atlas-model/model"
)

// getByShopIdPaged is the paged form of the per-shop visit-log query,
// backing the GET /merchants/{shopId}/visits list route (task-117). The
// prior unpaged form had no internal caller (visit recording upserts), so
// it is deleted rather than kept alongside a paged sibling, per the Group A
// "delete, don't shadow" convention. PagedQuery appends the surrogate-uuid
// PK as tiebreaker after the count ordering so pages form a total order.
func getByShopIdPaged(shopId uuid.UUID, page model.Page) database.EntityProvider[model.Paged[Entity]] {
	return func(db *gorm.DB) model.Provider[model.Paged[Entity]] {
		return database.PagedQuery[Entity](db.Where("shop_id = ?", shopId).Order("count DESC"), page)
	}
}
