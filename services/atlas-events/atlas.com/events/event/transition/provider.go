package transition

import (
	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/Chronicle20/atlas/libs/atlas-model/model"
)

// ByOccurrenceProvider provides the full transition history for an
// occurrence, ordered oldest-first.
func ByOccurrenceProvider(occurrenceId uuid.UUID) func(db *gorm.DB) model.Provider[[]Entity] {
	return func(db *gorm.DB) model.Provider[[]Entity] {
		var results []Entity
		err := db.Where("occurrence_id = ?", occurrenceId).Order("occurred_at ASC").Find(&results).Error
		if err != nil {
			return model.ErrorProvider[[]Entity](err)
		}
		return model.FixedProvider(results)
	}
}
