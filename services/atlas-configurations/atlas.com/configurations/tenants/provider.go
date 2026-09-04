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
			return database.Query[Entity](scope.Strict(db.WithContext(ctx), caller), map[string]any{"id": id})
		}
	}
}

func byRegionVersionEntityProvider(ctx context.Context) func(region string, majorVersion uint16, minorVersion uint16) database.EntityProvider[Entity] {
	return func(region string, majorVersion uint16, minorVersion uint16) database.EntityProvider[Entity] {
		return func(db *gorm.DB) model.Provider[Entity] {
			caller := env.MustFromContext(ctx)
			query := map[string]any{
				"region":        region,
				"major_version": majorVersion,
				"minor_version": minorVersion,
			}
			return database.Query[Entity](scope.Strict(db.WithContext(ctx), caller), query)
		}
	}
}
