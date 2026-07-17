package global

import (
	database "github.com/Chronicle20/atlas/libs/atlas-database"

	"gorm.io/gorm"
)

func CreateItem(db *gorm.DB, m Model) error {
	e := &entity{
		TenantId: m.TenantId(),
		ItemId:   m.ItemId(),
		Quantity: m.Quantity(),
		Tier:     m.Tier(),
	}
	return db.Create(e).Error
}

func BulkCreateItem(db *gorm.DB, models []Model) error {
	return database.ExecuteTransaction(db, func(tx *gorm.DB) error {
		for _, m := range models {
			if err := CreateItem(tx, m); err != nil {
				return err
			}
		}
		return nil
	})
}

func UpdateItem(db *gorm.DB, id uint32, itemId uint32, quantity uint32, tier string) error {
	result := db.Model(&entity{}).
		Where(&entity{ID: id}).
		Updates(map[string]interface{}{
			"item_id":  itemId,
			"quantity": quantity,
			"tier":     tier,
		})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}

func DeleteItem(db *gorm.DB, id uint32) error {
	return db.Where(&entity{ID: id}).Delete(&entity{}).Error
}

func DeleteAllForTenant(db *gorm.DB) (int64, error) {
	result := db.Unscoped().Delete(&entity{})
	return result.RowsAffected, result.Error
}
