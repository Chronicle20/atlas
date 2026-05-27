package drop

import (
	database "github.com/Chronicle20/atlas/libs/atlas-database"

	"gorm.io/gorm"
)

func BulkCreateContinentDrop(db *gorm.DB, continentDrops []Model) error {
	return database.ExecuteTransaction(db, func(tx *gorm.DB) error {
		for _, md := range continentDrops {
			m := &entity{
				TenantId:        md.TenantId(),
				ContinentId:     md.ContinentId(),
				ItemId:          md.ItemId(),
				MinimumQuantity: md.MinimumQuantity(),
				MaximumQuantity: md.MaximumQuantity(),
				QuestId:         md.QuestId(),
				Chance:          md.Chance(),
			}

			err := tx.Create(m).Error
			if err != nil {
				return err
			}
		}
		return nil
	})
}

// DeleteAll deletes all continent drops for the tenant in context.
func DeleteAll(db *gorm.DB) (int64, error) {
	result := db.Unscoped().Where("1 = 1").Delete(&entity{})
	return result.RowsAffected, result.Error
}
