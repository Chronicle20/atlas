package batch

import (
	"github.com/google/uuid"
	"gorm.io/gorm"

	database "github.com/Chronicle20/atlas/libs/atlas-database"
	"github.com/Chronicle20/atlas/libs/atlas-model/model"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

func byIdEntityProvider(t tenant.Model, id uuid.UUID) database.EntityProvider[Entity] {
	return func(db *gorm.DB) model.Provider[Entity] {
		var result Entity
		err := db.Where("tenant_id = ? AND id = ?", t.Id(), id).First(&result).Error
		if err != nil {
			return model.ErrorProvider[Entity](err)
		}
		return model.FixedProvider[Entity](result)
	}
}

// allPagedEntityProvider pages the tenant's batches oldest first. Batches carry
// no filterable attributes of their own — a caller narrowing to one batch's
// coupons uses coupon.Filters.BatchId instead.
//
// The id tiebreaker matters for the same reason it does on coupons: created_at
// is not unique, and this ordering is paged on.
func allPagedEntityProvider(t tenant.Model, page model.Page) database.EntityProvider[model.Paged[Entity]] {
	return func(db *gorm.DB) model.Provider[model.Paged[Entity]] {
		return database.PagedQuery[Entity](db.Where("tenant_id = ?", t.Id()).Order("created_at, id"), page)
	}
}
