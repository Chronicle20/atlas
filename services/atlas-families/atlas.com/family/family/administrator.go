package family

import (
	"errors"
	"time"

	"github.com/Chronicle20/atlas-model/model"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

// Administrator interface defines high-level data coordination operations
type Administrator interface {
	// High-level coordination operations
	CreateMember(db *gorm.DB, log logrus.FieldLogger) func(characterId uint32, tenantId uuid.UUID, level uint16, world byte) model.Provider[FamilyMember]
	AddJuniorWithValidation(db *gorm.DB, log logrus.FieldLogger) func(seniorId uint32, juniorId uint32, tenantId uuid.UUID, juniorLevel uint16, juniorWorld byte) model.Provider[[]FamilyMember]
	RemoveMemberCascade(db *gorm.DB, log logrus.FieldLogger) func(characterId uint32, reason string) model.Provider[[]FamilyMember]
	DissolveSubtree(db *gorm.DB, log logrus.FieldLogger) func(seniorId uint32, reason string) model.Provider[[]FamilyMember]
	AwardRepToSenior(db *gorm.DB, log logrus.FieldLogger) func(juniorId uint32, amount uint32, source string) model.Provider[FamilyMember]
	ProcessRepActivity(db *gorm.DB, log logrus.FieldLogger) func(juniorId uint32, activityType string, value uint32) model.Provider[FamilyMember]
	BatchResetDailyRep(db *gorm.DB, log logrus.FieldLogger) func() model.Provider[BatchResetResult]
	EnsureMemberExists(db *gorm.DB, log logrus.FieldLogger) func(characterId uint32, tenantId uuid.UUID, level uint16, world byte) model.Provider[FamilyMember]

	// Database persistence operations
	SaveMember(db *gorm.DB, log logrus.FieldLogger) func(member FamilyMember) model.Provider[FamilyMember]
	DeleteMember(db *gorm.DB, log logrus.FieldLogger) func(characterId uint32) model.Provider[bool]
}

// AdministratorImpl implements the Administrator interface
type AdministratorImpl struct{}

// NewAdministrator creates a new administrator instance
func NewAdministrator() Administrator {
	return &AdministratorImpl{}
}

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
func (a *AdministratorImpl) CreateMember(db *gorm.DB, log logrus.FieldLogger) func(characterId uint32, tenantId uuid.UUID, level uint16, world byte) model.Provider[FamilyMember] {
	return func(characterId uint32, tenantId uuid.UUID, level uint16, world byte) model.Provider[FamilyMember] {
		return func() (FamilyMember, error) {
			log.WithFields(logrus.Fields{
				"characterId": characterId,
				"tenantId":    tenantId,
				"level":       level,
				"world":       world,
			}).Info("Creating new family member")

			// Check if member already exists
			exists, err := ExistsProvider(characterId)(db)()
			if err != nil {
				return FamilyMember{}, err
			}
			if exists {
				return FamilyMember{}, ErrMemberAlreadyExists
			}

			// Create new member using builder
			member, err := NewBuilder(characterId, tenantId, level, world).Build()
			if err != nil {
				return FamilyMember{}, err
			}

			// Save to database
			return a.SaveMember(db, log)(member)()
		}
	}
}

// AddJuniorWithValidation adds a junior with full validation and member creation if needed
func (a *AdministratorImpl) AddJuniorWithValidation(db *gorm.DB, log logrus.FieldLogger) func(seniorId uint32, juniorId uint32, tenantId uuid.UUID, juniorLevel uint16, juniorWorld byte) model.Provider[[]FamilyMember] {
	return func(seniorId uint32, juniorId uint32, tenantId uuid.UUID, juniorLevel uint16, juniorWorld byte) model.Provider[[]FamilyMember] {
		return func() ([]FamilyMember, error) {
			log.WithFields(logrus.Fields{
				"seniorId":    seniorId,
				"juniorId":    juniorId,
				"tenantId":    tenantId,
				"juniorLevel": juniorLevel,
				"juniorWorld": juniorWorld,
			}).Info("Adding junior with validation")

			err := db.Transaction(func(tx *gorm.DB) error {
				// Ensure senior exists
				_, err := GetByCharacterIdProvider(seniorId)(tx)()
				if err != nil {
					return ErrSeniorMustExist
				}

				// Ensure junior exists or create
				junior, err := GetByCharacterIdProvider(juniorId)(tx)()
				if err != nil {
					if errors.Is(err, ErrMemberNotFound) {
						// Create junior member
						junior, err = a.CreateMember(tx, log)(juniorId, tenantId, juniorLevel, juniorWorld)()
						if err != nil {
							return err
						}
					} else {
						return err
					}
				}

				// Update junior's location to match provided data
				if junior.Level() != juniorLevel || junior.World() != juniorWorld {
					updatedJunior, err := junior.Builder().
						SetLevel(juniorLevel).
						SetWorld(juniorWorld).
						Touch().
						Build()
					if err != nil {
						return err
					}

					if _, err := a.SaveMember(tx, log)(updatedJunior)(); err != nil {
						return err
					}
				}

				// Update senior to add junior
				senior, err := GetByCharacterIdProvider(seniorId)(tx)()
				if err != nil {
					return err
				}

				updatedSenior, err := senior.Builder().
					AddJunior(juniorId).
					Touch().
					Build()
				if err != nil {
					return err
				}

				if _, err := a.SaveMember(tx, log)(updatedSenior)(); err != nil {
					return err
				}

				// Update junior to set senior
				junior, err = GetByCharacterIdProvider(juniorId)(tx)()
				if err != nil {
					return err
				}

				updatedJuniorWithSenior, err := junior.Builder().
					SetSeniorId(seniorId).
					Touch().
					Build()
				if err != nil {
					return err
				}

				if _, err := a.SaveMember(tx, log)(updatedJuniorWithSenior)(); err != nil {
					return err
				}

				return nil
			})
			if err != nil {
				return []FamilyMember{}, err
			}

			// If transaction succeeded, return the updated members
			updatedSenior, err := GetByCharacterIdProvider(seniorId)(db)()
			if err != nil {
				return []FamilyMember{}, err
			}

			updatedJunior, err := GetByCharacterIdProvider(juniorId)(db)()
			if err != nil {
				return []FamilyMember{}, err
			}

			return []FamilyMember{updatedSenior, updatedJunior}, nil
		}
	}
}

// RemoveMemberCascade removes a member and handles all cascade operations
func (a *AdministratorImpl) RemoveMemberCascade(db *gorm.DB, log logrus.FieldLogger) func(characterId uint32, reason string) model.Provider[[]FamilyMember] {
	return func(characterId uint32, reason string) model.Provider[[]FamilyMember] {
		return func() ([]FamilyMember, error) {
			log.WithFields(logrus.Fields{
				"characterId": characterId,
				"reason":      reason,
			}).Info("Removing member with cascade operations")

			// Get the member to remove
			member, err := GetByCharacterIdProvider(characterId)(db)()
			if err != nil {
				return []FamilyMember{}, err
			}

			var updatedMembers []FamilyMember
			err = db.Transaction(func(tx *gorm.DB) error {
				// If member has a senior, remove from senior's junior list
				if member.HasSenior() {
					if senior, err := GetByCharacterIdProvider(*member.SeniorId())(tx)(); err == nil {
						updatedSenior, err := senior.Builder().
							RemoveJunior(characterId).
							Touch().
							Build()
						if err != nil {
							return err
						}

						if _, err := a.SaveMember(tx, log)(updatedSenior)(); err != nil {
							return err
						}
						updatedMembers = append(updatedMembers, updatedSenior)
					}
				}

				// If member has juniors, remove their senior reference
				if member.HasJuniors() {
					for _, juniorId := range member.JuniorIds() {
						if junior, err := GetByCharacterIdProvider(juniorId)(tx)(); err == nil {
							updatedJunior, err := junior.Builder().
								ClearSeniorId().
								Touch().
								Build()
							if err != nil {
								return err
							}

							if _, err := a.SaveMember(tx, log)(updatedJunior)(); err != nil {
								return err
							}
							updatedMembers = append(updatedMembers, updatedJunior)
						}
					}
				}

				// Remove the member
				if _, err := a.DeleteMember(tx, log)(characterId)(); err != nil {
					return err
				}

				return nil
			})

			return updatedMembers, err
		}
	}
}

// DissolveSubtree removes a senior and all their juniors
func (a *AdministratorImpl) DissolveSubtree(db *gorm.DB, log logrus.FieldLogger) func(seniorId uint32, reason string) model.Provider[[]FamilyMember] {
	return func(seniorId uint32, reason string) model.Provider[[]FamilyMember] {
		return func() ([]FamilyMember, error) {
			log.WithFields(logrus.Fields{
				"seniorId": seniorId,
				"reason":   reason,
			}).Info("Dissolving family subtree")

			// Get the senior
			senior, err := GetByCharacterIdProvider(seniorId)(db)()
			if err != nil {
				return []FamilyMember{}, err
			}

			var allUpdated []FamilyMember

			// If senior has juniors, remove them first
			if senior.HasJuniors() {
				for _, juniorId := range senior.JuniorIds() {
					updatedMembers, err := a.RemoveMemberCascade(db, log)(juniorId, reason)()
					if err != nil {
						return []FamilyMember{}, err
					}
					allUpdated = append(allUpdated, updatedMembers...)
				}
			}

			// Remove the senior
			seniorUpdated, err := a.RemoveMemberCascade(db, log)(seniorId, reason)()
			if err != nil {
				return []FamilyMember{}, err
			}
			allUpdated = append(allUpdated, seniorUpdated...)

			return allUpdated, nil
		}
	}
}

// AwardRepToSenior awards reputation to a junior's senior
func (a *AdministratorImpl) AwardRepToSenior(db *gorm.DB, log logrus.FieldLogger) func(juniorId uint32, amount uint32, source string) model.Provider[FamilyMember] {
	return func(juniorId uint32, amount uint32, source string) model.Provider[FamilyMember] {
		return func() (FamilyMember, error) {
			log.WithFields(logrus.Fields{
				"juniorId": juniorId,
				"amount":   amount,
				"source":   source,
			}).Info("Awarding reputation to senior")

			// Get the junior
			junior, err := GetByCharacterIdProvider(juniorId)(db)()
			if err != nil {
				return FamilyMember{}, err
			}

			// Check if junior has a senior
			if !junior.HasSenior() {
				return FamilyMember{}, errors.New("junior has no senior to award reputation to")
			}

			seniorId := *junior.SeniorId()

			// Get the senior
			senior, err := GetByCharacterIdProvider(seniorId)(db)()
			if err != nil {
				return FamilyMember{}, err
			}

			// Check if junior outlevels senior (half rep)
			finalAmount := amount
			if junior.Level() > senior.Level() {
				finalAmount = amount / 2
				log.WithFields(logrus.Fields{
					"juniorLevel":    junior.Level(),
					"seniorLevel":    senior.Level(),
					"originalAmount": amount,
					"finalAmount":    finalAmount,
				}).Info("Junior outlevels senior, halving reputation award")
			}

			// Award reputation to senior
			updatedSenior, err := senior.Builder().
				AddRep(finalAmount).
				AddDailyRep(finalAmount).
				Touch().
				Build()
			if err != nil {
				return FamilyMember{}, err
			}

			return a.SaveMember(db, log)(updatedSenior)()
		}
	}
}

// ProcessRepActivity processes different types of reputation activities
func (a *AdministratorImpl) ProcessRepActivity(db *gorm.DB, log logrus.FieldLogger) func(juniorId uint32, activityType string, value uint32) model.Provider[FamilyMember] {
	return func(juniorId uint32, activityType string, value uint32) model.Provider[FamilyMember] {
		return func() (FamilyMember, error) {
			log.WithFields(logrus.Fields{
				"juniorId":     juniorId,
				"activityType": activityType,
				"value":        value,
			}).Info("Processing reputation activity")

			var repAmount uint32
			var source string

			switch activityType {
			case "mob_kill":
				// 2 Rep per 5 kills
				repAmount = (value / 5) * 2
				source = "mob_kills"
			case "expedition":
				// Coin reward * 10
				repAmount = value * 10
				source = "expedition"
			default:
				return FamilyMember{}, ErrInvalidActivityType
			}

			if repAmount == 0 {
				// No reputation to award
				return FamilyMember{}, nil
			}

			return a.AwardRepToSenior(db, log)(juniorId, repAmount, source)()
		}
	}
}

// BatchResetDailyRep resets daily reputation for all members
func (a *AdministratorImpl) BatchResetDailyRep(db *gorm.DB, log logrus.FieldLogger) func() model.Provider[BatchResetResult] {
	return func() model.Provider[BatchResetResult] {
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
}

// EnsureMemberExists ensures a family member exists, creating if necessary
func (a *AdministratorImpl) EnsureMemberExists(db *gorm.DB, log logrus.FieldLogger) func(characterId uint32, tenantId uuid.UUID, level uint16, world byte) model.Provider[FamilyMember] {
	return func(characterId uint32, tenantId uuid.UUID, level uint16, world byte) model.Provider[FamilyMember] {
		return func() (FamilyMember, error) {
			log.WithFields(logrus.Fields{
				"characterId": characterId,
				"tenantId":    tenantId,
			}).Info("Ensuring family member exists")

			// Try to get existing member
			member, err := GetByCharacterIdProvider(characterId)(db)()
			if err != nil {
				if errors.Is(err, ErrMemberNotFound) {
					// Create new member
					return a.CreateMember(db, log)(characterId, tenantId, level, world)()
				}
				return FamilyMember{}, err
			}

			// Update member's location/level if changed
			needsUpdate := false
			builder := member.Builder()

			if member.Level() != level {
				builder = builder.SetLevel(level)
				needsUpdate = true
			}
			if member.World() != world {
				builder = builder.SetWorld(world)
				needsUpdate = true
			}

			if needsUpdate {
				updatedMember, err := builder.Touch().Build()
				if err != nil {
					return FamilyMember{}, err
				}
				return a.SaveMember(db, log)(updatedMember)()
			}

			return member, nil
		}
	}
}

// SaveMember saves a family member to the database (create or update)
func (a *AdministratorImpl) SaveMember(db *gorm.DB, log logrus.FieldLogger) func(member FamilyMember) model.Provider[FamilyMember] {
	return func(member FamilyMember) model.Provider[FamilyMember] {
		return func() (FamilyMember, error) {
			log.WithFields(logrus.Fields{
				"characterId": member.CharacterId(),
				"id":          member.Id(),
			}).Debug("Saving family member to database")

			entity := ToEntity(member)

			// Use Save which handles both create and update
			if err := db.Save(&entity).Error; err != nil {
				return FamilyMember{}, err
			}

			return Make(entity)
		}
	}
}

// DeleteMember deletes a family member from the database
func (a *AdministratorImpl) DeleteMember(db *gorm.DB, log logrus.FieldLogger) func(characterId uint32) model.Provider[bool] {
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
