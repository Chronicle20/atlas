// Package services administrator provides transaction functions for write operations.
package services

import (
	"context"
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

func create(ctx context.Context, serviceId uuid.UUID, serviceType ServiceType, data json.RawMessage) func(db *gorm.DB) error {
	return func(db *gorm.DB) error {
		// Use raw SQL to ensure the ID is always included, even for nil UUID
		return db.WithContext(ctx).Exec(
			"INSERT INTO services (id, type, data) VALUES (?, ?, ?)",
			serviceId, serviceType, data,
		).Error
	}
}

func update(ctx context.Context, serviceId uuid.UUID, serviceType ServiceType, data json.RawMessage) func(db *gorm.DB) error {
	return func(db *gorm.DB) error {
		e, err := byIdEntityProvider(ctx)(serviceId)(db)()
		if err != nil {
			return err
		}

		he := &HistoryEntity{
			ServiceId: e.Id,
			Type:      e.Type,
			Data:      e.Data,
			CreatedAt: time.Now(),
		}
		err = db.Create(he).Error
		if err != nil {
			return err
		}

		e.Type = serviceType
		e.Data = data
		return db.Save(e).Error
	}
}

func delete(ctx context.Context, serviceId uuid.UUID) func(db *gorm.DB) error {
	return func(db *gorm.DB) error {
		e, err := byIdEntityProvider(ctx)(serviceId)(db)()
		if err != nil {
			return err
		}

		he := &HistoryEntity{
			ServiceId: e.Id,
			Type:      e.Type,
			Data:      e.Data,
			CreatedAt: time.Now(),
		}
		err = db.Create(he).Error
		if err != nil {
			return err
		}

		return db.Delete(&e).Error
	}
}
