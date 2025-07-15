package marriage

import (
	"context"
	"errors"
	"time"

	"atlas-marriages/character"
	"atlas-marriages/kafka/message"
	"atlas-marriages/kafka/message/marriage"
	"atlas-marriages/kafka/producer"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/Chronicle20/atlas-tenant"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

// EligibilityRequirement represents minimum level requirement for marriage
const EligibilityRequirement = 10

// Processor interface defines the proposal and ceremony processing operations
type Processor interface {
	WithCharacterProcessor(characterProcessor character.Processor) Processor

	// Proposal operations
	Propose(proposerId, targetId uint32) model.Provider[Proposal]
	ProposeAndEmit(transactionId uuid.UUID, proposerId, targetId uint32) (Proposal, error)

	// Eligibility checks
	CheckEligibility(characterId uint32) model.Provider[bool]
	CheckProposalEligibility(proposerId, targetId uint32) model.Provider[bool]

	// Cooldown operations
	CheckGlobalCooldown(proposerId uint32) model.Provider[bool]
	CheckPerTargetCooldown(proposerId, targetId uint32) model.Provider[bool]

	// Proposal queries
	GetActiveProposal(proposerId, targetId uint32) model.Provider[*Proposal]
	GetPendingProposalsByCharacter(characterId uint32) model.Provider[[]Proposal]
	GetProposalHistory(proposerId, targetId uint32) model.Provider[[]Proposal]

	// Ceremony operations
	ScheduleCeremony(marriageId uint32, scheduledAt time.Time, invitees []uint32) model.Provider[Ceremony]
	ScheduleCeremonyAndEmit(transactionId uuid.UUID, marriageId uint32, scheduledAt time.Time, invitees []uint32) (Ceremony, error)
	StartCeremony(ceremonyId uint32) model.Provider[Ceremony]
	StartCeremonyAndEmit(transactionId uuid.UUID, ceremonyId uint32) (Ceremony, error)
	CompleteCeremony(ceremonyId uint32) model.Provider[Ceremony]
	CompleteCeremonyAndEmit(transactionId uuid.UUID, ceremonyId uint32) (Ceremony, error)
	CancelCeremony(ceremonyId uint32) model.Provider[Ceremony]
	CancelCeremonyAndEmit(transactionId uuid.UUID, ceremonyId uint32) (Ceremony, error)
	PostponeCeremony(ceremonyId uint32) model.Provider[Ceremony]
	PostponeCeremonyAndEmit(transactionId uuid.UUID, ceremonyId uint32) (Ceremony, error)
	RescheduleCeremony(ceremonyId uint32, newScheduledAt time.Time) model.Provider[Ceremony]
	RescheduleCeremonyAndEmit(transactionId uuid.UUID, ceremonyId uint32, newScheduledAt time.Time) (Ceremony, error)

	// Ceremony invitee management
	AddInvitee(ceremonyId uint32, characterId uint32) model.Provider[Ceremony]
	AddInviteeAndEmit(transactionId uuid.UUID, ceremonyId uint32, characterId uint32) (Ceremony, error)
	RemoveInvitee(ceremonyId uint32, characterId uint32) model.Provider[Ceremony]
	RemoveInviteeAndEmit(transactionId uuid.UUID, ceremonyId uint32, characterId uint32) (Ceremony, error)

	// Ceremony queries
	GetCeremonyById(ceremonyId uint32) model.Provider[*Ceremony]
	GetCeremonyByMarriage(marriageId uint32) model.Provider[*Ceremony]
	GetUpcomingCeremonies() model.Provider[[]Ceremony]
	GetActiveCeremonies() model.Provider[[]Ceremony]
}

// ProcessorImpl implements the Processor interface
type ProcessorImpl struct {
	log                logrus.FieldLogger
	ctx                context.Context
	db                 *gorm.DB
	producer           producer.Provider
	characterProcessor character.Processor
}

// NewProcessor creates a new processor instance
func NewProcessor(log logrus.FieldLogger, ctx context.Context, db *gorm.DB) Processor {
	return &ProcessorImpl{
		log:                log,
		ctx:                ctx,
		db:                 db,
		producer:           producer.ProviderImpl(log)(ctx),
		characterProcessor: character.NewProcessor(log, ctx, db),
	}
}

// WithCharacterProcessor creates a new processor instance with a custom character processor for testing
func (p *ProcessorImpl) WithCharacterProcessor(characterProcessor character.Processor) Processor {
	return &ProcessorImpl{
		log:                p.log,
		ctx:                p.ctx,
		db:                 p.db,
		producer:           p.producer,
		characterProcessor: characterProcessor,
	}
}

// Propose creates a new marriage proposal with all eligibility checks
func (p *ProcessorImpl) Propose(proposerId, targetId uint32) model.Provider[Proposal] {
	return func() (Proposal, error) {
		p.log.WithFields(logrus.Fields{
			"proposerId": proposerId,
			"targetId":   targetId,
		}).Debug("Processing marriage proposal")

		// Check basic eligibility
		eligible, err := p.CheckProposalEligibility(proposerId, targetId)()
		if err != nil {
			return Proposal{}, err
		}
		if !eligible {
			return Proposal{}, errors.New("proposal eligibility check failed")
		}

		// Check global cooldown
		canPropose, err := p.CheckGlobalCooldown(proposerId)()
		if err != nil {
			return Proposal{}, err
		}
		if !canPropose {
			return Proposal{}, errors.New("proposer is in global cooldown period")
		}

		// Check per-target cooldown
		canProposeToTarget, err := p.CheckPerTargetCooldown(proposerId, targetId)()
		if err != nil {
			return Proposal{}, err
		}
		if !canProposeToTarget {
			return Proposal{}, errors.New("proposer is in cooldown period for this target")
		}

		// Get tenant from context
		t := tenant.MustFromContext(p.ctx)

		// Create proposal using administrator
		entityProvider := CreateProposal(p.db, p.log)(proposerId, targetId, t.Id())
		entity, err := entityProvider()
		if err != nil {
			return Proposal{}, err
		}

		// Transform entity to domain model
		proposal, err := MakeProposal(entity)
		if err != nil {
			return Proposal{}, err
		}

		p.log.WithFields(logrus.Fields{
			"proposalId": proposal.Id(),
			"proposerId": proposerId,
			"targetId":   targetId,
		}).Info("Marriage proposal created successfully")

		return proposal, nil
	}
}

// ProposeAndEmit creates a proposal and emits events
func (p *ProcessorImpl) ProposeAndEmit(transactionId uuid.UUID, proposerId, targetId uint32) (Proposal, error) {
	proposal, err := p.Propose(proposerId, targetId)()
	if err != nil {
		return Proposal{}, err
	}

	// Emit ProposalCreated event
	err = message.Emit(p.producer)(func(buf *message.Buffer) error {
		eventProvider := ProposalCreatedEventProvider(
			proposal.Id(),
			proposerId,
			targetId,
			proposal.ProposedAt(),
			proposal.ExpiresAt(),
		)
		return buf.Put(marriage.EnvEventTopicStatus, eventProvider)
	})
	if err != nil {
		return Proposal{}, err
	}

	p.log.WithFields(logrus.Fields{
		"transactionId": transactionId,
		"proposalId":    proposal.Id(),
	}).Debug("ProposalCreated event emitted")

	return proposal, nil
}

// CheckEligibility checks if a character meets the minimum level requirement
func (p *ProcessorImpl) CheckEligibility(characterId uint32) model.Provider[bool] {
	return func() (bool, error) {
		p.log.WithField("characterId", characterId).Debug("Checking character eligibility")

		// Get character information using character processor
		char, err := p.characterProcessor.GetById(characterId)
		if err != nil {
			p.log.WithError(err).WithField("characterId", characterId).Error("Failed to retrieve character")
			return false, err
		}

		// Check if character meets minimum level requirement
		if char.Level() < byte(EligibilityRequirement) {
			p.log.WithFields(logrus.Fields{
				"characterId": characterId,
				"level":       char.Level(),
				"required":    EligibilityRequirement,
			}).Debug("Character level too low for marriage")
			return false, nil
		}

		return true, nil
	}
}

// CheckProposalEligibility performs comprehensive eligibility checks for a proposal
func (p *ProcessorImpl) CheckProposalEligibility(proposerId, targetId uint32) model.Provider[bool] {
	return func() (bool, error) {
		// Get tenant from context
		t := tenant.MustFromContext(p.ctx)

		// Check basic character eligibility
		proposerEligible, err := p.CheckEligibility(proposerId)()
		if err != nil {
			return false, err
		}
		if !proposerEligible {
			return false, nil
		}

		targetEligible, err := p.CheckEligibility(targetId)()
		if err != nil {
			return false, err
		}
		if !targetEligible {
			return false, nil
		}

		// Check if proposer is already married or engaged
		proposerMarriageProvider := GetActiveMarriageByCharacterProvider(p.db, p.log)(proposerId, t.Id())
		proposerMarriage, err := proposerMarriageProvider()
		if err != nil {
			return false, err
		}
		if proposerMarriage != nil {
			return false, nil
		}

		// Check if target is already married or engaged
		targetMarriageProvider := GetActiveMarriageByCharacterProvider(p.db, p.log)(targetId, t.Id())
		targetMarriage, err := targetMarriageProvider()
		if err != nil {
			return false, err
		}
		if targetMarriage != nil {
			return false, nil
		}

		// Check if there's already a pending proposal between these characters
		existingProposalProvider := GetActiveProposalProvider(p.db, p.log)(proposerId, targetId, t.Id())
		existingProposal, err := existingProposalProvider()
		if err != nil {
			return false, err
		}
		if existingProposal != nil {
			return false, nil
		}

		return true, nil
	}
}

// CheckGlobalCooldown checks if the proposer is in global cooldown period
func (p *ProcessorImpl) CheckGlobalCooldown(proposerId uint32) model.Provider[bool] {
	return func() (bool, error) {
		t := tenant.MustFromContext(p.ctx)

		cooldownProvider := CheckGlobalCooldownProvider(p.db, p.log)(proposerId, t.Id())
		return cooldownProvider()
	}
}

// CheckPerTargetCooldown checks if the proposer is in cooldown period for specific target
func (p *ProcessorImpl) CheckPerTargetCooldown(proposerId, targetId uint32) model.Provider[bool] {
	return func() (bool, error) {
		t := tenant.MustFromContext(p.ctx)

		cooldownProvider := CheckPerTargetCooldownProvider(p.db, p.log)(proposerId, targetId, t.Id())
		return cooldownProvider()
	}
}

// GetActiveProposal retrieves an active proposal between two characters
func (p *ProcessorImpl) GetActiveProposal(proposerId, targetId uint32) model.Provider[*Proposal] {
	return func() (*Proposal, error) {
		t := tenant.MustFromContext(p.ctx)

		proposalProvider := GetActiveProposalProvider(p.db, p.log)(proposerId, targetId, t.Id())
		return proposalProvider()
	}
}

// GetPendingProposalsByCharacter retrieves all pending proposals for a character (sent or received)
func (p *ProcessorImpl) GetPendingProposalsByCharacter(characterId uint32) model.Provider[[]Proposal] {
	return func() ([]Proposal, error) {
		t := tenant.MustFromContext(p.ctx)

		proposalsProvider := GetPendingProposalsByCharacterProvider(p.db, p.log)(characterId, t.Id())
		return proposalsProvider()
	}
}

// GetProposalHistory retrieves the history of proposals between two characters
func (p *ProcessorImpl) GetProposalHistory(proposerId, targetId uint32) model.Provider[[]Proposal] {
	return func() ([]Proposal, error) {
		t := tenant.MustFromContext(p.ctx)

		historyProvider := GetProposalHistoryProvider(p.db, p.log)(proposerId, targetId, t.Id())
		return historyProvider()
	}
}

// EligibilityError represents specific eligibility validation errors
type EligibilityError struct {
	Code    string
	Message string
}

func (e EligibilityError) Error() string {
	return e.Message
}

// Predefined eligibility errors
var (
	ErrCharacterTooLowLevel = EligibilityError{
		Code:    "CHARACTER_TOO_LOW_LEVEL",
		Message: "character level is too low for marriage",
	}
	ErrCharacterAlreadyMarried = EligibilityError{
		Code:    "CHARACTER_ALREADY_MARRIED",
		Message: "character is already married or engaged",
	}
	ErrTargetAlreadyEngaged = EligibilityError{
		Code:    "TARGET_ALREADY_ENGAGED",
		Message: "target character already has a pending proposal",
	}
	ErrGlobalCooldownActive = EligibilityError{
		Code:    "GLOBAL_COOLDOWN_ACTIVE",
		Message: "proposer is in global cooldown period",
	}
	ErrTargetCooldownActive = EligibilityError{
		Code:    "TARGET_COOLDOWN_ACTIVE",
		Message: "proposer is in cooldown period for this target",
	}
)

// Ceremony-related processor methods

// ScheduleCeremony creates a new ceremony for an engaged marriage
func (p *ProcessorImpl) ScheduleCeremony(marriageId uint32, scheduledAt time.Time, invitees []uint32) model.Provider[Ceremony] {
	return func() (Ceremony, error) {
		p.log.WithFields(logrus.Fields{
			"marriageId":  marriageId,
			"scheduledAt": scheduledAt,
			"invitees":    len(invitees),
		}).Debug("Scheduling ceremony")

		// Validate invitees limit
		if len(invitees) > MaxInvitees {
			return Ceremony{}, errors.New("too many invitees, maximum is 15")
		}

		// Get tenant from context
		t := tenant.MustFromContext(p.ctx)

		// Verify marriage exists and is engaged
		marriageProvider := GetMarriageByIdProvider(p.db, p.log)(marriageId, t.Id())
		marriage, err := marriageProvider()
		if err != nil {
			return Ceremony{}, err
		}
		if marriage == nil {
			return Ceremony{}, errors.New("marriage not found")
		}
		if marriage.Status() != StatusEngaged {
			return Ceremony{}, errors.New("marriage must be engaged to schedule ceremony")
		}

		// Create ceremony using administrator
		entityProvider := CreateCeremony(p.db, p.log)(marriageId, marriage.CharacterId1(), marriage.CharacterId2(), scheduledAt, invitees, t.Id())
		entity, err := entityProvider()
		if err != nil {
			return Ceremony{}, err
		}

		// Transform entity to domain model
		ceremony, err := MakeCeremony(entity)
		if err != nil {
			return Ceremony{}, err
		}

		p.log.WithFields(logrus.Fields{
			"ceremonyId": ceremony.Id(),
			"marriageId": marriageId,
		}).Info("Ceremony scheduled successfully")

		return ceremony, nil
	}
}

// ScheduleCeremonyAndEmit schedules a ceremony and emits events
func (p *ProcessorImpl) ScheduleCeremonyAndEmit(transactionId uuid.UUID, marriageId uint32, scheduledAt time.Time, invitees []uint32) (Ceremony, error) {
	ceremony, err := p.ScheduleCeremony(marriageId, scheduledAt, invitees)()
	if err != nil {
		return Ceremony{}, err
	}

	// Emit CeremonyScheduled event
	err = message.Emit(p.producer)(func(buf *message.Buffer) error {
		eventProvider := CeremonyScheduledEventProvider(
			ceremony.Id(),
			marriageId,
			ceremony.CharacterId1(),
			ceremony.CharacterId2(),
			scheduledAt,
			invitees,
		)
		return buf.Put(marriage.EnvEventTopicStatus, eventProvider)
	})
	if err != nil {
		return Ceremony{}, err
	}

	p.log.WithFields(logrus.Fields{
		"transactionId": transactionId,
		"ceremonyId":    ceremony.Id(),
	}).Debug("CeremonyScheduled event emitted")

	return ceremony, nil
}

// StartCeremony transitions a ceremony to active state
func (p *ProcessorImpl) StartCeremony(ceremonyId uint32) model.Provider[Ceremony] {
	return func() (Ceremony, error) {
		p.log.WithField("ceremonyId", ceremonyId).Debug("Starting ceremony")

		// Get tenant from context
		t := tenant.MustFromContext(p.ctx)

		// Get ceremony
		ceremonyProvider := GetCeremonyByIdProvider(p.db, p.log)(ceremonyId, t.Id())
		ceremony, err := ceremonyProvider()
		if err != nil {
			return Ceremony{}, err
		}
		if ceremony == nil {
			return Ceremony{}, errors.New("ceremony not found")
		}

		// Validate state transition
		if !ceremony.CanStart() {
			return Ceremony{}, errors.New("ceremony cannot be started in current state")
		}

		// Start ceremony
		updatedCeremony, err := ceremony.Start()
		if err != nil {
			return Ceremony{}, err
		}

		// Update ceremony using administrator
		entityProvider := UpdateCeremony(p.db, p.log)(ceremonyId, updatedCeremony.ToEntity(), t.Id())
		entity, err := entityProvider()
		if err != nil {
			return Ceremony{}, err
		}

		// Transform entity to domain model
		result, err := MakeCeremony(entity)
		if err != nil {
			return Ceremony{}, err
		}

		p.log.WithField("ceremonyId", ceremonyId).Info("Ceremony started successfully")

		return result, nil
	}
}

// StartCeremonyAndEmit starts a ceremony and emits events
func (p *ProcessorImpl) StartCeremonyAndEmit(transactionId uuid.UUID, ceremonyId uint32) (Ceremony, error) {
	ceremony, err := p.StartCeremony(ceremonyId)()
	if err != nil {
		return Ceremony{}, err
	}

	// Emit CeremonyStarted event
	err = message.Emit(p.producer)(func(buf *message.Buffer) error {
		startedAt := time.Now()
		if ceremony.StartedAt() != nil {
			startedAt = *ceremony.StartedAt()
		}
		eventProvider := CeremonyStartedEventProvider(
			ceremony.Id(),
			ceremony.MarriageId(),
			ceremony.CharacterId1(),
			ceremony.CharacterId2(),
			startedAt,
		)
		return buf.Put(marriage.EnvEventTopicStatus, eventProvider)
	})
	if err != nil {
		return Ceremony{}, err
	}

	p.log.WithFields(logrus.Fields{
		"transactionId": transactionId,
		"ceremonyId":    ceremonyId,
	}).Debug("CeremonyStarted event emitted")

	return ceremony, nil
}

// CompleteCeremony transitions a ceremony to completed state
func (p *ProcessorImpl) CompleteCeremony(ceremonyId uint32) model.Provider[Ceremony] {
	return func() (Ceremony, error) {
		p.log.WithField("ceremonyId", ceremonyId).Debug("Completing ceremony")

		// Get tenant from context
		t := tenant.MustFromContext(p.ctx)

		// Get ceremony
		ceremonyProvider := GetCeremonyByIdProvider(p.db, p.log)(ceremonyId, t.Id())
		ceremony, err := ceremonyProvider()
		if err != nil {
			return Ceremony{}, err
		}
		if ceremony == nil {
			return Ceremony{}, errors.New("ceremony not found")
		}

		// Validate state transition
		if !ceremony.CanComplete() {
			return Ceremony{}, errors.New("ceremony cannot be completed in current state")
		}

		// Complete ceremony
		updatedCeremony, err := ceremony.Complete()
		if err != nil {
			return Ceremony{}, err
		}

		// Update ceremony using administrator
		entityProvider := UpdateCeremony(p.db, p.log)(ceremonyId, updatedCeremony.ToEntity(), t.Id())
		entity, err := entityProvider()
		if err != nil {
			return Ceremony{}, err
		}

		// Transform entity to domain model
		result, err := MakeCeremony(entity)
		if err != nil {
			return Ceremony{}, err
		}

		p.log.WithField("ceremonyId", ceremonyId).Info("Ceremony completed successfully")

		return result, nil
	}
}

// CompleteCeremonyAndEmit completes a ceremony and emits events
func (p *ProcessorImpl) CompleteCeremonyAndEmit(transactionId uuid.UUID, ceremonyId uint32) (Ceremony, error) {
	ceremony, err := p.CompleteCeremony(ceremonyId)()
	if err != nil {
		return Ceremony{}, err
	}

	// Emit CeremonyCompleted event
	err = message.Emit(p.producer)(func(buf *message.Buffer) error {
		completedAt := time.Now()
		if ceremony.CompletedAt() != nil {
			completedAt = *ceremony.CompletedAt()
		}
		eventProvider := CeremonyCompletedEventProvider(
			ceremony.Id(),
			ceremony.MarriageId(),
			ceremony.CharacterId1(),
			ceremony.CharacterId2(),
			completedAt,
		)
		return buf.Put(marriage.EnvEventTopicStatus, eventProvider)
	})
	if err != nil {
		return Ceremony{}, err
	}

	p.log.WithFields(logrus.Fields{
		"transactionId": transactionId,
		"ceremonyId":    ceremonyId,
	}).Debug("CeremonyCompleted event emitted")

	return ceremony, nil
}

// CancelCeremony transitions a ceremony to cancelled state
func (p *ProcessorImpl) CancelCeremony(ceremonyId uint32) model.Provider[Ceremony] {
	return func() (Ceremony, error) {
		p.log.WithField("ceremonyId", ceremonyId).Debug("Cancelling ceremony")

		// Get tenant from context
		t := tenant.MustFromContext(p.ctx)

		// Get ceremony
		ceremonyProvider := GetCeremonyByIdProvider(p.db, p.log)(ceremonyId, t.Id())
		ceremony, err := ceremonyProvider()
		if err != nil {
			return Ceremony{}, err
		}
		if ceremony == nil {
			return Ceremony{}, errors.New("ceremony not found")
		}

		// Validate state transition
		if !ceremony.CanCancel() {
			return Ceremony{}, errors.New("ceremony cannot be cancelled in current state")
		}

		// Cancel ceremony
		updatedCeremony, err := ceremony.Cancel()
		if err != nil {
			return Ceremony{}, err
		}

		// Update ceremony using administrator
		entityProvider := UpdateCeremony(p.db, p.log)(ceremonyId, updatedCeremony.ToEntity(), t.Id())
		entity, err := entityProvider()
		if err != nil {
			return Ceremony{}, err
		}

		// Transform entity to domain model
		result, err := MakeCeremony(entity)
		if err != nil {
			return Ceremony{}, err
		}

		p.log.WithField("ceremonyId", ceremonyId).Info("Ceremony cancelled successfully")

		return result, nil
	}
}

// CancelCeremonyAndEmit cancels a ceremony and emits events
func (p *ProcessorImpl) CancelCeremonyAndEmit(transactionId uuid.UUID, ceremonyId uint32) (Ceremony, error) {
	ceremony, err := p.CancelCeremony(ceremonyId)()
	if err != nil {
		return Ceremony{}, err
	}

	// Emit CeremonyCancelled event
	err = message.Emit(p.producer)(func(buf *message.Buffer) error {
		cancelledAt := time.Now()
		if ceremony.CancelledAt() != nil {
			cancelledAt = *ceremony.CancelledAt()
		}
		eventProvider := CeremonyCancelledEventProvider(
			ceremony.Id(),
			ceremony.MarriageId(),
			ceremony.CharacterId1(),
			ceremony.CharacterId2(),
			cancelledAt,
			0, // TODO: Track cancelled by character ID when needed
			"ceremony_cancelled", // TODO: Track cancel reason when needed
		)
		return buf.Put(marriage.EnvEventTopicStatus, eventProvider)
	})
	if err != nil {
		return Ceremony{}, err
	}

	p.log.WithFields(logrus.Fields{
		"transactionId": transactionId,
		"ceremonyId":    ceremonyId,
	}).Debug("CeremonyCancelled event emitted")

	return ceremony, nil
}

// PostponeCeremony transitions a ceremony to postponed state
func (p *ProcessorImpl) PostponeCeremony(ceremonyId uint32) model.Provider[Ceremony] {
	return func() (Ceremony, error) {
		p.log.WithField("ceremonyId", ceremonyId).Debug("Postponing ceremony")

		// Get tenant from context
		t := tenant.MustFromContext(p.ctx)

		// Get ceremony
		ceremonyProvider := GetCeremonyByIdProvider(p.db, p.log)(ceremonyId, t.Id())
		ceremony, err := ceremonyProvider()
		if err != nil {
			return Ceremony{}, err
		}
		if ceremony == nil {
			return Ceremony{}, errors.New("ceremony not found")
		}

		// Validate state transition
		if !ceremony.CanPostpone() {
			return Ceremony{}, errors.New("ceremony cannot be postponed in current state")
		}

		// Postpone ceremony
		updatedCeremony, err := ceremony.Postpone()
		if err != nil {
			return Ceremony{}, err
		}

		// Update ceremony using administrator
		entityProvider := UpdateCeremony(p.db, p.log)(ceremonyId, updatedCeremony.ToEntity(), t.Id())
		entity, err := entityProvider()
		if err != nil {
			return Ceremony{}, err
		}

		// Transform entity to domain model
		result, err := MakeCeremony(entity)
		if err != nil {
			return Ceremony{}, err
		}

		p.log.WithField("ceremonyId", ceremonyId).Info("Ceremony postponed successfully")

		return result, nil
	}
}

// PostponeCeremonyAndEmit postpones a ceremony and emits events
func (p *ProcessorImpl) PostponeCeremonyAndEmit(transactionId uuid.UUID, ceremonyId uint32) (Ceremony, error) {
	ceremony, err := p.PostponeCeremony(ceremonyId)()
	if err != nil {
		return Ceremony{}, err
	}

	// Emit CeremonyPostponed event
	err = message.Emit(p.producer)(func(buf *message.Buffer) error {
		postponedAt := time.Now()
		if ceremony.PostponedAt() != nil {
			postponedAt = *ceremony.PostponedAt()
		}
		eventProvider := CeremonyPostponedEventProvider(
			ceremony.Id(),
			ceremony.MarriageId(),
			ceremony.CharacterId1(),
			ceremony.CharacterId2(),
			postponedAt,
			"ceremony_postponed", // TODO: Track postpone reason when needed
		)
		return buf.Put(marriage.EnvEventTopicStatus, eventProvider)
	})
	if err != nil {
		return Ceremony{}, err
	}

	p.log.WithFields(logrus.Fields{
		"transactionId": transactionId,
		"ceremonyId":    ceremonyId,
	}).Debug("CeremonyPostponed event emitted")

	return ceremony, nil
}

// RescheduleCeremony reschedules a ceremony to a new time
func (p *ProcessorImpl) RescheduleCeremony(ceremonyId uint32, newScheduledAt time.Time) model.Provider[Ceremony] {
	return func() (Ceremony, error) {
		p.log.WithFields(logrus.Fields{
			"ceremonyId":      ceremonyId,
			"newScheduledAt": newScheduledAt,
		}).Debug("Rescheduling ceremony")

		// Get tenant from context
		t := tenant.MustFromContext(p.ctx)

		// Get ceremony
		ceremonyProvider := GetCeremonyByIdProvider(p.db, p.log)(ceremonyId, t.Id())
		ceremony, err := ceremonyProvider()
		if err != nil {
			return Ceremony{}, err
		}
		if ceremony == nil {
			return Ceremony{}, errors.New("ceremony not found")
		}

		// Validate state transition
		if !ceremony.CanReschedule() {
			return Ceremony{}, errors.New("ceremony cannot be rescheduled in current state")
		}

		// Reschedule ceremony
		updatedCeremony, err := ceremony.Reschedule(newScheduledAt)
		if err != nil {
			return Ceremony{}, err
		}

		// Update ceremony using administrator
		entityProvider := UpdateCeremony(p.db, p.log)(ceremonyId, updatedCeremony.ToEntity(), t.Id())
		entity, err := entityProvider()
		if err != nil {
			return Ceremony{}, err
		}

		// Transform entity to domain model
		result, err := MakeCeremony(entity)
		if err != nil {
			return Ceremony{}, err
		}

		p.log.WithField("ceremonyId", ceremonyId).Info("Ceremony rescheduled successfully")

		return result, nil
	}
}

// RescheduleCeremonyAndEmit reschedules a ceremony and emits events
func (p *ProcessorImpl) RescheduleCeremonyAndEmit(transactionId uuid.UUID, ceremonyId uint32, newScheduledAt time.Time) (Ceremony, error) {
	ceremony, err := p.RescheduleCeremony(ceremonyId, newScheduledAt)()
	if err != nil {
		return Ceremony{}, err
	}

	// Emit CeremonyRescheduled event
	err = message.Emit(p.producer)(func(buf *message.Buffer) error {
		rescheduledAt := time.Now()
		eventProvider := CeremonyRescheduledEventProvider(
			ceremony.Id(),
			ceremony.MarriageId(),
			ceremony.CharacterId1(),
			ceremony.CharacterId2(),
			rescheduledAt,
			newScheduledAt,
			0, // TODO: Track rescheduled by character ID when needed
		)
		return buf.Put(marriage.EnvEventTopicStatus, eventProvider)
	})
	if err != nil {
		return Ceremony{}, err
	}

	p.log.WithFields(logrus.Fields{
		"transactionId": transactionId,
		"ceremonyId":    ceremonyId,
	}).Debug("CeremonyRescheduled event emitted")

	return ceremony, nil
}

// AddInvitee adds an invitee to a ceremony
func (p *ProcessorImpl) AddInvitee(ceremonyId uint32, characterId uint32) model.Provider[Ceremony] {
	return func() (Ceremony, error) {
		p.log.WithFields(logrus.Fields{
			"ceremonyId":  ceremonyId,
			"characterId": characterId,
		}).Debug("Adding invitee to ceremony")

		// Get tenant from context
		t := tenant.MustFromContext(p.ctx)

		// Get ceremony
		ceremonyProvider := GetCeremonyByIdProvider(p.db, p.log)(ceremonyId, t.Id())
		ceremony, err := ceremonyProvider()
		if err != nil {
			return Ceremony{}, err
		}
		if ceremony == nil {
			return Ceremony{}, errors.New("ceremony not found")
		}

		// Validate invitee addition
		if !ceremony.CanAddInvitee(characterId) {
			return Ceremony{}, errors.New("invitee cannot be added to ceremony")
		}

		// Add invitee
		updatedCeremony, err := ceremony.AddInvitee(characterId)
		if err != nil {
			return Ceremony{}, err
		}

		// Update ceremony using administrator
		entityProvider := UpdateCeremony(p.db, p.log)(ceremonyId, updatedCeremony.ToEntity(), t.Id())
		entity, err := entityProvider()
		if err != nil {
			return Ceremony{}, err
		}

		// Transform entity to domain model
		result, err := MakeCeremony(entity)
		if err != nil {
			return Ceremony{}, err
		}

		p.log.WithFields(logrus.Fields{
			"ceremonyId":  ceremonyId,
			"characterId": characterId,
		}).Info("Invitee added to ceremony successfully")

		return result, nil
	}
}

// AddInviteeAndEmit adds an invitee and emits events
func (p *ProcessorImpl) AddInviteeAndEmit(transactionId uuid.UUID, ceremonyId uint32, characterId uint32) (Ceremony, error) {
	ceremony, err := p.AddInvitee(ceremonyId, characterId)()
	if err != nil {
		return Ceremony{}, err
	}

	// TODO: Emit InviteeAdded event when specific invitee events are defined
	// For now, we could emit a generic ceremony updated event
	p.log.WithFields(logrus.Fields{
		"transactionId": transactionId,
		"ceremonyId":    ceremonyId,
		"characterId":   characterId,
	}).Debug("Invitee added - would emit InviteeAdded event")

	return ceremony, nil
}

// RemoveInvitee removes an invitee from a ceremony
func (p *ProcessorImpl) RemoveInvitee(ceremonyId uint32, characterId uint32) model.Provider[Ceremony] {
	return func() (Ceremony, error) {
		p.log.WithFields(logrus.Fields{
			"ceremonyId":  ceremonyId,
			"characterId": characterId,
		}).Debug("Removing invitee from ceremony")

		// Get tenant from context
		t := tenant.MustFromContext(p.ctx)

		// Get ceremony
		ceremonyProvider := GetCeremonyByIdProvider(p.db, p.log)(ceremonyId, t.Id())
		ceremony, err := ceremonyProvider()
		if err != nil {
			return Ceremony{}, err
		}
		if ceremony == nil {
			return Ceremony{}, errors.New("ceremony not found")
		}

		// Validate invitee removal
		if !ceremony.CanRemoveInvitee(characterId) {
			return Ceremony{}, errors.New("invitee cannot be removed from ceremony")
		}

		// Remove invitee
		updatedCeremony, err := ceremony.RemoveInvitee(characterId)
		if err != nil {
			return Ceremony{}, err
		}

		// Update ceremony using administrator
		entityProvider := UpdateCeremony(p.db, p.log)(ceremonyId, updatedCeremony.ToEntity(), t.Id())
		entity, err := entityProvider()
		if err != nil {
			return Ceremony{}, err
		}

		// Transform entity to domain model
		result, err := MakeCeremony(entity)
		if err != nil {
			return Ceremony{}, err
		}

		p.log.WithFields(logrus.Fields{
			"ceremonyId":  ceremonyId,
			"characterId": characterId,
		}).Info("Invitee removed from ceremony successfully")

		return result, nil
	}
}

// RemoveInviteeAndEmit removes an invitee and emits events
func (p *ProcessorImpl) RemoveInviteeAndEmit(transactionId uuid.UUID, ceremonyId uint32, characterId uint32) (Ceremony, error) {
	ceremony, err := p.RemoveInvitee(ceremonyId, characterId)()
	if err != nil {
		return Ceremony{}, err
	}

	// TODO: Emit InviteeRemoved event when specific invitee events are defined
	// For now, we could emit a generic ceremony updated event
	p.log.WithFields(logrus.Fields{
		"transactionId": transactionId,
		"ceremonyId":    ceremonyId,
		"characterId":   characterId,
	}).Debug("Invitee removed - would emit InviteeRemoved event")

	return ceremony, nil
}

// GetCeremonyById retrieves a ceremony by its ID
func (p *ProcessorImpl) GetCeremonyById(ceremonyId uint32) model.Provider[*Ceremony] {
	return func() (*Ceremony, error) {
		t := tenant.MustFromContext(p.ctx)

		ceremonyProvider := GetCeremonyByIdProvider(p.db, p.log)(ceremonyId, t.Id())
		return ceremonyProvider()
	}
}

// GetCeremonyByMarriage retrieves a ceremony by its associated marriage ID
func (p *ProcessorImpl) GetCeremonyByMarriage(marriageId uint32) model.Provider[*Ceremony] {
	return func() (*Ceremony, error) {
		t := tenant.MustFromContext(p.ctx)

		ceremonyProvider := GetCeremonyByMarriageProvider(p.db, p.log)(marriageId, t.Id())
		return ceremonyProvider()
	}
}

// GetUpcomingCeremonies retrieves all upcoming ceremonies
func (p *ProcessorImpl) GetUpcomingCeremonies() model.Provider[[]Ceremony] {
	return func() ([]Ceremony, error) {
		t := tenant.MustFromContext(p.ctx)

		ceremoniesProvider := GetUpcomingCeremoniesProvider(p.db, p.log)(t.Id())
		return ceremoniesProvider()
	}
}

// GetActiveCeremonies retrieves all active ceremonies
func (p *ProcessorImpl) GetActiveCeremonies() model.Provider[[]Ceremony] {
	return func() ([]Ceremony, error) {
		t := tenant.MustFromContext(p.ctx)

		ceremoniesProvider := GetActiveCeremoniesProvider(p.db, p.log)(t.Id())
		return ceremoniesProvider()
	}
}
