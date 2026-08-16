package tenants

import (
	"atlas-configurations/scope"
	"context"

	database "github.com/Chronicle20/atlas/libs/atlas-database"

	"github.com/google/uuid"
	"gorm.io/gorm"

	env "github.com/Chronicle20/atlas/libs/atlas-env"
	"github.com/Chronicle20/atlas/libs/atlas-model/model"
)

func getAll(ctx context.Context, page model.Page) database.EntityProvider[model.Paged[Entity]] {
	return func(db *gorm.DB) model.Provider[model.Paged[Entity]] {
		caller := env.MustFromContext(ctx)
		return database.PagedQuery[Entity](scope.Strict(db.WithContext(ctx).Model(&Entity{}), caller), page)
	}
}

func byIdEntityProvider(ctx context.Context) func(id uuid.UUID) database.EntityProvider[Entity] {
	return func(id uuid.UUID) database.EntityProvider[Entity] {
		return func(db *gorm.DB) model.Provider[Entity] {
			caller := env.MustFromContext(ctx)
			var result Entity
			err := scope.Strict(db.WithContext(ctx), caller).Where("id = ?", id).First(&result).Error
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
			caller := env.MustFromContext(ctx)
			var result Entity
			err := scope.Strict(db.WithContext(ctx), caller).
				Where("region = ? AND major_version = ? AND minor_version = ?", region, majorVersion, minorVersion).
				First(&result).Error
			if err != nil {
				return model.ErrorProvider[Entity](err)
			}
			return model.FixedProvider[Entity](result)
		}
	}
}
