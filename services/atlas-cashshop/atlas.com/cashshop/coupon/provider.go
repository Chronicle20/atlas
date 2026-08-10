package coupon

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	database "github.com/Chronicle20/atlas/libs/atlas-database"
	"github.com/Chronicle20/atlas/libs/atlas-model/model"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

// byCodeEntityProvider looks a coupon up by its NORMALIZED code. Callers must
// pass Normalize(code); the column stores the normalized form and the unique
// index on (tenant_id, code) is what makes the lookup case-insensitive.
func byCodeEntityProvider(t tenant.Model, code string) database.EntityProvider[Entity] {
	return func(db *gorm.DB) model.Provider[Entity] {
		var result Entity
		err := db.Where("tenant_id = ? AND code = ?", t.Id(), code).First(&result).Error
		if err != nil {
			return model.ErrorProvider[Entity](err)
		}
		return model.FixedProvider[Entity](result)
	}
}

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

// Filters are the optional narrowings the admin list endpoint accepts (PRD §5).
// A nil field means "do not narrow on this" — each predicate is applied only
// when its field is set, so the zero Filters lists every coupon in the tenant.
type Filters struct {
	Code          *string
	Active        *bool
	BatchId       *uuid.UUID
	ExpiresBefore *time.Time
	ExpiresAfter  *time.Time
}

// scopedQuery narrows db to the tenant's coupons matching f.
//
// The expiry predicates deliberately do NOT match rows with a NULL expires_at:
// a coupon that never expires is neither before nor after a given instant, and
// silently including it in "expires before X" would misreport the set an admin
// is about to clean up.
//
// The ORDER BY carries a tiebreaker. `created_at` alone is not unique — two
// coupons written in the same clock tick order arbitrarily — and the admin
// list endpoint PAGES on this ordering, so a nondeterministic sort would drop
// or duplicate rows across page boundaries.
func scopedQuery(db *gorm.DB, t tenant.Model, f Filters) *gorm.DB {
	tx := db.Where("tenant_id = ?", t.Id())
	if f.Code != nil {
		tx = tx.Where("code = ?", *f.Code)
	}
	if f.Active != nil {
		tx = tx.Where("active = ?", *f.Active)
	}
	if f.BatchId != nil {
		tx = tx.Where("batch_id = ?", *f.BatchId)
	}
	if f.ExpiresBefore != nil {
		tx = tx.Where("expires_at IS NOT NULL AND expires_at < ?", *f.ExpiresBefore)
	}
	if f.ExpiresAfter != nil {
		tx = tx.Where("expires_at IS NOT NULL AND expires_at > ?", *f.ExpiresAfter)
	}
	return tx.Order("created_at, id")
}

// allPagedEntityProvider pages the tenant's coupons in SQL (docs/rest-pagination.md §5).
func allPagedEntityProvider(t tenant.Model, f Filters, page model.Page) database.EntityProvider[model.Paged[Entity]] {
	return func(db *gorm.DB) model.Provider[model.Paged[Entity]] {
		return database.PagedQuery[Entity](scopedQuery(db, t, f), page)
	}
}
