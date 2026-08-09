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

// allEntityProvider lists the tenant's batches oldest first. Batches carry no
// filterable attributes of their own — a caller narrowing to one batch's
// coupons uses coupon.Filters.BatchId instead.
func allEntityProvider(t tenant.Model) database.EntityProvider[[]Entity] {
	return func(db *gorm.DB) model.Provider[[]Entity] {
		var results []Entity
		err := db.Where("tenant_id = ?", t.Id()).Order("created_at").Find(&results).Error
		if err != nil {
			return model.ErrorProvider[[]Entity](err)
		}
		return model.FixedProvider(results)
	}
}
