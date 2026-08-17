package tenant

import (
	"atlas-tenants/scope"
	"context"
	"errors"

	database "github.com/Chronicle20/atlas/libs/atlas-database"

	"github.com/google/uuid"
	"gorm.io/gorm"

	env "github.com/Chronicle20/atlas/libs/atlas-env"
)

// byIdUnscoped loads a row by id WITHOUT the caller's environment filter,
// unlike GetByIdProvider. Write authorization needs to see the row's actual
// owning environment to distinguish a cross-environment write
// (ErrCrossEnvironmentWrite) from a genuinely nonexistent id - scope.Strict
// would make both look like gorm.ErrRecordNotFound.
func byIdUnscoped(ctx context.Context, db *gorm.DB, id uuid.UUID) (Entity, error) {
	var e Entity
	err := db.WithContext(ctx).Where("id = ?", id).First(&e).Error
	return e, err
}

// CreateTenant creates a new tenant in the database. Environment is
// server-owned (task-232 FR-7.3): ProcessorImpl.Create sets e.Environment
// from the caller's context before FromModel builds this Entity, so this
// function persists it as given rather than deriving it itself - that
// keeps it usable for direct, already-scoped seeding (tests, migrations).
func CreateTenant(db *gorm.DB, e Entity) error {
	return database.ExecuteTransaction(db, func(tx *gorm.DB) error {
		return tx.Create(&e).Error
	})
}

// UpdateTenant updates an existing tenant in the database. Rejects the
// write with scope.ErrCrossEnvironmentWrite when the caller's environment
// does not match the persisted row's environment (task-232 FR-7.1).
func UpdateTenant(ctx context.Context, db *gorm.DB, e Entity) error {
	existing, err := byIdUnscoped(ctx, db, e.ID)
	if err != nil {
		return err
	}

	caller := env.MustFromContext(ctx)
	if err := scope.AuthorizeWrite(caller, env.Id(existing.Environment)); err != nil {
		return err
	}

	return database.ExecuteTransaction(db, func(tx *gorm.DB) error {
		return tx.Save(&e).Error
	})
}

// DeleteTenant deletes a tenant from the database. Rejects the delete with
// scope.ErrCrossEnvironmentWrite when the caller's environment does not
// match the persisted row's environment (task-232 FR-7.1).
func DeleteTenant(ctx context.Context, db *gorm.DB, id uuid.UUID) error {
	e, err := byIdUnscoped(ctx, db, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errors.New("tenant not found")
		}
		return err
	}

	caller := env.MustFromContext(ctx)
	if err := scope.AuthorizeWrite(caller, env.Id(e.Environment)); err != nil {
		return err
	}

	return database.ExecuteTransaction(db, func(tx *gorm.DB) error {
		return tx.Delete(&e).Error
	})
}
