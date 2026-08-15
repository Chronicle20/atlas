// Package environments administrator provides transaction functions for
// write operations.
package environments

import (
	"context"
	"encoding/json"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

func create(ctx context.Context, id uuid.UUID, name string, baseline string, namespace string, tenant string, overrides json.RawMessage, phase string) func(db *gorm.DB) error {
	return func(db *gorm.DB) error {
		e := &Entity{
			Id:        id,
			Name:      name,
			Baseline:  baseline,
			Namespace: namespace,
			Tenant:    tenant,
			Overrides: overrides,
			Phase:     phase,
		}
		return db.WithContext(ctx).Create(e).Error
	}
}

// update takes every column explicitly, deliberately, rather than a
// partial struct: a Builder-style input makes every field optional at the
// call site, and an Update that only copies the fields it happens to
// receive silently zeroes whatever the caller forgot (see tenants.update /
// services.update for the same shape). Every field environments carries
// must be threaded through here.
func update(ctx context.Context, name string, baseline string, namespace string, tenant string, overrides json.RawMessage, phase string) func(db *gorm.DB) error {
	return func(db *gorm.DB) error {
		e, err := byNameEntityProvider(ctx)(name)(db)()
		if err != nil {
			return err
		}

		e.Baseline = baseline
		e.Namespace = namespace
		e.Tenant = tenant
		e.Overrides = overrides
		e.Phase = phase
		return db.WithContext(ctx).Save(&e).Error
	}
}
