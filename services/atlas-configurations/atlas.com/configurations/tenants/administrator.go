// Package tenants administrator provides transaction functions for write operations.
package tenants

import (
	"atlas-configurations/scope"
	"context"
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	env "github.com/Chronicle20/atlas/libs/atlas-env"
)

// byIdUnscoped loads a row by id WITHOUT the caller's environment filter,
// unlike byIdEntityProvider. Write authorization needs to see the row's
// actual owning environment to distinguish a cross-environment write
// (ErrCrossEnvironmentWrite) from a genuinely nonexistent id - scope.Strict
// would make both look like gorm.ErrRecordNotFound.
func byIdUnscoped(ctx context.Context, db *gorm.DB, tenantId uuid.UUID) (Entity, error) {
	var e Entity
	err := db.WithContext(ctx).Where("id = ?", tenantId).First(&e).Error
	return e, err
}

func delete(ctx context.Context, tenantId uuid.UUID) func(db *gorm.DB) error {
	return func(db *gorm.DB) error {
		e, err := byIdUnscoped(ctx, db, tenantId)
		if err != nil {
			return err
		}

		caller := env.MustFromContext(ctx)
		if err := scope.AuthorizeWrite(caller, env.Id(e.Environment)); err != nil {
			return err
		}

		he := &HistoryEntity{
			TenantId:    e.Id,
			Data:        e.Data,
			CreatedAt:   time.Now(),
			Environment: e.Environment,
		}
		err = db.Create(he).Error
		if err != nil {
			return err
		}

		err = db.Delete(&e).Error
		if err != nil {
			return err
		}
		return nil
	}
}

func update(ctx context.Context, tenantId uuid.UUID, region string, majorVersion uint16, minorVersion uint16, data json.RawMessage) func(db *gorm.DB) error {
	return func(db *gorm.DB) error {
		e, err := byIdUnscoped(ctx, db, tenantId)
		if err != nil {
			return err
		}

		caller := env.MustFromContext(ctx)
		if err := scope.AuthorizeWrite(caller, env.Id(e.Environment)); err != nil {
			return err
		}

		he := &HistoryEntity{
			TenantId:    e.Id,
			Data:        e.Data,
			CreatedAt:   time.Now(),
			Environment: e.Environment,
		}
		err = db.Create(he).Error
		if err != nil {
			return err
		}

		e.Region = region
		e.MajorVersion = majorVersion
		e.MinorVersion = minorVersion
		e.Data = data
		err = db.Save(&e).Error
		if err != nil {
			return err
		}
		return nil
	}
}
