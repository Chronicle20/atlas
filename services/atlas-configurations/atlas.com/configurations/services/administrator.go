// Package services administrator provides transaction functions for write operations.
package services

import (
	"atlas-configurations/scope"
	"context"
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	env "github.com/Chronicle20/atlas/libs/atlas-env"
)

func create(ctx context.Context, serviceId uuid.UUID, serviceType ServiceType, data json.RawMessage) func(db *gorm.DB) error {
	return func(db *gorm.DB) error {
		// Use raw SQL to ensure the ID is always included, even for nil UUID
		return db.WithContext(ctx).Exec(
			"INSERT INTO services (id, type, data, environment) VALUES (?, ?, ?, ?)",
			serviceId, serviceType, data, string(env.MustFromContext(ctx)),
		).Error
	}
}

// byIdUnscoped loads a row by id WITHOUT the caller's environment filter,
// unlike byIdEntityProvider. Write authorization needs to see the row's
// actual owning environment to distinguish a cross-environment write
// (ErrCrossEnvironmentWrite) from a genuinely nonexistent id - scope.Strict
// would make both look like gorm.ErrRecordNotFound.
func byIdUnscoped(ctx context.Context, db *gorm.DB, serviceId uuid.UUID) (Entity, error) {
	var e Entity
	err := db.WithContext(ctx).Where("id = ?", serviceId).First(&e).Error
	return e, err
}

func update(ctx context.Context, serviceId uuid.UUID, serviceType ServiceType, data json.RawMessage) func(db *gorm.DB) error {
	return func(db *gorm.DB) error {
		e, err := byIdUnscoped(ctx, db, serviceId)
		if err != nil {
			return err
		}

		caller := env.MustFromContext(ctx)
		if err := scope.AuthorizeWrite(caller, env.Id(e.Environment)); err != nil {
			return err
		}

		he := &HistoryEntity{
			ServiceId:   e.Id,
			Type:        e.Type,
			Data:        e.Data,
			CreatedAt:   time.Now(),
			Environment: e.Environment,
		}
		err = db.Create(he).Error
		if err != nil {
			return err
		}

		e.Type = serviceType
		e.Data = data
		return db.Save(&e).Error
	}
}

func delete(ctx context.Context, serviceId uuid.UUID) func(db *gorm.DB) error {
	return func(db *gorm.DB) error {
		e, err := byIdUnscoped(ctx, db, serviceId)
		if err != nil {
			return err
		}

		caller := env.MustFromContext(ctx)
		if err := scope.AuthorizeWrite(caller, env.Id(e.Environment)); err != nil {
			return err
		}

		he := &HistoryEntity{
			ServiceId:   e.Id,
			Type:        e.Type,
			Data:        e.Data,
			CreatedAt:   time.Now(),
			Environment: e.Environment,
		}
		err = db.Create(he).Error
		if err != nil {
			return err
		}

		return db.Delete(&e).Error
	}
}
