// Package templates administrator provides transaction functions for write operations.
package templates

import (
	"context"
	"encoding/json"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

func update(ctx context.Context, templateId uuid.UUID, region string, majorVersion uint16, minorVersion uint16, data json.RawMessage) func(db *gorm.DB) error {
	return func(db *gorm.DB) error {
		e, err := byIdEntityProvider(ctx)(templateId)(db)()
		if err != nil {
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

		err = db.Delete(&e).Error
		if err != nil {
			return err
		}
		return nil
	}
}
