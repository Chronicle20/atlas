package drop

import (
	"gorm.io/gorm"
)

func BulkCreateContinentDrop(db *gorm.DB, continentDrops []Model) error {
	tx := db.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()
	if tx.Error != nil {
		return tx.Error
	}

	for _, md := range continentDrops {
		m := &entity{
			ContinentId:     md.ContinentId(),
			ItemId:          md.ItemId(),
			MinimumQuantity: md.MinimumQuantity(),
			MaximumQuantity: md.MaximumQuantity(),
			QuestId:         md.QuestId(),
			Chance:          md.Chance(),
		}

		err := tx.Create(m).Error
		if err != nil {
			tx.Rollback()
			return err
		}
	}
	return tx.Commit().Error
}
