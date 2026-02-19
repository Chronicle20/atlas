package global

import (
	database "github.com/Chronicle20/atlas-database"

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

func DeleteItem(db *gorm.DB, id uint32) error {
	return db.Where(&entity{ID: id}).Delete(&entity{}).Error
}

func DeleteAllForTenant(db *gorm.DB) (int64, error) {
	result := db.Unscoped().Delete(&entity{})
	return result.RowsAffected, result.Error
}
