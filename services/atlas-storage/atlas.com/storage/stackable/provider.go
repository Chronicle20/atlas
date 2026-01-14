package stackable

import (
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

// GetByAssetId retrieves stackable data for an asset
func GetByAssetId(l logrus.FieldLogger, db *gorm.DB) func(assetId uint32) (Model, error) {
	return func(assetId uint32) (Model, error) {
		var e Entity
		err := db.Where("asset_id = ?", assetId).First(&e).Error
		if err != nil {
			return Model{}, err
		}
		return Make(e), nil
	}
}

// GetByAssetIds retrieves stackable data for multiple assets
func GetByAssetIds(l logrus.FieldLogger, db *gorm.DB) func(assetIds []uint32) ([]Model, error) {
	return func(assetIds []uint32) ([]Model, error) {
		if len(assetIds) == 0 {
			return []Model{}, nil
		}
		var entities []Entity
		err := db.Where("asset_id IN ?", assetIds).Find(&entities).Error
		if err != nil {
			return nil, err
		}
		models := make([]Model, 0, len(entities))
		for _, e := range entities {
			models = append(models, Make(e))
		}
		return models, nil
	}
}
