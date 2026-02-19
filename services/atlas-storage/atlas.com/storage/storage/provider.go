package storage

import (
	"atlas-storage/asset"

	"github.com/Chronicle20/atlas-constants/world"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

func GetByWorldAndAccountId(l logrus.FieldLogger, db *gorm.DB) func(worldId world.Id, accountId uint32) (Model, error) {
	return func(worldId world.Id, accountId uint32) (Model, error) {
		var e Entity
		err := db.Where("world_id = ? AND account_id = ?", byte(worldId), accountId).First(&e).Error
		if err != nil {
			return Model{}, err
		}

		assets, err := asset.GetByStorageId(db)(e.Id)
		if err != nil {
			l.WithError(err).Warnf("Failed to load assets for storage %s, returning empty assets", e.Id)
			assets = []asset.Model{}
		}

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

func GetByAccountId(l logrus.FieldLogger, db *gorm.DB) func(accountId uint32) ([]Model, error) {
	return func(accountId uint32) ([]Model, error) {
		var entities []Entity
		err := db.Where("account_id = ?", accountId).Find(&entities).Error
		if err != nil {
			return nil, err
		}

		var models []Model
		for _, e := range entities {
			assets, err := asset.GetByStorageId(db)(e.Id)
			if err != nil {
				l.WithError(err).Warnf("Failed to load assets for storage %s, returning empty assets", e.Id)
				assets = []asset.Model{}
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
