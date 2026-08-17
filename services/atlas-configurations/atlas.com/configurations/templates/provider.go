package templates

import (
	"context"

	database "github.com/Chronicle20/atlas/libs/atlas-database"

	"github.com/google/uuid"
	"gorm.io/gorm"

	env "github.com/Chronicle20/atlas/libs/atlas-env"
	"github.com/Chronicle20/atlas/libs/atlas-model/model"
)

// callerBaseline resolves ctx's caller environment and its registered
// baseline. With no registry populated (legacy mode) or an unknown/legacy
// caller, ok is false and baseline is the empty Id, which degrades every
// overlay helper to scope.Strict - today's unfiltered behaviour.
func callerBaseline(ctx context.Context) (caller env.Id, baseline env.Id) {
	caller = env.MustFromContext(ctx)
	baseline, _ = env.CurrentRegistry().BaselineOf(caller)
	return caller, baseline
}

func getAll(ctx context.Context, page model.Page) database.EntityProvider[model.Paged[Entity]] {
	return func(db *gorm.DB) model.Provider[model.Paged[Entity]] {
		caller, baseline := callerBaseline(ctx)
		return database.PagedQuery[Entity](OverlayCollection(db.WithContext(ctx).Model(&Entity{}), caller, baseline), page)
	}
}

func byIdEntityProvider(ctx context.Context) func(id uuid.UUID) database.EntityProvider[Entity] {
	return func(id uuid.UUID) database.EntityProvider[Entity] {
		return func(db *gorm.DB) model.Provider[Entity] {
			caller, baseline := callerBaseline(ctx)
			var result Entity
			err := VisibleById(db.WithContext(ctx), caller, baseline).Where("id = ?", id).First(&result).Error
			if err != nil {
				return model.ErrorProvider[Entity](err)
			}
			return model.FixedProvider[Entity](result)
		}
	}
}

func byRegionVersionEntityProvider(ctx context.Context) func(region string, majorVersion uint16, minorVersion uint16) database.EntityProvider[Entity] {
	return func(region string, majorVersion uint16, minorVersion uint16) database.EntityProvider[Entity] {
		return func(db *gorm.DB) model.Provider[Entity] {
			caller, baseline := callerBaseline(ctx)
			var result Entity
			err := OverlaySingle(db.WithContext(ctx), caller, baseline).
				Where("region = ? AND major_version = ? AND minor_version = ?", region, majorVersion, minorVersion).
				First(&result).Error
			if err != nil {
				return model.ErrorProvider[Entity](err)
			}
			return model.FixedProvider[Entity](result)
		}
	}
}
