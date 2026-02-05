package storage

import (
	"github.com/Chronicle20/atlas-constants/world"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

// Create creates a new storage for an account in a world
func Create(l logrus.FieldLogger, db *gorm.DB, tenantId uuid.UUID) func(worldId world.Id, accountId uint32) (Model, error) {
	return func(worldId world.Id, accountId uint32) (Model, error) {
		e := Entity{
			TenantId:  tenantId,
			WorldId:   byte(worldId),
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

// Delete deletes a storage by ID
func Delete(l logrus.FieldLogger, db *gorm.DB, tenantId uuid.UUID) func(id uuid.UUID) error {
	return func(id uuid.UUID) error {
		return db.Where("tenant_id = ? AND id = ?", tenantId, id).Delete(&Entity{}).Error
	}
}
