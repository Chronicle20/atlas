package stackable

import (
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

// Create creates stackable data for an asset
func Create(l logrus.FieldLogger, db *gorm.DB) func(assetId uint32, quantity uint32, ownerId uint32, flag uint16) (Model, error) {
	return func(assetId uint32, quantity uint32, ownerId uint32, flag uint16) (Model, error) {
		e := Entity{
			AssetId:  assetId,
			Quantity: quantity,
			OwnerId:  ownerId,
			Flag:     flag,
		}
		err := db.Create(&e).Error
		if err != nil {
			return Model{}, err
		}
		return Make(e), nil
	}
}

// Delete removes stackable data for an asset
func Delete(l logrus.FieldLogger, db *gorm.DB) func(assetId uint32) error {
	return func(assetId uint32) error {
		return db.Where("asset_id = ?", assetId).Delete(&Entity{}).Error
	}
}

// UpdateQuantity updates the quantity of a stackable item
func UpdateQuantity(l logrus.FieldLogger, db *gorm.DB) func(assetId uint32, quantity uint32) error {
	return func(assetId uint32, quantity uint32) error {
		return db.Model(&Entity{}).
			Where("asset_id = ?", assetId).
			Update("quantity", quantity).Error
	}
}
