package drop

import (
	"gorm.io/gorm"
)

func BulkCreateMonsterDrop(db *gorm.DB, monsterDrops []Model) error {
	//db.Begin()
	for _, md := range monsterDrops {
		m := &entity{
			MonsterId:       md.MonsterId(),
			ItemId:          md.ItemId(),
			MinimumQuantity: md.MinimumQuantity(),
			MaximumQuantity: md.MaximumQuantity(),
			QuestId:         md.QuestId(),
			Chance:          md.Chance(),
		}

		err := db.Create(m).Error
		if err != nil {
			//db.Rollback()
			return err
		}
	}
	//db.Commit()
	return nil
}
