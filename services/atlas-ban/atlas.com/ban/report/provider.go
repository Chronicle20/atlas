package report

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	database "github.com/Chronicle20/atlas/libs/atlas-database"
	"github.com/Chronicle20/atlas/libs/atlas-model/model"
)

// countClaimsByReporterSince counts the reporter's claims created at or after
// `since` — the rolling-window numerator for the claim quota. Tenant scoping
// comes from the `tenant:query` GORM callback
// (libs/atlas-database/tenant_scope.go), which Count() runs through the same
// way Find() does, so a reporter id shared across tenants is not conflated.
func countClaimsByReporterSince(db *gorm.DB) func(reporterId uint32, since time.Time) (int64, error) {
	return func(reporterId uint32, since time.Time) (int64, error) {
		var count int64
		err := db.Model(&Entity{}).
			Where("kind = ? AND reporter_id = ? AND created_at >= ?", string(KindClaim), reporterId, since).
			Count(&count).Error
		if err != nil {
			return 0, err
		}
		return count, nil
	}
}

func entityById(id uuid.UUID) database.EntityProvider[Entity] {
	return func(db *gorm.DB) model.Provider[Entity] {
		var result Entity
		err := db.Where("id = ?", id).First(&result).Error
		if err != nil {
			return model.ErrorProvider[Entity](err)
		}
		return model.FixedProvider[Entity](result)
	}
}

func entitiesByTenant() database.EntityProvider[[]Entity] {
	return func(db *gorm.DB) model.Provider[[]Entity] {
		var results []Entity
		err := db.Order("created_at DESC").Find(&results).Error
		if err != nil {
			return model.ErrorProvider[[]Entity](err)
		}
		return model.FixedProvider[[]Entity](results)
	}
}

func entitiesByStatus(status Status) database.EntityProvider[[]Entity] {
	return func(db *gorm.DB) model.Provider[[]Entity] {
		var results []Entity
		err := db.Where("status = ?", string(status)).Order("created_at DESC").Find(&results).Error
		if err != nil {
			return model.ErrorProvider[[]Entity](err)
		}
		return model.FixedProvider[[]Entity](results)
	}
}
