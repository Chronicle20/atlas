package family

import (
	"context"
	"errors"

	"atlas-family/kafka/message"
	familymsg "atlas-family/kafka/message/family"
	"atlas-family/kafka/producer"

	"github.com/Chronicle20/atlas-constants/world"
	"github.com/Chronicle20/atlas-model/model"
	tenant "github.com/Chronicle20/atlas-tenant"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

// Processor interface defines the core business logic operations
type Processor interface {
	WithTransaction(db *gorm.DB) Processor
	AddJunior(buf *message.Buffer) func(worldId world.Id, seniorId uint32, seniorLevel uint16, juniorId uint32, juniorLevel uint16) model.Provider[FamilyMember]
	RemoveMember(buf *message.Buffer) func(characterId uint32, reason string) model.Provider[[]FamilyMember]
	BreakLink(buf *message.Buffer) func(characterId uint32, reason string) model.Provider[[]FamilyMember]
	AwardRep(buf *message.Buffer) func(characterId uint32, amount uint32, source string) model.Provider[FamilyMember]
	DeductRep(buf *message.Buffer) func(characterId uint32, amount uint32, reason string) model.Provider[FamilyMember]
	ResetDailyRep(buf *message.Buffer) model.Provider[BatchResetResult]

	// AndEmit variants for Kafka message emission
	AddJuniorAndEmit(transactionId uuid.UUID, worldId world.Id, seniorId uint32, seniorLevel uint16, juniorId uint32, juniorLevel uint16) model.Provider[FamilyMember]
	RemoveMemberAndEmit(transactionId uuid.UUID, characterId uint32, reason string) model.Provider[[]FamilyMember]
	BreakLinkAndEmit(transactionId uuid.UUID, characterId uint32, reason string) model.Provider[[]FamilyMember]
	AwardRepAndEmit(transactionId uuid.UUID, characterId uint32, amount uint32, source string) model.Provider[FamilyMember]
	DeductRepAndEmit(transactionId uuid.UUID, characterId uint32, amount uint32, reason string) model.Provider[FamilyMember]

	GetFamilyTree(characterId uint32) ([]FamilyMember, error)
	GetByCharacterId(characterId uint32) (FamilyMember, error)
}

// ProcessorImpl implements the Processor interface
type ProcessorImpl struct {
	log      logrus.FieldLogger
	ctx      context.Context
	db       *gorm.DB
	producer producer.Provider
}

// NewProcessor creates a new processor instance
func NewProcessor(l logrus.FieldLogger, ctx context.Context, db *gorm.DB) Processor {
	return &ProcessorImpl{
		log:      l,
		ctx:      ctx,
		db:       db,
		producer: producer.ProviderImpl(l)(ctx),
	}
}

// Business logic errors
var (
	ErrMemberNotFound          = errors.New("family member not found")
	ErrSeniorNotFound          = errors.New("senior member not found")
	ErrJuniorNotFound          = errors.New("junior member not found")
	ErrSeniorHasTooManyJuniors = errors.New("senior already has maximum number of juniors")
	ErrJuniorAlreadyLinked     = errors.New("junior is already linked to a senior")
	ErrLevelDifferenceTooLarge = errors.New("level difference exceeds maximum allowed")
	ErrNotOnSameMap            = errors.New("members must be on the same map to link")
	ErrInsufficientRep         = errors.New("insufficient reputation for operation")
	ErrRepCapExceeded          = errors.New("daily reputation cap exceeded")
	ErrCannotRemoveSelf        = errors.New("cannot remove self from family")
	ErrNoLinkToBreak           = errors.New("no family link exists to break")
)

func (p *ProcessorImpl) WithTransaction(db *gorm.DB) Processor {
	return &ProcessorImpl{
		log:      p.log,
		ctx:      p.ctx,
		db:       db,
		producer: p.producer,
	}
}

// AddJunior adds a junior to a senior's family
func (p *ProcessorImpl) AddJunior(buf *message.Buffer) func(worldId world.Id, seniorId uint32, seniorLevel uint16, juniorId uint32, juniorLevel uint16) model.Provider[FamilyMember] {
	return func(worldId world.Id, seniorId uint32, seniorLevel uint16, juniorId uint32, juniorLevel uint16) model.Provider[FamilyMember] {
		return func() (FamilyMember, error) {
			p.log.WithFields(logrus.Fields{
				"seniorId": seniorId,
				"juniorId": juniorId,
			}).Info("Adding junior to family")

			// Validate input
			if seniorId == juniorId {
				if buf != nil {
					if putErr := buf.Put(familymsg.EnvEventTopicErrors, LinkErrorEventProvider(0, seniorId, seniorId, juniorId, "SELF_REFERENCE", ErrSelfReference.Error())); putErr != nil {
						p.log.WithError(putErr).Error("Failed to add link error event to buffer")
					}
				}
				return FamilyMember{}, ErrSelfReference
			}

			// Get senior member
			seniorModel, err := p.GetByCharacterId(seniorId)
			if err != nil {
				if errors.Is(err, ErrMemberNotFound) {
					t := tenant.MustFromContext(p.ctx)
					seniorModel, err = model.Map(Make)(CreateMember(p.db, p.log)(seniorId, t.Id(), seniorLevel, worldId))()
					if buf != nil {
						if putErr := buf.Put(familymsg.EnvEventTopicErrors, LinkErrorEventProvider(0, seniorId, seniorId, juniorId, "SENIOR_NOT_FOUND", ErrSeniorNotFound.Error())); putErr != nil {
							p.log.WithError(putErr).Error("Failed to add link error event to buffer")
						}
					}
					return FamilyMember{}, ErrSeniorNotFound
				}
				return FamilyMember{}, err
			}

			// Check if senior can add more juniors
			if !seniorModel.CanAddJunior() {
				if buf != nil {
					if putErr := buf.Put(familymsg.EnvEventTopicErrors, LinkErrorEventProvider(seniorModel.World(), seniorId, seniorId, juniorId, "TOO_MANY_JUNIORS", ErrSeniorHasTooManyJuniors.Error())); putErr != nil {
						p.log.WithError(putErr).Error("Failed to add link error event to buffer")
					}
				}
				return FamilyMember{}, ErrSeniorHasTooManyJuniors
			}

			// Get junior member
			juniorModel, err := p.GetByCharacterId(juniorId)
			if err != nil {
				if errors.Is(err, ErrMemberNotFound) {
					if buf != nil {
						if putErr := buf.Put(familymsg.EnvEventTopicErrors, LinkErrorEventProvider(seniorModel.World(), seniorId, seniorId, juniorId, "JUNIOR_NOT_FOUND", ErrJuniorNotFound.Error())); putErr != nil {
							p.log.WithError(putErr).Error("Failed to add link error event to buffer")
						}
					}
					return FamilyMember{}, ErrJuniorNotFound
				}
				return FamilyMember{}, err
			}

			// Check if junior already has a senior
			if juniorModel.HasSenior() {
				if buf != nil {
					if putErr := buf.Put(familymsg.EnvEventTopicErrors, LinkErrorEventProvider(seniorModel.World(), seniorId, seniorId, juniorId, "JUNIOR_ALREADY_LINKED", ErrJuniorAlreadyLinked.Error())); putErr != nil {
						p.log.WithError(putErr).Error("Failed to add link error event to buffer")
					}
				}
				return FamilyMember{}, ErrJuniorAlreadyLinked
			}

			// Validate level difference
			if !seniorModel.ValidateLevelDifference(juniorModel.Level()) {
				if buf != nil {
					if putErr := buf.Put(familymsg.EnvEventTopicErrors, LinkErrorEventProvider(seniorModel.World(), seniorId, seniorId, juniorId, "LEVEL_DIFFERENCE_TOO_LARGE", ErrLevelDifferenceTooLarge.Error())); putErr != nil {
						p.log.WithError(putErr).Error("Failed to add link error event to buffer")
					}
				}
				return FamilyMember{}, ErrLevelDifferenceTooLarge
			}

			// Validate same world
			if !seniorModel.IsSameWorld(juniorModel.World()) {
				if buf != nil {
					if putErr := buf.Put(familymsg.EnvEventTopicErrors, LinkErrorEventProvider(seniorModel.World(), seniorId, seniorId, juniorId, "NOT_ON_SAME_MAP", ErrNotOnSameMap.Error())); putErr != nil {
						p.log.WithError(putErr).Error("Failed to add link error event to buffer")
					}
				}
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
				if _, err := SaveMember(tx, p.log)(updatedSenior)(); err != nil {
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
				if _, err := SaveMember(tx, p.log)(updatedJunior)(); err != nil {
					return err
				}

				result = updatedSenior
				return nil
			})

			if err != nil {
				// Add error event to buffer if provided
				if buf != nil {
					if putErr := buf.Put(familymsg.EnvEventTopicErrors, LinkErrorEventProvider(seniorModel.World(), seniorId, seniorId, juniorId, "ADD_JUNIOR_FAILED", err.Error())); putErr != nil {
						p.log.WithError(putErr).Error("Failed to add link error event to buffer")
					}
				}
				return FamilyMember{}, err
			}

			// Add success event to buffer if provided
			if buf != nil {
				if putErr := buf.Put(familymsg.EnvEventTopicStatus, LinkCreatedEventProvider(result.World(), seniorId, seniorId, juniorId)); putErr != nil {
					p.log.WithError(putErr).Error("Failed to add link created event to buffer")
				}
			}

			return result, nil
		}
	}
}

// RemoveMember removes a member from the family and handles cascade operations
func (p *ProcessorImpl) RemoveMember(_ *message.Buffer) func(characterId uint32, reason string) model.Provider[[]FamilyMember] {
	return func(characterId uint32, reason string) model.Provider[[]FamilyMember] {
		return func() ([]FamilyMember, error) {
			p.log.WithFields(logrus.Fields{
				"characterId": characterId,
				"reason":      reason,
			}).Info("Removing member from family")

			// Get the member to remove
			memberModel, err := p.GetByCharacterId(characterId)
			if err != nil {
				return []FamilyMember{}, err
			}

			var updatedMembers []FamilyMember
			err = p.db.Transaction(func(tx *gorm.DB) error {
				// If member has a senior, remove from senior's junior list
				if memberModel.HasSenior() {
					if seniorModel, err := p.GetByCharacterId(*memberModel.SeniorId()); err == nil {
						updatedSenior, err := seniorModel.Builder().
							RemoveJunior(characterId).
							Touch().
							Build()
						if err != nil {
							return err
						}

						if _, err := SaveMember(tx, p.log)(updatedSenior)(); err != nil {
							return err
						}
						updatedMembers = append(updatedMembers, updatedSenior)
					}
				}

				// If member has juniors, remove their senior reference
				if memberModel.HasJuniors() {
					for _, juniorId := range memberModel.JuniorIds() {
						if juniorModel, err := p.GetByCharacterId(juniorId); err == nil {
							updatedJunior, err := juniorModel.Builder().
								ClearSeniorId().
								Touch().
								Build()
							if err != nil {
								return err
							}

							if _, err := SaveMember(tx, p.log)(updatedJunior)(); err != nil {
								return err
							}
							updatedMembers = append(updatedMembers, updatedJunior)
						}
					}
				}

				// Remove the member
				if _, err := DeleteMember(tx, p.log)(characterId)(); err != nil {
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
			memberModel, err := p.GetByCharacterId(characterId)
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
					if seniorModel, err := p.WithTransaction(tx).GetByCharacterId(*memberModel.SeniorId()); err == nil {
						updatedSenior, err := seniorModel.Builder().
							RemoveJunior(characterId).
							Touch().
							Build()
						if err != nil {
							return err
						}

						if _, err := SaveMember(tx, p.log)(updatedSenior)(); err != nil {
							return err
						}
						updatedMembers = append(updatedMembers, updatedSenior)
					}
					// Clear member's senior reference
					updatedMember, err := memberModel.Builder().
						ClearSeniorId().
						Touch().
						Build()
					if err != nil {
						return err
					}

					if _, err := SaveMember(tx, p.log)(updatedMember)(); err != nil {
						return err
					}
					updatedMembers = append(updatedMembers, updatedMember)
				}

				// If member has juniors, clear their senior reference
				if memberModel.HasJuniors() {
					for _, juniorId := range memberModel.JuniorIds() {
						if juniorModel, err := p.WithTransaction(tx).GetByCharacterId(juniorId); err == nil {
							updatedJunior, err := juniorModel.Builder().
								ClearSeniorId().
								Touch().
								Build()
							if err != nil {
								return err
							}

							if _, err := SaveMember(tx, p.log)(updatedJunior)(); err != nil {
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

					if _, err := SaveMember(tx, p.log)(updatedMember)(); err != nil {
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

			if err != nil {
				return []FamilyMember{}, err
			}

			// Add link broken events to buffer for all affected relationships
			if buf != nil {
				if memberModel.HasSenior() {
					if putErr := buf.Put(familymsg.EnvEventTopicStatus, LinkBrokenEventProvider(memberModel.World(), characterId, *memberModel.SeniorId(), characterId, reason)); putErr != nil {
						p.log.WithError(putErr).Error("Failed to add link broken event to buffer for senior")
					}
				}

				for _, juniorId := range memberModel.JuniorIds() {
					if putErr := buf.Put(familymsg.EnvEventTopicStatus, LinkBrokenEventProvider(memberModel.World(), characterId, characterId, juniorId, reason)); putErr != nil {
						p.log.WithError(putErr).Error("Failed to add link broken event to buffer for junior")
					}
				}
			}

			return updatedMembers, nil
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
			memberModel, err := p.GetByCharacterId(characterId)
			if err != nil {
				return FamilyMember{}, err
			}

			// Check if member can receive more rep today
			if !memberModel.CanReceiveRep(amount) {
				// Add error event to buffer if provided
				if buf != nil {
					if putErr := buf.Put(familymsg.EnvEventTopicErrors, RepErrorEventProvider(memberModel.World(), characterId, "AWARD_REP_FAILED", ErrRepCapExceeded.Error(), amount)); putErr != nil {
						p.log.WithError(putErr).Error("Failed to add rep error event to buffer")
					}
				}
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

			if _, err := SaveMember(p.db, p.log)(updatedMember)(); err != nil {
				return FamilyMember{}, err
			}

			// Add success event to buffer if provided
			if buf != nil {
				if putErr := buf.Put(familymsg.EnvEventTopicRep, RepGainedEventProvider(updatedMember.World(), characterId, amount, updatedMember.DailyRep(), source)); putErr != nil {
					p.log.WithError(putErr).Error("Failed to add rep gained event to buffer")
				}
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
			memberModel, err := p.GetByCharacterId(characterId)
			if err != nil {
				return FamilyMember{}, err
			}

			// Check if member has enough rep
			if memberModel.Rep() < amount {
				// Add error event to buffer if provided
				if buf != nil {
					if putErr := buf.Put(familymsg.EnvEventTopicErrors, RepErrorEventProvider(memberModel.World(), characterId, "DEDUCT_REP_FAILED", ErrInsufficientRep.Error(), amount)); putErr != nil {
						p.log.WithError(putErr).Error("Failed to add rep error event to buffer")
					}
				}
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

			if _, err := SaveMember(p.db, p.log)(updatedMember)(); err != nil {
				return FamilyMember{}, err
			}

			// Add success event to buffer if provided
			if buf != nil {
				if putErr := buf.Put(familymsg.EnvEventTopicRep, RepRedeemedEventProvider(updatedMember.World(), characterId, amount, reason)); putErr != nil {
					p.log.WithError(putErr).Error("Failed to add rep redeemed event to buffer")
				}
			}

			return updatedMember, nil
		}
	}
}

// ResetDailyRep resets daily reputation for all members
func (p *ProcessorImpl) ResetDailyRep(buf *message.Buffer) model.Provider[BatchResetResult] {
	return func() (BatchResetResult, error) {
		p.log.Info("Resetting daily reputation for all members")

		result, err := BatchResetDailyRep(p.db, p.log)()
		if err != nil {
			_ = buf.Put(familymsg.EnvEventTopicErrors, RepErrorEventProvider(0, 0, "RESET_FAILED", err.Error(), 0))
			return BatchResetResult{}, err
		}

		_ = buf.Put(familymsg.EnvEventTopicRep, RepResetEventProvider(0, 0, uint32(result.AffectedCount)))
		return result, nil
	}
}

// AndEmit variants - combine business logic with event emission

// AddJuniorAndEmit adds a junior and emits appropriate events
func (p *ProcessorImpl) AddJuniorAndEmit(_ uuid.UUID, worldId world.Id, seniorId uint32, seniorLevel uint16, juniorId uint32, juniorLevel uint16) model.Provider[FamilyMember] {
	return func() (FamilyMember, error) {
		return message.EmitWithResult[FamilyMember, struct{}](p.producer)(func(buf *message.Buffer) func(struct{}) (FamilyMember, error) {
			return func(struct{}) (FamilyMember, error) {
				// Use base function which handles event emission
				return p.AddJunior(buf)(worldId, seniorId, seniorLevel, juniorId, juniorLevel)()
			}
		})(struct{}{})
	}
}

// RemoveMemberAndEmit removes a member and emits appropriate events
func (p *ProcessorImpl) RemoveMemberAndEmit(_ uuid.UUID, characterId uint32, reason string) model.Provider[[]FamilyMember] {
	return func() ([]FamilyMember, error) {
		return message.EmitWithResult[[]FamilyMember, struct{}](p.producer)(func(buf *message.Buffer) func(struct{}) ([]FamilyMember, error) {
			return func(struct{}) ([]FamilyMember, error) {
				// Use base function which handles event emission
				return p.RemoveMember(buf)(characterId, reason)()
			}
		})(struct{}{})
	}
}

// BreakLinkAndEmit breaks a link and emits appropriate events
func (p *ProcessorImpl) BreakLinkAndEmit(_ uuid.UUID, characterId uint32, reason string) model.Provider[[]FamilyMember] {
	return func() ([]FamilyMember, error) {
		return message.EmitWithResult[[]FamilyMember, struct{}](p.producer)(func(buf *message.Buffer) func(struct{}) ([]FamilyMember, error) {
			return func(struct{}) ([]FamilyMember, error) {
				// Use base function which handles event emission
				return p.BreakLink(buf)(characterId, reason)()
			}
		})(struct{}{})
	}
}

// AwardRepAndEmit awards reputation and emits appropriate events
func (p *ProcessorImpl) AwardRepAndEmit(_ uuid.UUID, characterId uint32, amount uint32, source string) model.Provider[FamilyMember] {
	return func() (FamilyMember, error) {
		return message.EmitWithResult[FamilyMember, struct{}](p.producer)(func(buf *message.Buffer) func(struct{}) (FamilyMember, error) {
			return func(struct{}) (FamilyMember, error) {
				// Use base function which handles event emission
				return p.AwardRep(buf)(characterId, amount, source)()
			}
		})(struct{}{})
	}
}

// DeductRepAndEmit deducts reputation and emits appropriate events
func (p *ProcessorImpl) DeductRepAndEmit(_ uuid.UUID, characterId uint32, amount uint32, reason string) model.Provider[FamilyMember] {
	return func() (FamilyMember, error) {
		return message.EmitWithResult[FamilyMember, struct{}](p.producer)(func(buf *message.Buffer) func(struct{}) (FamilyMember, error) {
			return func(struct{}) (FamilyMember, error) {
				// Use base function which handles event emission
				return p.DeductRep(buf)(characterId, amount, reason)()
			}
		})(struct{}{})
	}
}

func (p *ProcessorImpl) GetFamilyTree(characterId uint32) ([]FamilyMember, error) {
	return model.SliceMap(Make)(GetFamilyTreeProvider(characterId)(p.db))(model.ParallelMap())()
}

func (p *ProcessorImpl) GetByCharacterId(characterId uint32) (FamilyMember, error) {
	return model.Map(Make)(GetByIdProvider(characterId)(p.db))()
}
