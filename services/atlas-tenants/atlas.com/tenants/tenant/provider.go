package tenant

import (
	"atlas-tenants/scope"
	"context"

	database "github.com/Chronicle20/atlas/libs/atlas-database"

	"github.com/google/uuid"
	"gorm.io/gorm"

	env "github.com/Chronicle20/atlas/libs/atlas-env"
	"github.com/Chronicle20/atlas/libs/atlas-model/model"
)

// GetByIdProvider returns a provider for a tenant by ID, scoped to the
// caller's environment (task-232 FR-7.1). A legacy caller (no environment
// in context) sees every tenant, unfiltered.
func GetByIdProvider(ctx context.Context, id uuid.UUID) database.EntityProvider[Entity] {
	return func(db *gorm.DB) model.Provider[Entity] {
		caller := env.MustFromContext(ctx)
		return database.Query[Entity](scope.Strict(db.WithContext(ctx), caller), map[string]interface{}{"id": id})
	}
}

// getAll returns a paged provider for all tenants, scoped to the caller's
// environment (task-232 FR-7.1).
func getAll(ctx context.Context, page model.Page) database.EntityProvider[model.Paged[Entity]] {
	return func(db *gorm.DB) model.Provider[model.Paged[Entity]] {
		caller := env.MustFromContext(ctx)
		return database.PagedQuery[Entity](scope.Strict(db.WithContext(ctx).Model(&Entity{}), caller), page)
	}
}
