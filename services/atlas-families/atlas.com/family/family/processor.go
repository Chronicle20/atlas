package family

import (
	"errors"
	"time"

	"github.com/Chronicle20/atlas-model/model"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

// Processor interface defines the core business logic operations
type Processor interface {
	AddJunior(db *gorm.DB, log logrus.FieldLogger) func(seniorId uint32, juniorId uint32) model.Provider[FamilyMember]
	RemoveMember(db *gorm.DB, log logrus.FieldLogger) func(characterId uint32, reason string) model.Provider[[]FamilyMember]
	BreakLink(db *gorm.DB, log logrus.FieldLogger) func(characterId uint32, reason string) model.Provider[[]FamilyMember]
	AwardRep(db *gorm.DB, log logrus.FieldLogger) func(characterId uint32, amount uint32, source string) model.Provider[FamilyMember]
	DeductRep(db *gorm.DB, log logrus.FieldLogger) func(characterId uint32, amount uint32, reason string) model.Provider[FamilyMember]
	ResetDailyRep(db *gorm.DB, log logrus.FieldLogger) func() model.Provider[int64]
	UpdateWorld(db *gorm.DB, log logrus.FieldLogger) func(characterId uint32, world byte) model.Provider[FamilyMember]
	UpdateLevel(db *gorm.DB, log logrus.FieldLogger) func(characterId uint32, level uint16) model.Provider[FamilyMember]
	
	// AndEmit variants for Kafka message emission
	AddJuniorAndEmit(db *gorm.DB, log logrus.FieldLogger) func(transactionId string, seniorId uint32, juniorId uint32) model.Provider[FamilyMember]
	RemoveMemberAndEmit(db *gorm.DB, log logrus.FieldLogger) func(transactionId string, characterId uint32, reason string) model.Provider[[]FamilyMember]
	BreakLinkAndEmit(db *gorm.DB, log logrus.FieldLogger) func(transactionId string, characterId uint32, reason string) model.Provider[[]FamilyMember]
	AwardRepAndEmit(db *gorm.DB, log logrus.FieldLogger) func(transactionId string, characterId uint32, amount uint32, source string) model.Provider[FamilyMember]
	DeductRepAndEmit(db *gorm.DB, log logrus.FieldLogger) func(transactionId string, characterId uint32, amount uint32, reason string) model.Provider[FamilyMember]
}

// ProcessorImpl implements the Processor interface  
type ProcessorImpl struct{
	administrator Administrator
	producer     EventProducer
}

// EventProducer interface for emitting family events
type EventProducer interface {
	EmitLinkCreated(transactionId string, characterId uint32, seniorId uint32, juniorId uint32) error
	EmitLinkBroken(transactionId string, characterId uint32, seniorId uint32, juniorId uint32, reason string) error
	EmitRepGained(transactionId string, characterId uint32, repGained uint32, dailyRep uint32, source string) error
	EmitRepRedeemed(transactionId string, characterId uint32, repRedeemed uint32, reason string) error
	EmitRepError(transactionId string, characterId uint32, errorCode string, errorMessage string, amount uint32) error
	EmitLinkError(transactionId string, characterId uint32, seniorId uint32, juniorId uint32, errorCode string, errorMessage string) error
}

// NewProcessor creates a new processor instance
func NewProcessor(administrator Administrator, producer EventProducer) Processor {
	return &ProcessorImpl{
		administrator: administrator,
		producer:     producer,
	}
}

// Business logic errors
var (
	ErrMemberNotFound         = errors.New("family member not found")
	ErrSeniorNotFound         = errors.New("senior member not found")
	ErrJuniorNotFound         = errors.New("junior member not found")
	ErrSeniorHasTooManyJuniors = errors.New("senior already has maximum number of juniors")
	ErrJuniorAlreadyLinked    = errors.New("junior is already linked to a senior")
	ErrLevelDifferenceTooLarge = errors.New("level difference exceeds maximum allowed")
	ErrNotOnSameMap           = errors.New("members must be on the same map to link")
	ErrInsufficientRep        = errors.New("insufficient reputation for operation")
	ErrRepCapExceeded         = errors.New("daily reputation cap exceeded")
	ErrCannotRemoveSelf       = errors.New("cannot remove self from family")
	ErrNoLinkToBreak          = errors.New("no family link exists to break")
)

// AddJunior adds a junior to a senior's family
func (p *ProcessorImpl) AddJunior(db *gorm.DB, log logrus.FieldLogger) func(seniorId uint32, juniorId uint32) model.Provider[FamilyMember] {
	return func(seniorId uint32, juniorId uint32) model.Provider[FamilyMember] {
		return func() (FamilyMember, error) {
			log.WithFields(logrus.Fields{
				"seniorId": seniorId,
				"juniorId": juniorId,
			}).Info("Adding junior to family")

			// Validate input
			if seniorId == juniorId {
				return FamilyMember{}, ErrSelfReference
			}

			// Get senior member
			seniorModel, err := GetByCharacterIdProvider(seniorId)(db)()
			if err != nil {
				if errors.Is(err, ErrMemberNotFound) {
					return FamilyMember{}, ErrSeniorNotFound
				}
				return FamilyMember{}, err
			}

			// Check if senior can add more juniors
			if !seniorModel.CanAddJunior() {
				return FamilyMember{}, ErrSeniorHasTooManyJuniors
			}

			// Get junior member
			juniorModel, err := GetByCharacterIdProvider(juniorId)(db)()
			if err != nil {
				if errors.Is(err, ErrMemberNotFound) {
					return FamilyMember{}, ErrJuniorNotFound
				}
				return FamilyMember{}, err
			}

			// Check if junior already has a senior
			if juniorModel.HasSenior() {
				return FamilyMember{}, ErrJuniorAlreadyLinked
			}

			// Validate level difference
			if !seniorModel.ValidateLevelDifference(juniorModel.Level()) {
				return FamilyMember{}, ErrLevelDifferenceTooLarge
			}

			// Validate same world
			if !seniorModel.IsSameWorld(juniorModel.World()) {
				return FamilyMember{}, ErrNotOnSameMap
			}

			// Begin transaction
			var result FamilyMember
			err = db.Transaction(func(tx *gorm.DB) error {
				// Update senior - add junior
				updatedSenior, err := seniorModel.Builder().
					AddJunior(juniorId).
					Touch().
					Build()
				if err != nil {
					return err
				}

				// Save updated senior via administrator
				if _, err := p.administrator.SaveMember(tx, log)(updatedSenior)(); err != nil {
					return err
				}
				result = updatedSenior

				// Update junior - set senior
				updatedJunior, err := juniorModel.Builder().
					SetSeniorId(seniorId).
					Touch().
					Build()
				if err != nil {
					return err
				}

				// Save updated junior via administrator
				if _, err := p.administrator.SaveMember(tx, log)(updatedJunior)(); err != nil {
					return err
				}

				result = updatedSenior
				return nil
			})
			
			return result, err
		}
	}
}

// RemoveMember removes a member from the family and handles cascade operations
func (p *ProcessorImpl) RemoveMember(db *gorm.DB, log logrus.FieldLogger) func(characterId uint32, reason string) model.Provider[[]FamilyMember] {
	return func(characterId uint32, reason string) model.Provider[[]FamilyMember] {
		return func() ([]FamilyMember, error) {
			log.WithFields(logrus.Fields{
				"characterId": characterId,
				"reason":      reason,
			}).Info("Removing member from family")

			// Get the member to remove
			memberModel, err := GetByCharacterIdProvider(characterId)(db)()
			if err != nil {
				return []FamilyMember{}, err
			}

			var updatedMembers []FamilyMember
			err = db.Transaction(func(tx *gorm.DB) error {
				// If member has a senior, remove from senior's junior list
				if memberModel.HasSenior() {
					if seniorModel, err := GetByCharacterIdProvider(*memberModel.SeniorId())(tx)(); err == nil {
						updatedSenior, err := seniorModel.Builder().
							RemoveJunior(characterId).
							Touch().
							Build()
						if err != nil {
							return err
						}

						if _, err := p.administrator.SaveMember(tx, log)(updatedSenior)(); err != nil {
							return err
						}
						updatedMembers = append(updatedMembers, updatedSenior)
					}
				}

				// If member has juniors, remove their senior reference
				if memberModel.HasJuniors() {
					for _, juniorId := range memberModel.JuniorIds() {
						if juniorModel, err := GetByCharacterIdProvider(juniorId)(tx)(); err == nil {
							updatedJunior, err := juniorModel.Builder().
								ClearSeniorId().
								Touch().
								Build()
							if err != nil {
								return err
							}

							if _, err := p.administrator.SaveMember(tx, log)(updatedJunior)(); err != nil {
								return err
							}
							updatedMembers = append(updatedMembers, updatedJunior)
						}
					}
				}

				// Remove the member
				if _, err := p.administrator.DeleteMember(tx, log)(characterId)(); err != nil {
					return err
				}

				return nil
			})
			
			return updatedMembers, err
		}
	}
}

// BreakLink breaks the family link for a character
func (p *ProcessorImpl) BreakLink(db *gorm.DB, log logrus.FieldLogger) func(characterId uint32, reason string) model.Provider[[]FamilyMember] {
	return func(characterId uint32, reason string) model.Provider[[]FamilyMember] {
		return func() ([]FamilyMember, error) {
			log.WithFields(logrus.Fields{
				"characterId": characterId,
				"reason":      reason,
			}).Info("Breaking family link")

			// Get the member
			memberModel, err := GetByCharacterIdProvider(characterId)(db)()
			if err != nil {
				return []FamilyMember{}, err
			}

			// Check if there's a link to break
			if !memberModel.HasSenior() && !memberModel.HasJuniors() {
				return []FamilyMember{}, ErrNoLinkToBreak
			}

			var updatedMembers []FamilyMember
			err = db.Transaction(func(tx *gorm.DB) error {
				// If member has a senior, remove from senior's junior list
				if memberModel.HasSenior() {
					var seniorEntity Entity
					if err := tx.Where("character_id = ?", *memberModel.SeniorId()).First(&seniorEntity).Error; err == nil {
						seniorModel, err := Make(seniorEntity)
						if err == nil {
							updatedSenior, err := seniorModel.Builder().
								RemoveJunior(characterId).
								Touch().
								Build()
							if err != nil {
								return err
							}

							seniorEntity = ToEntity(updatedSenior)
							if err := tx.Save(&seniorEntity).Error; err != nil {
								return err
							}
							updatedMembers = append(updatedMembers, updatedSenior)
						}
					}

					// Clear member's senior reference
					updatedMember, err := memberModel.Builder().
						ClearSeniorId().
						Touch().
						Build()
					if err != nil {
						return err
					}

					if _, err := p.administrator.SaveMember(tx, log)(updatedMember)(); err != nil {
						return err
					}
					updatedMembers = append(updatedMembers, updatedMember)
				}

				// If member has juniors, clear their senior reference
				if memberModel.HasJuniors() {
					for _, juniorId := range memberModel.JuniorIds() {
						if juniorModel, err := GetByCharacterIdProvider(juniorId)(tx)(); err == nil {
							updatedJunior, err := juniorModel.Builder().
								ClearSeniorId().
								Touch().
								Build()
							if err != nil {
								return err
							}

							if _, err := p.administrator.SaveMember(tx, log)(updatedJunior)(); err != nil {
								return err
							}
							updatedMembers = append(updatedMembers, updatedJunior)
						}
					}

					// Clear member's junior list
					updatedMember, err := memberModel.Builder().
						SetJuniorIds([]uint32{}).
						Touch().
						Build()
					if err != nil {
						return err
					}

					if _, err := p.administrator.SaveMember(tx, log)(updatedMember)(); err != nil {
						return err
					}

					// Update the member in the result
					for i, member := range updatedMembers {
						if member.CharacterId() == characterId {
							updatedMembers[i] = updatedMember
							break
						}
					}
					if len(updatedMembers) == 0 || updatedMembers[len(updatedMembers)-1].CharacterId() != characterId {
						updatedMembers = append(updatedMembers, updatedMember)
					}
				}

				return nil
			})
			
			return updatedMembers, err
		}
	}
}

// AwardRep awards reputation to a character
func (p *ProcessorImpl) AwardRep(db *gorm.DB, log logrus.FieldLogger) func(characterId uint32, amount uint32, source string) model.Provider[FamilyMember] {
	return func(characterId uint32, amount uint32, source string) model.Provider[FamilyMember] {
		return func() (FamilyMember, error) {
			log.WithFields(logrus.Fields{
				"characterId": characterId,
				"amount":      amount,
				"source":      source,
			}).Info("Awarding reputation")

			// Get the member
			memberModel, err := GetByCharacterIdProvider(characterId)(db)()
			if err != nil {
				return FamilyMember{}, err
			}

			// Check if member can receive more rep today
			if !memberModel.CanReceiveRep(amount) {
				return FamilyMember{}, ErrRepCapExceeded
			}

			// Update member with new rep
			updatedMember, err := memberModel.Builder().
				AddRep(amount).
				AddDailyRep(amount).
				Touch().
				Build()
			if err != nil {
				return FamilyMember{}, err
			}

			if _, err := p.administrator.SaveMember(db, log)(updatedMember)(); err != nil {
				return FamilyMember{}, err
			}

			return updatedMember, nil
		}
	}
}

// DeductRep deducts reputation from a character
func (p *ProcessorImpl) DeductRep(db *gorm.DB, log logrus.FieldLogger) func(characterId uint32, amount uint32, reason string) model.Provider[FamilyMember] {
	return func(characterId uint32, amount uint32, reason string) model.Provider[FamilyMember] {
		return func() (FamilyMember, error) {
			log.WithFields(logrus.Fields{
				"characterId": characterId,
				"amount":      amount,
				"reason":      reason,
			}).Info("Deducting reputation")

			// Get the member
			memberModel, err := GetByCharacterIdProvider(characterId)(db)()
			if err != nil {
				return FamilyMember{}, err
			}

			// Check if member has enough rep
			if memberModel.Rep() < amount {
				return FamilyMember{}, ErrInsufficientRep
			}

			// Update member with deducted rep
			updatedMember, err := memberModel.Builder().
				SubtractRep(amount).
				Touch().
				Build()
			if err != nil {
				return FamilyMember{}, err
			}

			if _, err := p.administrator.SaveMember(db, log)(updatedMember)(); err != nil {
				return FamilyMember{}, err
			}

			return updatedMember, nil
		}
	}
}

// ResetDailyRep resets daily reputation for all members
func (p *ProcessorImpl) ResetDailyRep(db *gorm.DB, log logrus.FieldLogger) func() model.Provider[int64] {
	return func() model.Provider[int64] {
		return func() (int64, error) {
			log.Info("Resetting daily reputation for all members")

			result := db.Model(&Entity{}).
				Where("daily_rep > 0").
				Updates(map[string]interface{}{
					"daily_rep":  0,
					"updated_at": time.Now(),
				})

			if result.Error != nil {
				return 0, result.Error
			}

			return result.RowsAffected, nil
		}
	}
}

// UpdateWorld updates a member's world information
func (p *ProcessorImpl) UpdateWorld(db *gorm.DB, log logrus.FieldLogger) func(characterId uint32, world byte) model.Provider[FamilyMember] {
	return func(characterId uint32, world byte) model.Provider[FamilyMember] {
		return func() (FamilyMember, error) {
			log.WithFields(logrus.Fields{
				"characterId": characterId,
				"world":       world,
			}).Info("Updating member world")

			// Get the member
			memberModel, err := GetByCharacterIdProvider(characterId)(db)()
			if err != nil {
				return FamilyMember{}, err
			}

			// Update member world
			updatedMember, err := memberModel.Builder().
				SetWorld(world).
				Touch().
				Build()
			if err != nil {
				return FamilyMember{}, err
			}

			if _, err := p.administrator.SaveMember(db, log)(updatedMember)(); err != nil {
				return FamilyMember{}, err
			}

			return updatedMember, nil
		}
	}
}

// UpdateLevel updates a member's level
func (p *ProcessorImpl) UpdateLevel(db *gorm.DB, log logrus.FieldLogger) func(characterId uint32, level uint16) model.Provider[FamilyMember] {
	return func(characterId uint32, level uint16) model.Provider[FamilyMember] {
		return func() (FamilyMember, error) {
			log.WithFields(logrus.Fields{
				"characterId": characterId,
				"level":       level,
			}).Info("Updating member level")

			// Validate level
			if err := ValidateLevel(level); err != nil {
				return FamilyMember{}, err
			}

			// Get the member
			memberModel, err := GetByCharacterIdProvider(characterId)(db)()
			if err != nil {
				return FamilyMember{}, err
			}

			// Update member level
			updatedMember, err := memberModel.Builder().
				SetLevel(level).
				Touch().
				Build()
			if err != nil {
				return FamilyMember{}, err
			}

			if _, err := p.administrator.SaveMember(db, log)(updatedMember)(); err != nil {
				return FamilyMember{}, err
			}

			return updatedMember, nil
		}
	}
}

// AndEmit variants - combine business logic with event emission

// AddJuniorAndEmit adds a junior and emits appropriate events
func (p *ProcessorImpl) AddJuniorAndEmit(db *gorm.DB, log logrus.FieldLogger) func(transactionId string, seniorId uint32, juniorId uint32) model.Provider[FamilyMember] {
	return func(transactionId string, seniorId uint32, juniorId uint32) model.Provider[FamilyMember] {
		return func() (FamilyMember, error) {
			// Execute business logic
			result, err := p.AddJunior(db, log)(seniorId, juniorId)()
			if err != nil {
				// Emit error event
				if emitErr := p.producer.EmitLinkError(transactionId, seniorId, seniorId, juniorId, "ADD_JUNIOR_FAILED", err.Error()); emitErr != nil {
					log.WithError(emitErr).Error("Failed to emit link error event")
				}
				return FamilyMember{}, err
			}
			
			// Emit success event
			if emitErr := p.producer.EmitLinkCreated(transactionId, seniorId, seniorId, juniorId); emitErr != nil {
				log.WithError(emitErr).Error("Failed to emit link created event")
			}
			
			return result, nil
		}
	}
}

// RemoveMemberAndEmit removes a member and emits appropriate events
func (p *ProcessorImpl) RemoveMemberAndEmit(db *gorm.DB, log logrus.FieldLogger) func(transactionId string, characterId uint32, reason string) model.Provider[[]FamilyMember] {
	return func(transactionId string, characterId uint32, reason string) model.Provider[[]FamilyMember] {
		return func() ([]FamilyMember, error) {
			// Get member info before removal for event emission
			memberModel, err := GetByCharacterIdProvider(characterId)(db)()
			if err != nil {
				return []FamilyMember{}, err
			}
			
			// Execute business logic
			result, err := p.RemoveMember(db, log)(characterId, reason)()
			if err != nil {
				return []FamilyMember{}, err
			}
			
			// Emit link broken events for all affected relationships
			if memberModel.HasSenior() {
				if emitErr := p.producer.EmitLinkBroken(transactionId, characterId, *memberModel.SeniorId(), characterId, reason); emitErr != nil {
					log.WithError(emitErr).Error("Failed to emit link broken event for senior")
				}
			}
			
			for _, juniorId := range memberModel.JuniorIds() {
				if emitErr := p.producer.EmitLinkBroken(transactionId, characterId, characterId, juniorId, reason); emitErr != nil {
					log.WithError(emitErr).Error("Failed to emit link broken event for junior")
				}
			}
			
			return result, nil
		}
	}
}

// BreakLinkAndEmit breaks a link and emits appropriate events
func (p *ProcessorImpl) BreakLinkAndEmit(db *gorm.DB, log logrus.FieldLogger) func(transactionId string, characterId uint32, reason string) model.Provider[[]FamilyMember] {
	return func(transactionId string, characterId uint32, reason string) model.Provider[[]FamilyMember] {
		return func() ([]FamilyMember, error) {
			// Get member info before breaking links for event emission
			memberModel, err := GetByCharacterIdProvider(characterId)(db)()
			if err != nil {
				return []FamilyMember{}, err
			}
			
			// Execute business logic
			result, err := p.BreakLink(db, log)(characterId, reason)()
			if err != nil {
				return []FamilyMember{}, err
			}
			
			// Emit link broken events for all affected relationships
			if memberModel.HasSenior() {
				if emitErr := p.producer.EmitLinkBroken(transactionId, characterId, *memberModel.SeniorId(), characterId, reason); emitErr != nil {
					log.WithError(emitErr).Error("Failed to emit link broken event for senior")
				}
			}
			
			for _, juniorId := range memberModel.JuniorIds() {
				if emitErr := p.producer.EmitLinkBroken(transactionId, characterId, characterId, juniorId, reason); emitErr != nil {
					log.WithError(emitErr).Error("Failed to emit link broken event for junior")
				}
			}
			
			return result, nil
		}
	}
}

// AwardRepAndEmit awards reputation and emits appropriate events
func (p *ProcessorImpl) AwardRepAndEmit(db *gorm.DB, log logrus.FieldLogger) func(transactionId string, characterId uint32, amount uint32, source string) model.Provider[FamilyMember] {
	return func(transactionId string, characterId uint32, amount uint32, source string) model.Provider[FamilyMember] {
		return func() (FamilyMember, error) {
			// Execute business logic
			result, err := p.AwardRep(db, log)(characterId, amount, source)()
			if err != nil {
				// Emit error event
				if emitErr := p.producer.EmitRepError(transactionId, characterId, "AWARD_REP_FAILED", err.Error(), amount); emitErr != nil {
					log.WithError(emitErr).Error("Failed to emit rep error event")
				}
				return FamilyMember{}, err
			}
			
			// Emit success event
			if emitErr := p.producer.EmitRepGained(transactionId, characterId, amount, result.DailyRep(), source); emitErr != nil {
				log.WithError(emitErr).Error("Failed to emit rep gained event")
			}
			
			return result, nil
		}
	}
}

// DeductRepAndEmit deducts reputation and emits appropriate events
func (p *ProcessorImpl) DeductRepAndEmit(db *gorm.DB, log logrus.FieldLogger) func(transactionId string, characterId uint32, amount uint32, reason string) model.Provider[FamilyMember] {
	return func(transactionId string, characterId uint32, amount uint32, reason string) model.Provider[FamilyMember] {
		return func() (FamilyMember, error) {
			// Execute business logic
			result, err := p.DeductRep(db, log)(characterId, amount, reason)()
			if err != nil {
				// Emit error event
				if emitErr := p.producer.EmitRepError(transactionId, characterId, "DEDUCT_REP_FAILED", err.Error(), amount); emitErr != nil {
					log.WithError(emitErr).Error("Failed to emit rep error event")
				}
				return FamilyMember{}, err
			}
			
			// Emit success event
			if emitErr := p.producer.EmitRepRedeemed(transactionId, characterId, amount, reason); emitErr != nil {
				log.WithError(emitErr).Error("Failed to emit rep redeemed event")
			}
			
			return result, nil
		}
	}
}