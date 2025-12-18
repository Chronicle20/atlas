package fame

import (
	"atlas-fame/database"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/google/uuid"
	"gorm.io/gorm"
	"time"
)

func byCharacterIdLastMonthEntityProvider(tenantId uuid.UUID, characterId uint32) database.EntityProvider[[]Entity] {
	return func(db *gorm.DB) model.Provider[[]Entity] {
		lastMonth := time.Now().AddDate(0, -1, 0)
		var result []Entity
		err := db.Where("tenant_id = ? AND character_id = ? AND created_at >= ?", tenantId, characterId, lastMonth).Find(&result).Find(&result).Error
		if err != nil {
			return model.ErrorProvider[[]Entity](err)
		}
		return model.FixedProvider[[]Entity](result)
	}
}
