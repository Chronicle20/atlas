package storage

import (
	"atlas-storage/asset"
	"context"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

// GetByWorldAndAccountId retrieves storage by world and account with decorated assets
func GetByWorldAndAccountId(l logrus.FieldLogger, db *gorm.DB, tenantId uuid.UUID, ctx context.Context) func(worldId byte, accountId uint32) (Model, error) {
	return func(worldId byte, accountId uint32) (Model, error) {
		var e Entity
		err := db.Where("tenant_id = ? AND world_id = ? AND account_id = ?", tenantId, worldId, accountId).First(&e).Error
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

		return NewModelBuilder().
			SetId(e.Id).
			SetWorldId(e.WorldId).
			SetAccountId(e.AccountId).
			SetCapacity(e.Capacity).
			SetMesos(e.Mesos).
			SetAssets(assets).
			Build(), nil
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

		return NewModelBuilder().
			SetId(e.Id).
			SetWorldId(e.WorldId).
			SetAccountId(e.AccountId).
			SetCapacity(e.Capacity).
			SetMesos(e.Mesos).
			SetAssets(assets).
			Build(), nil
	}
}

// Create creates a new storage for an account in a world
func Create(l logrus.FieldLogger, db *gorm.DB, tenantId uuid.UUID) func(worldId byte, accountId uint32) (Model, error) {
	return func(worldId byte, accountId uint32) (Model, error) {
		e := Entity{
			TenantId:  tenantId,
			WorldId:   worldId,
			AccountId: accountId,
			Capacity:  4,
			Mesos:     0,
		}
		err := db.Create(&e).Error
		if err != nil {
			return Model{}, err
		}
		return Make(e), nil
	}
}

// UpdateMesos updates the mesos amount in storage
func UpdateMesos(l logrus.FieldLogger, db *gorm.DB, tenantId uuid.UUID) func(id uuid.UUID, mesos uint32) error {
	return func(id uuid.UUID, mesos uint32) error {
		return db.Model(&Entity{}).
			Where("tenant_id = ? AND id = ?", tenantId, id).
			Update("mesos", mesos).Error
	}
}

// UpdateCapacity updates the storage capacity
func UpdateCapacity(l logrus.FieldLogger, db *gorm.DB, tenantId uuid.UUID) func(id uuid.UUID, capacity uint32) error {
	return func(id uuid.UUID, capacity uint32) error {
		return db.Model(&Entity{}).
			Where("tenant_id = ? AND id = ?", tenantId, id).
			Update("capacity", capacity).Error
	}
}
