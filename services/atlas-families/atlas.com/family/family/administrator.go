package family

import (
	"errors"
	"time"

	"github.com/Chronicle20/atlas-model/model"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

// BatchResetResult represents the result of a batch daily rep reset operation
type BatchResetResult struct {
	AffectedCount int64
	ResetTime     time.Time
}

// Administrator-specific errors
var (
	ErrMemberAlreadyExists = errors.New("family member already exists")
	ErrCannotCreateMember  = errors.New("cannot create family member")
	ErrTransactionFailed   = errors.New("transaction failed")
	ErrInvalidActivityType = errors.New("invalid activity type")
	ErrJuniorMustExist     = errors.New("junior must exist before creating family link")
	ErrSeniorMustExist     = errors.New("senior must exist before creating family link")
)

// CreateMember creates a new family member with proper validation
func CreateMember(db *gorm.DB, log logrus.FieldLogger) func(characterId uint32, tenantId uuid.UUID, level uint16, world byte) model.Provider[Entity] {
	return func(characterId uint32, tenantId uuid.UUID, level uint16, world byte) model.Provider[Entity] {
		log.WithFields(logrus.Fields{
			"characterId": characterId,
			"tenantId":    tenantId,
			"level":       level,
			"world":       world,
		}).Info("Creating new family member")

		// Check if member already exists
		exists, err := ExistsProvider(characterId)(db)()
		if err != nil {
			return model.ErrorProvider[Entity](err)
		}
		if exists {
			return model.ErrorProvider[Entity](ErrMemberAlreadyExists)
		}

		// Create new member using builder
		member, err := NewBuilder(characterId, tenantId, level, world).Build()
		if err != nil {
			return model.ErrorProvider[Entity](err)
		}

		// Save to database
		return SaveMember(db, log)(member)
	}
}

// BatchResetDailyRep resets daily reputation for all members
func BatchResetDailyRep(db *gorm.DB, log logrus.FieldLogger) model.Provider[BatchResetResult] {
	return func() (BatchResetResult, error) {
		log.Info("Performing batch daily reputation reset")

		resetTime := time.Now()

		// Reset daily rep for all members
		result := db.Model(&Entity{}).
			Where("daily_rep > 0").
			Updates(map[string]interface{}{
				"daily_rep":  0,
				"updated_at": time.Now(),
			})

		if result.Error != nil {
			return BatchResetResult{}, result.Error
		}
		affectedCount := result.RowsAffected

		return BatchResetResult{
			AffectedCount: affectedCount,
			ResetTime:     resetTime,
		}, nil
	}
}

// SaveMember saves a family member to the database (create or update)
func SaveMember(db *gorm.DB, log logrus.FieldLogger) func(member FamilyMember) model.Provider[Entity] {
	return func(member FamilyMember) model.Provider[Entity] {
		log.WithFields(logrus.Fields{
			"characterId": member.CharacterId(),
			"id":          member.Id(),
		}).Debug("Saving family member to database")

		entity := ToEntity(member)

		// Use Save which handles both create and update
		if err := db.Save(&entity).Error; err != nil {
			return model.ErrorProvider[Entity](err)
		}
		return model.FixedProvider(entity)
	}
}

// DeleteMember deletes a family member from the database
func DeleteMember(db *gorm.DB, log logrus.FieldLogger) func(characterId uint32) model.Provider[bool] {
	return func(characterId uint32) model.Provider[bool] {
		return func() (bool, error) {
			log.WithFields(logrus.Fields{
				"characterId": characterId,
			}).Debug("Deleting family member from database")

			result := db.Where("character_id = ?", characterId).Delete(&Entity{})
			if result.Error != nil {
				return false, result.Error
			}

			return result.RowsAffected > 0, nil
		}
	}
}
