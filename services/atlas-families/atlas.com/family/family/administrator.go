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
	CreateMember(db *gorm.DB, log logrus.FieldLogger) func(characterId uint32, tenantId uuid.UUID, level uint16, world byte, channel byte, mapId uint32) model.Provider[FamilyMember]
	AddJuniorWithValidation(db *gorm.DB, log logrus.FieldLogger) func(seniorId uint32, juniorId uint32, tenantId uuid.UUID, juniorLevel uint16, juniorWorld byte, juniorChannel byte, juniorMapId uint32) model.Provider[[]FamilyMember]
	RemoveMemberCascade(db *gorm.DB, log logrus.FieldLogger) func(characterId uint32, reason string) model.Provider[[]FamilyMember]
	DissolveSubtree(db *gorm.DB, log logrus.FieldLogger) func(seniorId uint32, reason string) model.Provider[[]FamilyMember]
	AwardRepToSenior(db *gorm.DB, log logrus.FieldLogger) func(juniorId uint32, amount uint32, source string) model.Provider[FamilyMember]
	ProcessRepActivity(db *gorm.DB, log logrus.FieldLogger) func(juniorId uint32, activityType string, value uint32) model.Provider[FamilyMember]
	BatchResetDailyRep(db *gorm.DB, log logrus.FieldLogger) func() model.Provider[BatchResetResult]
	EnsureMemberExists(db *gorm.DB, log logrus.FieldLogger) func(characterId uint32, tenantId uuid.UUID, level uint16, world byte, channel byte, mapId uint32) model.Provider[FamilyMember]
}

// AdministratorImpl implements the Administrator interface
type AdministratorImpl struct {
	processor Processor
}

// NewAdministrator creates a new administrator instance
func NewAdministrator(processor Processor) Administrator {
	return &AdministratorImpl{
		processor: processor,
	}
}

// BatchResetResult represents the result of a batch daily rep reset operation
type BatchResetResult struct {
	AffectedCount int64
	ResetTime     time.Time
}

// Administrator-specific errors
var (
	ErrMemberAlreadyExists   = errors.New("family member already exists")
	ErrCannotCreateMember    = errors.New("cannot create family member")
	ErrTransactionFailed     = errors.New("transaction failed")
	ErrInvalidActivityType   = errors.New("invalid activity type")
	ErrJuniorMustExist       = errors.New("junior must exist before creating family link")
	ErrSeniorMustExist       = errors.New("senior must exist before creating family link")
)

// CreateMember creates a new family member with proper validation
func (a *AdministratorImpl) CreateMember(db *gorm.DB, log logrus.FieldLogger) func(characterId uint32, tenantId uuid.UUID, level uint16, world byte, channel byte, mapId uint32) model.Provider[FamilyMember] {
	return func(characterId uint32, tenantId uuid.UUID, level uint16, world byte, channel byte, mapId uint32) model.Provider[FamilyMember] {
		return func() (FamilyMember, error) {
			log.WithFields(logrus.Fields{
				"characterId": characterId,
				"tenantId":    tenantId,
				"level":       level,
				"world":       world,
				"channel":     channel,
				"mapId":       mapId,
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
			member, err := NewBuilder(characterId, tenantId, level, world, channel, mapId).Build()
			if err != nil {
				return FamilyMember{}, err
			}

			// Save to database
			return CreateProvider(member)(db)()
		}
	}
}

// AddJuniorWithValidation adds a junior with full validation and member creation if needed
func (a *AdministratorImpl) AddJuniorWithValidation(db *gorm.DB, log logrus.FieldLogger) func(seniorId uint32, juniorId uint32, tenantId uuid.UUID, juniorLevel uint16, juniorWorld byte, juniorChannel byte, juniorMapId uint32) model.Provider[[]FamilyMember] {
	return func(seniorId uint32, juniorId uint32, tenantId uuid.UUID, juniorLevel uint16, juniorWorld byte, juniorChannel byte, juniorMapId uint32) model.Provider[[]FamilyMember] {
		return func() ([]FamilyMember, error) {
			log.WithFields(logrus.Fields{
				"seniorId":      seniorId,
				"juniorId":      juniorId,
				"tenantId":      tenantId,
				"juniorLevel":   juniorLevel,
				"juniorWorld":   juniorWorld,
				"juniorChannel": juniorChannel,
				"juniorMapId":   juniorMapId,
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
						junior, err = a.CreateMember(tx, log)(juniorId, tenantId, juniorLevel, juniorWorld, juniorChannel, juniorMapId)()
						if err != nil {
							return err
						}
					} else {
						return err
					}
				}

				// Update junior's location to match provided data
				if junior.Level() != juniorLevel || junior.World() != juniorWorld || junior.Channel() != juniorChannel || junior.MapId() != juniorMapId {
					updatedJunior, err := junior.Builder().
						SetLevel(juniorLevel).
						SetWorld(juniorWorld).
						SetChannel(juniorChannel).
						SetMapId(juniorMapId).
						Touch().
						Build()
					if err != nil {
						return err
					}
					
					if _, err := UpdateProvider(updatedJunior)(tx)(); err != nil {
						return err
					}
				}

				// Add junior using processor
				_, err = a.processor.AddJunior(tx, log)(seniorId, juniorId)()
				if err != nil {
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

			// Use processor to handle removal logic
			return a.processor.RemoveMember(db, log)(characterId, reason)()
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
					updatedMembers, err := a.processor.RemoveMember(db, log)(juniorId, reason)()
					if err != nil {
						return []FamilyMember{}, err
					}
					allUpdated = append(allUpdated, updatedMembers...)
				}
			}

			// Remove the senior
			seniorUpdated, err := a.processor.RemoveMember(db, log)(seniorId, reason)()
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
					"juniorLevel": junior.Level(),
					"seniorLevel": senior.Level(),
					"originalAmount": amount,
					"finalAmount": finalAmount,
				}).Info("Junior outlevels senior, halving reputation award")
			}

			// Award reputation using processor
			return a.processor.AwardRep(db, log)(seniorId, finalAmount, source)()
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

			// Use processor to reset daily rep
			affectedCount, err := a.processor.ResetDailyRep(db, log)()()
			if err != nil {
				return BatchResetResult{}, err
			}

			return BatchResetResult{
				AffectedCount: affectedCount,
				ResetTime:     resetTime,
			}, nil
		}
	}
}

// EnsureMemberExists ensures a family member exists, creating if necessary
func (a *AdministratorImpl) EnsureMemberExists(db *gorm.DB, log logrus.FieldLogger) func(characterId uint32, tenantId uuid.UUID, level uint16, world byte, channel byte, mapId uint32) model.Provider[FamilyMember] {
	return func(characterId uint32, tenantId uuid.UUID, level uint16, world byte, channel byte, mapId uint32) model.Provider[FamilyMember] {
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
					return a.CreateMember(db, log)(characterId, tenantId, level, world, channel, mapId)()
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
			if member.Channel() != channel {
				builder = builder.SetChannel(channel)
				needsUpdate = true
			}
			if member.MapId() != mapId {
				builder = builder.SetMapId(mapId)
				needsUpdate = true
			}

			if needsUpdate {
				updatedMember, err := builder.Touch().Build()
				if err != nil {
					return FamilyMember{}, err
				}
				return UpdateProvider(updatedMember)(db)()
			}

			return member, nil
		}
	}
}