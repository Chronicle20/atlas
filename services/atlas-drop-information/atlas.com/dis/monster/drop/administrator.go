package drop

import (
	database "github.com/Chronicle20/atlas-database"

	"gorm.io/gorm"
)

func BulkCreateMonsterDrop(db *gorm.DB, monsterDrops []Model) error {
	return database.ExecuteTransaction(db, func(tx *gorm.DB) error {
		for _, md := range monsterDrops {
			m := &entity{
				TenantId:        md.TenantId(),
				MonsterId:       md.MonsterId(),
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
