package storage

import (
	"atlas-storage/asset"
	"context"

	"github.com/Chronicle20/atlas-constants/world"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

// GetByWorldAndAccountId retrieves storage by world and account with decorated assets
func GetByWorldAndAccountId(l logrus.FieldLogger, db *gorm.DB, tenantId uuid.UUID, ctx context.Context) func(worldId world.Id, accountId uint32) (Model, error) {
	return func(worldId world.Id, accountId uint32) (Model, error) {
		var e Entity
		err := db.Where("tenant_id = ? AND world_id = ? AND account_id = ?", tenantId, byte(worldId), accountId).First(&e).Error
		if err != nil {
			return Model{}, err
		}

		// Create asset processor for decoration
		assetProcessor := asset.NewProcessor(l, ctx, db)

		// Load and decorate assets for this storage
		assets, err := assetProcessor.GetByStorageIdDecorated(tenantId, e.Id)
		if err != nil {
			l.WithError(err).Warnf("Failed to load assets for storage %s, returning empty assets", e.Id)
			assets = []asset.Model[any]{}
		}

		// MustBuild since entities from database are trusted
		return NewModelBuilder().
			SetId(e.Id).
			SetWorldId(world.Id(e.WorldId)).
			SetAccountId(e.AccountId).
			SetCapacity(e.Capacity).
			SetMesos(e.Mesos).
			SetAssets(assets).
			MustBuild(), nil
	}
}

// GetByAccountId retrieves all storages for an account (across all worlds)
func GetByAccountId(l logrus.FieldLogger, db *gorm.DB, tenantId uuid.UUID) func(accountId uint32) ([]Model, error) {
	return func(accountId uint32) ([]Model, error) {
		var entities []Entity
		err := db.Where("tenant_id = ? AND account_id = ?", tenantId, accountId).Find(&entities).Error
		if err != nil {
			return nil, err
		}

		var models []Model
		for _, e := range entities {
			// Load assets for this storage
			assets, err := asset.GetByStorageId(l, db, tenantId)(e.Id)
			if err != nil {
				l.WithError(err).Warnf("Failed to load assets for storage %s, returning empty assets", e.Id)
				assets = []asset.Model[any]{}
			}

			m := NewModelBuilder().
				SetId(e.Id).
				SetWorldId(world.Id(e.WorldId)).
				SetAccountId(e.AccountId).
				SetCapacity(e.Capacity).
				SetMesos(e.Mesos).
				SetAssets(assets).
				MustBuild()
			models = append(models, m)
		}

		return models, nil
	}
}

// GetById retrieves storage by ID
func GetById(l logrus.FieldLogger, db *gorm.DB, tenantId uuid.UUID) func(id uuid.UUID) (Model, error) {
	return func(id uuid.UUID) (Model, error) {
		var e Entity
		err := db.Where("tenant_id = ? AND id = ?", tenantId, id).First(&e).Error
		if err != nil {
			return Model{}, err
		}

		// Load assets for this storage
		assets, err := asset.GetByStorageId(l, db, tenantId)(e.Id)
		if err != nil {
			l.WithError(err).Warnf("Failed to load assets for storage %s, returning empty assets", e.Id)
			assets = []asset.Model[any]{}
		}

		// MustBuild since entities from database are trusted
		return NewModelBuilder().
			SetId(e.Id).
			SetWorldId(world.Id(e.WorldId)).
			SetAccountId(e.AccountId).
			SetCapacity(e.Capacity).
			SetMesos(e.Mesos).
			SetAssets(assets).
			MustBuild(), nil
	}
}
