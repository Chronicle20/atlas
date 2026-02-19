package fame

import (
	"time"

	database "github.com/Chronicle20/atlas-database"
	"github.com/Chronicle20/atlas-model/model"
	"gorm.io/gorm"
)

func byCharacterIdLastMonthEntityProvider(characterId uint32) database.EntityProvider[[]Entity] {
	return func(db *gorm.DB) model.Provider[[]Entity] {
		lastMonth := time.Now().AddDate(0, -1, 0)
		var result []Entity
		err := db.Where("character_id = ? AND created_at >= ?", characterId, lastMonth).Find(&result).Error
		if err != nil {
			return model.ErrorProvider[[]Entity](err)
		}
		return model.FixedProvider[[]Entity](result)
	}
}
