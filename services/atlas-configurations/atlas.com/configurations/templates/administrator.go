// Package templates administrator provides transaction functions for write operations.
package templates

import (
	"atlas-configurations/scope"
	"context"
	"encoding/json"

	"github.com/google/uuid"
	"gorm.io/gorm"

	env "github.com/Chronicle20/atlas/libs/atlas-env"
)

// update stays STRICT even though reads use the baseline-fallback overlay
// (byIdEntityProvider resolves to VisibleById): a PR environment may READ
// the baseline's row, but it overrides a template by inserting its own row,
// never by updating the baseline's (design V3). AuthorizeWrite rejects the
// write here if the loaded row belongs to another environment - including
// the baseline itself, when the caller isn't that baseline.
func update(ctx context.Context, templateId uuid.UUID, region string, majorVersion uint16, minorVersion uint16, data json.RawMessage) func(db *gorm.DB) error {
	return func(db *gorm.DB) error {
		e, err := byIdEntityProvider(ctx)(templateId)(db)()
		if err != nil {
			return err
		}

		caller := env.MustFromContext(ctx)
		if err := scope.AuthorizeWrite(caller, env.Id(e.Environment)); err != nil {
			return err
		}

		e.Region = region
		e.MajorVersion = majorVersion
		e.MinorVersion = minorVersion
		e.Data = data
		err = db.Save(e).Error
		if err != nil {
			return err
		}
		return nil
	}
}

func delete(ctx context.Context, templateId uuid.UUID) func(db *gorm.DB) error {
	return func(db *gorm.DB) error {
		e, err := byIdEntityProvider(ctx)(templateId)(db)()
		if err != nil {
			return err
		}

		caller := env.MustFromContext(ctx)
		if err := scope.AuthorizeWrite(caller, env.Id(e.Environment)); err != nil {
			return err
		}

		err = db.Delete(&e).Error
		if err != nil {
			return err
		}
		return nil
	}
}
