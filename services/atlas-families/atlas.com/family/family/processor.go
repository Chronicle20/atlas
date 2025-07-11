package family

import (
	"errors"
	"time"

	"atlas-family/kafka/message"
	"atlas-family/kafka/producer"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/segmentio/kafka-go"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

// Processor interface defines the core business logic operations
type Processor interface {
	AddJunior(buf *message.Buffer) func(seniorId uint32, juniorId uint32) model.Provider[FamilyMember]
	RemoveMember(buf *message.Buffer) func(characterId uint32, reason string) model.Provider[[]FamilyMember]
	BreakLink(buf *message.Buffer) func(characterId uint32, reason string) model.Provider[[]FamilyMember]
	AwardRep(buf *message.Buffer) func(characterId uint32, amount uint32, source string) model.Provider[FamilyMember]
	DeductRep(buf *message.Buffer) func(characterId uint32, amount uint32, reason string) model.Provider[FamilyMember]
	ResetDailyRep(buf *message.Buffer) func() model.Provider[int64]
	UpdateWorld(buf *message.Buffer) func(characterId uint32, world byte) model.Provider[FamilyMember]
	UpdateLevel(buf *message.Buffer) func(characterId uint32, level uint16) model.Provider[FamilyMember]
	
	// AndEmit variants for Kafka message emission
	AddJuniorAndEmit(transactionId string, seniorId uint32, juniorId uint32) model.Provider[FamilyMember]
	RemoveMemberAndEmit(transactionId string, characterId uint32, reason string) model.Provider[[]FamilyMember]
	BreakLinkAndEmit(transactionId string, characterId uint32, reason string) model.Provider[[]FamilyMember]
	AwardRepAndEmit(transactionId string, characterId uint32, amount uint32, source string) model.Provider[FamilyMember]
	DeductRepAndEmit(transactionId string, characterId uint32, amount uint32, reason string) model.Provider[FamilyMember]
}

// ProcessorImpl implements the Processor interface  
type ProcessorImpl struct{
	db           *gorm.DB
	log          logrus.FieldLogger
	administrator Administrator
	producer     producer.Provider
	eventProducer EventProducer
}

// EventProducer interface for creating event message providers
type EventProducer interface {
	LinkCreatedEventProvider(transactionId string, characterId uint32, seniorId uint32, juniorId uint32) model.Provider[[]kafka.Message]
	LinkBrokenEventProvider(transactionId string, characterId uint32, seniorId uint32, juniorId uint32, reason string) model.Provider[[]kafka.Message]
	RepGainedEventProvider(transactionId string, characterId uint32, repGained uint32, dailyRep uint32, source string) model.Provider[[]kafka.Message]
	RepRedeemedEventProvider(transactionId string, characterId uint32, repRedeemed uint32, reason string) model.Provider[[]kafka.Message]
	RepErrorEventProvider(transactionId string, characterId uint32, errorCode string, errorMessage string, amount uint32) model.Provider[[]kafka.Message]
	LinkErrorEventProvider(transactionId string, characterId uint32, seniorId uint32, juniorId uint32, errorCode string, errorMessage string) model.Provider[[]kafka.Message]
}

// NewProcessor creates a new processor instance
func NewProcessor(db *gorm.DB, log logrus.FieldLogger, administrator Administrator, producer producer.Provider, eventProducer EventProducer) Processor {
	return &ProcessorImpl{
		db:           db,
		log:          log,
		administrator: administrator,
		producer:     producer,
		eventProducer: eventProducer,
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
func (p *ProcessorImpl) AddJunior(buf *message.Buffer) func(seniorId uint32, juniorId uint32) model.Provider[FamilyMember] {
	return func(seniorId uint32, juniorId uint32) model.Provider[FamilyMember] {
		return func() (FamilyMember, error) {
			p.log.WithFields(logrus.Fields{
				"seniorId": seniorId,
				"juniorId": juniorId,
			}).Info("Adding junior to family")

			// Validate input
			if seniorId == juniorId {
				return FamilyMember{}, ErrSelfReference
			}

			// Get senior member
			seniorModel, err := GetByCharacterIdProvider(seniorId)(p.db)()
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
			juniorModel, err := GetByCharacterIdProvider(juniorId)(p.db)()
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
			err = p.db.Transaction(func(tx *gorm.DB) error {
				// Update senior - add junior
				updatedSenior, err := seniorModel.Builder().
					AddJunior(juniorId).
					Touch().
					Build()
				if err != nil {
					return err
				}

				// Save updated senior via administrator
				if _, err := p.administrator.SaveMember(tx, p.log)(updatedSenior)(); err != nil {
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
				if _, err := p.administrator.SaveMember(tx, p.log)(updatedJunior)(); err != nil {
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
func (p *ProcessorImpl) RemoveMember(buf *message.Buffer) func(characterId uint32, reason string) model.Provider[[]FamilyMember] {
	return func(characterId uint32, reason string) model.Provider[[]FamilyMember] {
		return func() ([]FamilyMember, error) {
			p.log.WithFields(logrus.Fields{
				"characterId": characterId,
				"reason":      reason,
			}).Info("Removing member from family")

			// Get the member to remove
			memberModel, err := GetByCharacterIdProvider(characterId)(p.db)()
			if err != nil {
				return []FamilyMember{}, err
			}

			var updatedMembers []FamilyMember
			err = p.db.Transaction(func(tx *gorm.DB) error {
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

						if _, err := p.administrator.SaveMember(tx, p.log)(updatedSenior)(); err != nil {
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

							if _, err := p.administrator.SaveMember(tx, p.log)(updatedJunior)(); err != nil {
								return err
							}
							updatedMembers = append(updatedMembers, updatedJunior)
						}
					}
				}

				// Remove the member
				if _, err := p.administrator.DeleteMember(tx, p.log)(characterId)(); err != nil {
					return err
				}

				return nil
			})
			
			return updatedMembers, err
		}
	}
}

// BreakLink breaks the family link for a character
func (p *ProcessorImpl) BreakLink(buf *message.Buffer) func(characterId uint32, reason string) model.Provider[[]FamilyMember] {
	return func(characterId uint32, reason string) model.Provider[[]FamilyMember] {
		return func() ([]FamilyMember, error) {
		p.log.WithFields(logrus.Fields{
			"characterId": characterId,
			"reason":      reason,
		}).Info("Breaking family link")

		// Get the member
		memberModel, err := GetByCharacterIdProvider(characterId)(p.db)()
		if err != nil {
			return []FamilyMember{}, err
		}

		// Check if there's a link to break
		if !memberModel.HasSenior() && !memberModel.HasJuniors() {
			return []FamilyMember{}, ErrNoLinkToBreak
		}

		var updatedMembers []FamilyMember
		err = p.db.Transaction(func(tx *gorm.DB) error {
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

					if _, err := p.administrator.SaveMember(tx, p.log)(updatedSenior)(); err != nil {
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

				if _, err := p.administrator.SaveMember(tx, p.log)(updatedMember)(); err != nil {
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

						if _, err := p.administrator.SaveMember(tx, p.log)(updatedJunior)(); err != nil {
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

				if _, err := p.administrator.SaveMember(tx, p.log)(updatedMember)(); err != nil {
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
func (p *ProcessorImpl) AwardRep(buf *message.Buffer) func(characterId uint32, amount uint32, source string) model.Provider[FamilyMember] {
	return func(characterId uint32, amount uint32, source string) model.Provider[FamilyMember] {
		return func() (FamilyMember, error) {
		p.log.WithFields(logrus.Fields{
			"characterId": characterId,
			"amount":      amount,
			"source":      source,
		}).Info("Awarding reputation")

		// Get the member
		memberModel, err := GetByCharacterIdProvider(characterId)(p.db)()
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

		if _, err := p.administrator.SaveMember(p.db, p.log)(updatedMember)(); err != nil {
			return FamilyMember{}, err
		}

		return updatedMember, nil
		}
	}
}

// DeductRep deducts reputation from a character
func (p *ProcessorImpl) DeductRep(buf *message.Buffer) func(characterId uint32, amount uint32, reason string) model.Provider[FamilyMember] {
	return func(characterId uint32, amount uint32, reason string) model.Provider[FamilyMember] {
		return func() (FamilyMember, error) {
		p.log.WithFields(logrus.Fields{
			"characterId": characterId,
			"amount":      amount,
			"reason":      reason,
		}).Info("Deducting reputation")

		// Get the member
		memberModel, err := GetByCharacterIdProvider(characterId)(p.db)()
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

		if _, err := p.administrator.SaveMember(p.db, p.log)(updatedMember)(); err != nil {
			return FamilyMember{}, err
		}

		return updatedMember, nil
		}
	}
}

// ResetDailyRep resets daily reputation for all members
func (p *ProcessorImpl) ResetDailyRep(buf *message.Buffer) func() model.Provider[int64] {
	return func() model.Provider[int64] {
		return func() (int64, error) {
		p.log.Info("Resetting daily reputation for all members")

		result, err := p.administrator.BatchResetDailyRep(p.db, p.log)()
		if err != nil {
			return 0, err
		}

		return result.AffectedCount, nil
		}
	}
}

// UpdateWorld updates a member's world information
func (p *ProcessorImpl) UpdateWorld(buf *message.Buffer) func(characterId uint32, world byte) model.Provider[FamilyMember] {
	return func(characterId uint32, world byte) model.Provider[FamilyMember] {
		return func() (FamilyMember, error) {
		p.log.WithFields(logrus.Fields{
			"characterId": characterId,
			"world":       world,
		}).Info("Updating member world")

		// Get the member
		memberModel, err := GetByCharacterIdProvider(characterId)(p.db)()
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

		if _, err := p.administrator.SaveMember(p.db, p.log)(updatedMember)(); err != nil {
			return FamilyMember{}, err
		}

		return updatedMember, nil
		}
	}
}

// UpdateLevel updates a member's level
func (p *ProcessorImpl) UpdateLevel(buf *message.Buffer) func(characterId uint32, level uint16) model.Provider[FamilyMember] {
	return func(characterId uint32, level uint16) model.Provider[FamilyMember] {
		return func() (FamilyMember, error) {
		p.log.WithFields(logrus.Fields{
			"characterId": characterId,
			"level":       level,
		}).Info("Updating member level")

		// Validate level
		if err := ValidateLevel(level); err != nil {
			return FamilyMember{}, err
		}

		// Get the member
		memberModel, err := GetByCharacterIdProvider(characterId)(p.db)()
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

		if _, err := p.administrator.SaveMember(p.db, p.log)(updatedMember)(); err != nil {
			return FamilyMember{}, err
		}

		return updatedMember, nil
		}
	}
}

// AndEmit variants - combine business logic with event emission

// AddJuniorAndEmit adds a junior and emits appropriate events
func (p *ProcessorImpl) AddJuniorAndEmit(transactionId string, seniorId uint32, juniorId uint32) model.Provider[FamilyMember] {
	emitFunc := message.EmitWithResult[FamilyMember, struct{}](p.producer)(func(buf *message.Buffer) func(struct{}) (FamilyMember, error) {
		return func(struct{}) (FamilyMember, error) {
			// Execute business logic with message buffer
			result, err := p.AddJunior(buf)(seniorId, juniorId)()
			if err != nil {
				// Add error event to buffer
				if putErr := buf.Put("FAMILY_ERRORS", p.eventProducer.LinkErrorEventProvider(transactionId, seniorId, seniorId, juniorId, "ADD_JUNIOR_FAILED", err.Error())); putErr != nil {
					p.log.WithError(putErr).Error("Failed to add link error event to buffer")
				}
				return FamilyMember{}, err
			}
			
			// Add success event to buffer
			if putErr := buf.Put("FAMILY_STATUS", p.eventProducer.LinkCreatedEventProvider(transactionId, seniorId, seniorId, juniorId)); putErr != nil {
				p.log.WithError(putErr).Error("Failed to add link created event to buffer")
			}
			
			return result, nil
		}
	})
	return func() (FamilyMember, error) {
		return emitFunc(struct{}{})
	}
}

// RemoveMemberAndEmit removes a member and emits appropriate events
func (p *ProcessorImpl) RemoveMemberAndEmit(transactionId string, characterId uint32, reason string) model.Provider[[]FamilyMember] {
	emitFunc := message.EmitWithResult[[]FamilyMember, struct{}](p.producer)(func(buf *message.Buffer) func(struct{}) ([]FamilyMember, error) {
		return func(struct{}) ([]FamilyMember, error) {
			// Get member info before removal for event emission
			memberModel, err := GetByCharacterIdProvider(characterId)(p.db)()
			if err != nil {
				return []FamilyMember{}, err
			}
			
			// Execute business logic with message buffer
			result, err := p.RemoveMember(buf)(characterId, reason)()
			if err != nil {
				return []FamilyMember{}, err
			}
			
			// Add link broken events to buffer for all affected relationships
			if memberModel.HasSenior() {
				if putErr := buf.Put("FAMILY_STATUS", p.eventProducer.LinkBrokenEventProvider(transactionId, characterId, *memberModel.SeniorId(), characterId, reason)); putErr != nil {
					p.log.WithError(putErr).Error("Failed to add link broken event to buffer for senior")
				}
			}
			
			for _, juniorId := range memberModel.JuniorIds() {
				if putErr := buf.Put("FAMILY_STATUS", p.eventProducer.LinkBrokenEventProvider(transactionId, characterId, characterId, juniorId, reason)); putErr != nil {
					p.log.WithError(putErr).Error("Failed to add link broken event to buffer for junior")
				}
			}
			
			return result, nil
		}
	})
	return func() ([]FamilyMember, error) {
		return emitFunc(struct{}{})
	}
}

// BreakLinkAndEmit breaks a link and emits appropriate events
func (p *ProcessorImpl) BreakLinkAndEmit(transactionId string, characterId uint32, reason string) model.Provider[[]FamilyMember] {
	emitFunc := message.EmitWithResult[[]FamilyMember, struct{}](p.producer)(func(buf *message.Buffer) func(struct{}) ([]FamilyMember, error) {
		return func(struct{}) ([]FamilyMember, error) {
			// Get member info before breaking links for event emission
			memberModel, err := GetByCharacterIdProvider(characterId)(p.db)()
			if err != nil {
				return []FamilyMember{}, err
			}
			
			// Execute business logic with message buffer
			result, err := p.BreakLink(buf)(characterId, reason)()
			if err != nil {
				return []FamilyMember{}, err
			}
			
			// Add link broken events to buffer for all affected relationships
			if memberModel.HasSenior() {
				if putErr := buf.Put("FAMILY_STATUS", p.eventProducer.LinkBrokenEventProvider(transactionId, characterId, *memberModel.SeniorId(), characterId, reason)); putErr != nil {
					p.log.WithError(putErr).Error("Failed to add link broken event to buffer for senior")
				}
			}
			
			for _, juniorId := range memberModel.JuniorIds() {
				if putErr := buf.Put("FAMILY_STATUS", p.eventProducer.LinkBrokenEventProvider(transactionId, characterId, characterId, juniorId, reason)); putErr != nil {
					p.log.WithError(putErr).Error("Failed to add link broken event to buffer for junior")
				}
			}
			
			return result, nil
		}
	})
	return func() ([]FamilyMember, error) {
		return emitFunc(struct{}{})
	}
}

// AwardRepAndEmit awards reputation and emits appropriate events
func (p *ProcessorImpl) AwardRepAndEmit(transactionId string, characterId uint32, amount uint32, source string) model.Provider[FamilyMember] {
	emitFunc := message.EmitWithResult[FamilyMember, struct{}](p.producer)(func(buf *message.Buffer) func(struct{}) (FamilyMember, error) {
		return func(struct{}) (FamilyMember, error) {
			// Execute business logic with message buffer
			result, err := p.AwardRep(buf)(characterId, amount, source)()
			if err != nil {
				// Add error event to buffer
				if putErr := buf.Put("FAMILY_ERRORS", p.eventProducer.RepErrorEventProvider(transactionId, characterId, "AWARD_REP_FAILED", err.Error(), amount)); putErr != nil {
					p.log.WithError(putErr).Error("Failed to add rep error event to buffer")
				}
				return FamilyMember{}, err
			}
			
			// Add success event to buffer
			if putErr := buf.Put("FAMILY_REPUTATION", p.eventProducer.RepGainedEventProvider(transactionId, characterId, amount, result.DailyRep(), source)); putErr != nil {
				p.log.WithError(putErr).Error("Failed to add rep gained event to buffer")
			}
			
			return result, nil
		}
	})
	return func() (FamilyMember, error) {
		return emitFunc(struct{}{})
	}
}

// DeductRepAndEmit deducts reputation and emits appropriate events
func (p *ProcessorImpl) DeductRepAndEmit(transactionId string, characterId uint32, amount uint32, reason string) model.Provider[FamilyMember] {
	emitFunc := message.EmitWithResult[FamilyMember, struct{}](p.producer)(func(buf *message.Buffer) func(struct{}) (FamilyMember, error) {
		return func(struct{}) (FamilyMember, error) {
			// Execute business logic with message buffer
			result, err := p.DeductRep(buf)(characterId, amount, reason)()
			if err != nil {
				// Add error event to buffer
				if putErr := buf.Put("FAMILY_ERRORS", p.eventProducer.RepErrorEventProvider(transactionId, characterId, "DEDUCT_REP_FAILED", err.Error(), amount)); putErr != nil {
					p.log.WithError(putErr).Error("Failed to add rep error event to buffer")
				}
				return FamilyMember{}, err
			}
			
			// Add success event to buffer
			if putErr := buf.Put("FAMILY_REPUTATION", p.eventProducer.RepRedeemedEventProvider(transactionId, characterId, amount, reason)); putErr != nil {
				p.log.WithError(putErr).Error("Failed to add rep redeemed event to buffer")
			}
			
			return result, nil
		}
	})
	return func() (FamilyMember, error) {
		return emitFunc(struct{}{})
	}
}