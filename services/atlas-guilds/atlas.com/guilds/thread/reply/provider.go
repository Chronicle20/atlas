package reply

import (
	database "github.com/Chronicle20/atlas-database"
	"github.com/Chronicle20/atlas-model/model"
	"gorm.io/gorm"
)

func getByThreadId(threadId uint32) database.EntityProvider[[]Entity] {
	return func(db *gorm.DB) model.Provider[[]Entity] {
		var results []Entity
		err := db.Where("thread_id = ?", threadId).Find(&results).Error
		if err != nil {
			return model.ErrorProvider[[]Entity](err)
		}
		return model.FixedProvider(results)
	}
}
