package drop

import (
	"atlas-drops-information/database"
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
