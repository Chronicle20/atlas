package drop

import (
	"atlas-drops-information/database"
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
