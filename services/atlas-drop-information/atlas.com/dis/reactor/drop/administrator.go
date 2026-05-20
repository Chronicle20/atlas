package drop

import (
	database "github.com/Chronicle20/atlas/libs/atlas-database"

	"gorm.io/gorm"
)

func BulkCreateReactorDrop(db *gorm.DB, reactorDrops []Model) error {
	return database.ExecuteTransaction(db, func(tx *gorm.DB) error {
		for _, rd := range reactorDrops {
			m := &entity{
				TenantId:  rd.TenantId(),
				ReactorId: rd.ReactorId(),
				ItemId:    rd.ItemId(),
				QuestId:   rd.QuestId(),
				Chance:    rd.Chance(),
			}

			err := tx.Create(m).Error
			if err != nil {
				return err
			}
		}
		return nil
	})
}

// DeleteAll deletes all reactor drops for the tenant in context.
func DeleteAll(db *gorm.DB) (int64, error) {
	result := db.Unscoped().Where("1 = 1").Delete(&entity{})
	return result.RowsAffected, result.Error
}
