package scheduling

import (
	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/Chronicle20/atlas/libs/atlas-model/model"
)

// activeStates are the states a dedupe check considers "still live" — a
// PENDING or PROCESSING row blocks a redelivery of the same dedupe key, but a
// CANCELLED or FAILED row does not block a legitimate retry.
var activeStates = []string{StatePending, StateProcessing}

// getByIdProvider provides the row identified by id.
func getByIdProvider(id uuid.UUID) func(db *gorm.DB) model.Provider[Entity] {
	return func(db *gorm.DB) model.Provider[Entity] {
		var result Entity
		err := db.Where("id = ?", id).First(&result).Error
		if err != nil {
			return model.ErrorProvider[Entity](err)
		}
		return model.FixedProvider(result)
	}
}

// getActiveByDedupeKeyProvider provides the PENDING or PROCESSING row
// matching dedupeKey, if one exists.
func getActiveByDedupeKeyProvider(dedupeKey string) func(db *gorm.DB) model.Provider[Entity] {
	return func(db *gorm.DB) model.Provider[Entity] {
		var result Entity
		err := db.Where("dedupe_key = ? AND state IN ?", dedupeKey, activeStates).First(&result).Error
		if err != nil {
			return model.ErrorProvider[Entity](err)
		}
		return model.FixedProvider(result)
	}
}
