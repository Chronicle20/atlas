// Package tenants administrator provides transaction functions for write operations.
package tenants

import (
	"context"
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

func delete(ctx context.Context, tenantId uuid.UUID) func(db *gorm.DB) error {
	return func(db *gorm.DB) error {
		e, err := byIdEntityProvider(ctx)(tenantId)(db)()
		if err != nil {
			return err
		}

		he := &HistoryEntity{
			TenantId:  e.Id,
			Data:      e.Data,
			CreatedAt: time.Now(),
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
		e, err := byIdEntityProvider(ctx)(tenantId)(db)()
		if err != nil {
			return err
		}

		he := &HistoryEntity{
			TenantId:  e.Id,
			Data:      e.Data,
			CreatedAt: time.Now(),
		}
		err = db.Create(he).Error
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
