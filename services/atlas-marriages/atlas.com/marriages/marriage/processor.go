package marriage

import (
	"context"
	"errors"
	"time"

	"atlas-marriages/character"
	"atlas-marriages/kafka/message"
	marriageMsg "atlas-marriages/kafka/message/marriage"
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
	WithProducer(producer producer.Provider) Processor
	WithCharacterProcessor(characterProcessor character.Processor) Processor

	// Proposal operations
	Propose(proposerId, targetId uint32) model.Provider[Proposal]
	ProposeAndEmit(transactionId uuid.UUID, proposerId, targetId uint32) (Proposal, error)
	AcceptProposal(proposalId uint32) model.Provider[Marriage]
	AcceptProposalAndEmit(transactionId uuid.UUID, proposalId uint32) (Marriage, error)
	DeclineProposal(proposalId uint32) model.Provider[Proposal]
	DeclineProposalAndEmit(transactionId uuid.UUID, proposalId uint32) (Proposal, error)
	CancelProposal(proposalId uint32) model.Provider[Proposal]
	CancelProposalAndEmit(transactionId uuid.UUID, proposalId uint32) (Proposal, error)

	// Marriage operations
	Divorce(marriageId uint32, initiatedBy uint32) model.Provider[Marriage]
	DivorceAndEmit(transactionId uuid.UUID, marriageId uint32, initiatedBy uint32) (Marriage, error)

	// Character deletion handling
	HandleCharacterDeletion(characterId uint32) error
	HandleCharacterDeletionAndEmit(transactionId uuid.UUID, characterId uint32) error

	// Enhanced transactional operations
	AcceptProposalWithTransactionAndEmit(transactionId uuid.UUID, proposalId uint32) (Marriage, error)

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
	CancelCeremonyAndEmit(transactionId uuid.UUID, ceremonyId uint32, cancelledBy uint32, reason string) (Ceremony, error)
	PostponeCeremony(ceremonyId uint32) model.Provider[Ceremony]
	PostponeCeremonyAndEmit(transactionId uuid.UUID, ceremonyId uint32, reason string) (Ceremony, error)
	RescheduleCeremony(ceremonyId uint32, newScheduledAt time.Time) model.Provider[Ceremony]
	RescheduleCeremonyAndEmit(transactionId uuid.UUID, ceremonyId uint32, newScheduledAt time.Time, rescheduledBy uint32) (Ceremony, error)

	// Ceremony invitee management
	AddInvitee(ceremonyId uint32, characterId uint32) model.Provider[Ceremony]
	AddInviteeAndEmit(transactionId uuid.UUID, ceremonyId uint32, characterId uint32, addedBy uint32) (Ceremony, error)
	RemoveInvitee(ceremonyId uint32, characterId uint32) model.Provider[Ceremony]
	RemoveInviteeAndEmit(transactionId uuid.UUID, ceremonyId uint32, characterId uint32, removedBy uint32) (Ceremony, error)

	// Ceremony state management
	AdvanceCeremonyState(ceremonyId uint32, nextState string) model.Provider[Ceremony]
	AdvanceCeremonyStateAndEmit(transactionId uuid.UUID, ceremonyId uint32, nextState string) (Ceremony, error)

	// Marriage queries
	GetMarriageByCharacter(characterId uint32) model.Provider[*Marriage]
	GetMarriageHistory(characterId uint32) model.Provider[[]Marriage]

	// Ceremony queries
	GetCeremonyById(ceremonyId uint32) model.Provider[*Ceremony]
	GetCeremonyByMarriage(marriageId uint32) model.Provider[*Ceremony]
	GetUpcomingCeremonies() model.Provider[[]Ceremony]
	GetActiveCeremonies() model.Provider[[]Ceremony]

	// Proposal expiry operations
	ExpireProposal(proposalId uint32) model.Provider[Proposal]
	ExpireProposalAndEmit(transactionId uuid.UUID, proposalId uint32) (Proposal, error)
	ProcessExpiredProposals() error

	// Ceremony timeout operations
	ProcessCeremonyTimeouts() error
}

// ProcessorImpl implements the Processor interface
type ProcessorImpl struct {
	log                logrus.FieldLogger
	ctx                context.Context
	db                 *gorm.DB
	producer           producer.Provider
	characterProcessor character.Processor
}

type ProcessorProducer func(log logrus.FieldLogger, ctx context.Context, db *gorm.DB) Processor

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

func (p *ProcessorImpl) WithProducer(producer producer.Provider) Processor {
	return &ProcessorImpl{
		log:                p.log,
		ctx:                p.ctx,
		db:                 p.db,
		producer:           producer,
		characterProcessor: p.characterProcessor,
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
		return buf.Put(marriageMsg.EnvEventTopicStatus, eventProvider)
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

// AcceptProposal accepts a proposal and creates a marriage
func (p *ProcessorImpl) AcceptProposal(proposalId uint32) model.Provider[Marriage] {
	return func() (Marriage, error) {
		p.log.WithField("proposalId", proposalId).Debug("Accepting proposal")

		// Get tenant from context
		t := tenant.MustFromContext(p.ctx)

		// Get the proposal
		proposalProvider := GetProposalByIdProvider(p.db, p.log)(proposalId, t.Id())
		proposal, err := proposalProvider()
		if err != nil {
			return Marriage{}, err
		}

		// Check if proposal can be accepted
		if !proposal.CanRespond() {
			return Marriage{}, errors.New("proposal cannot be accepted")
		}

		// Accept the proposal
		acceptedProposal, err := proposal.Accept()
		if err != nil {
			return Marriage{}, err
		}

		// Update the proposal in the database
		updateProposalProvider := UpdateProposal(p.db, p.log)(acceptedProposal)
		_, err = updateProposalProvider()
		if err != nil {
			return Marriage{}, err
		}

		// Create the marriage
		marriageProvider := CreateMarriage(p.db, p.log)(proposal.ProposerId(), proposal.TargetId(), t.Id())
		marriageEntity, err := marriageProvider()
		if err != nil {
			return Marriage{}, err
		}

		// Transform entity to domain model
		marriage, err := Make(marriageEntity)
		if err != nil {
			return Marriage{}, err
		}

		// Accept the marriage to set it to engaged status
		engagedMarriage, err := marriage.Accept()
		if err != nil {
			return Marriage{}, err
		}

		// Update the marriage in the database
		updateMarriageProvider := UpdateMarriage(p.db, p.log)(engagedMarriage)
		updatedEntity, err := updateMarriageProvider()
		if err != nil {
			return Marriage{}, err
		}

		// Transform entity to domain model
		result, err := Make(updatedEntity)
		if err != nil {
			return Marriage{}, err
		}

		p.log.WithFields(logrus.Fields{
			"proposalId": proposalId,
			"marriageId": result.Id(),
		}).Info("Proposal accepted and marriage created")

		return result, nil
	}
}

// AcceptProposalAndEmit accepts a proposal and emits events
func (p *ProcessorImpl) AcceptProposalAndEmit(transactionId uuid.UUID, proposalId uint32) (Marriage, error) {
	marriage, err := p.AcceptProposal(proposalId)()
	if err != nil {
		return Marriage{}, err
	}

	// Emit both ProposalAccepted and MarriageCreated events in a single transaction
	err = message.Emit(p.producer)(func(buf *message.Buffer) error {
		// Add ProposalAccepted event to buffer
		acceptedAt := time.Now()
		proposalAcceptedProvider := ProposalAcceptedEventProvider(
			proposalId,
			marriage.CharacterId1(),
			marriage.CharacterId2(),
			acceptedAt,
		)
		if err := buf.Put(marriageMsg.EnvEventTopicStatus, proposalAcceptedProvider); err != nil {
			return err
		}

		// Add MarriageCreated event to buffer
		marriedAt := time.Now()
		if marriage.EngagedAt() != nil {
			marriedAt = *marriage.EngagedAt()
		}
		marriageCreatedProvider := MarriageCreatedEventProvider(
			marriage.Id(),
			marriage.CharacterId1(),
			marriage.CharacterId2(),
			marriedAt,
		)
		return buf.Put(marriageMsg.EnvEventTopicStatus, marriageCreatedProvider)
	})
	if err != nil {
		return Marriage{}, err
	}

	p.log.WithFields(logrus.Fields{
		"transactionId": transactionId,
		"proposalId":    proposalId,
		"marriageId":    marriage.Id(),
	}).Debug("ProposalAccepted and MarriageCreated events emitted")

	return marriage, nil
}

// DeclineProposal declines a proposal and updates cooldown
func (p *ProcessorImpl) DeclineProposal(proposalId uint32) model.Provider[Proposal] {
	return func() (Proposal, error) {
		p.log.WithField("proposalId", proposalId).Debug("Declining proposal")

		// Get tenant from context
		t := tenant.MustFromContext(p.ctx)

		// Get the proposal
		proposalProvider := GetProposalByIdProvider(p.db, p.log)(proposalId, t.Id())
		proposal, err := proposalProvider()
		if err != nil {
			return Proposal{}, err
		}

		// Check if proposal can be declined
		if !proposal.CanRespond() {
			return Proposal{}, errors.New("proposal cannot be declined")
		}

		// Decline the proposal
		declinedProposal, err := proposal.Reject()
		if err != nil {
			return Proposal{}, err
		}

		// Update the proposal in the database
		updateProposalProvider := UpdateProposal(p.db, p.log)(declinedProposal)
		_, err = updateProposalProvider()
		if err != nil {
			return Proposal{}, err
		}

		p.log.WithFields(logrus.Fields{
			"proposalId":     proposalId,
			"rejectionCount": declinedProposal.RejectionCount(),
		}).Info("Proposal declined")

		return declinedProposal, nil
	}
}

// DeclineProposalAndEmit declines a proposal and emits events
func (p *ProcessorImpl) DeclineProposalAndEmit(transactionId uuid.UUID, proposalId uint32) (Proposal, error) {
	proposal, err := p.DeclineProposal(proposalId)()
	if err != nil {
		return Proposal{}, err
	}

	// Emit ProposalDeclined event
	err = message.Emit(p.producer)(func(buf *message.Buffer) error {
		declinedAt := time.Now()
		if proposal.RespondedAt() != nil {
			declinedAt = *proposal.RespondedAt()
		}
		cooldownUntil := time.Now()
		if proposal.CooldownUntil() != nil {
			cooldownUntil = *proposal.CooldownUntil()
		}
		eventProvider := ProposalDeclinedEventProvider(
			proposalId,
			proposal.ProposerId(),
			proposal.TargetId(),
			declinedAt,
			proposal.RejectionCount(),
			cooldownUntil,
		)
		return buf.Put(marriageMsg.EnvEventTopicStatus, eventProvider)
	})
	if err != nil {
		return Proposal{}, err
	}

	p.log.WithFields(logrus.Fields{
		"transactionId": transactionId,
		"proposalId":    proposalId,
	}).Debug("ProposalDeclined event emitted")

	return proposal, nil
}

// CancelProposal cancels a proposal by the proposer
func (p *ProcessorImpl) CancelProposal(proposalId uint32) model.Provider[Proposal] {
	return func() (Proposal, error) {
		p.log.WithField("proposalId", proposalId).Debug("Cancelling proposal")

		// Get tenant from context
		t := tenant.MustFromContext(p.ctx)

		// Get the proposal
		proposalProvider := GetProposalByIdProvider(p.db, p.log)(proposalId, t.Id())
		proposal, err := proposalProvider()
		if err != nil {
			return Proposal{}, err
		}

		// Check if proposal can be cancelled
		if !proposal.CanCancel() {
			return Proposal{}, errors.New("proposal cannot be cancelled")
		}

		// Cancel the proposal
		cancelledProposal, err := proposal.Cancel()
		if err != nil {
			return Proposal{}, err
		}

		// Update the proposal in the database
		updateProposalProvider := UpdateProposal(p.db, p.log)(cancelledProposal)
		_, err = updateProposalProvider()
		if err != nil {
			return Proposal{}, err
		}

		p.log.WithField("proposalId", proposalId).Info("Proposal cancelled")

		return cancelledProposal, nil
	}
}

// CancelProposalAndEmit cancels a proposal and emits events
func (p *ProcessorImpl) CancelProposalAndEmit(transactionId uuid.UUID, proposalId uint32) (Proposal, error) {
	proposal, err := p.CancelProposal(proposalId)()
	if err != nil {
		return Proposal{}, err
	}

	// Emit ProposalCancelled event
	err = message.Emit(p.producer)(func(buf *message.Buffer) error {
		cancelledAt := time.Now()
		eventProvider := ProposalCancelledEventProvider(
			proposalId,
			proposal.ProposerId(),
			proposal.TargetId(),
			cancelledAt,
		)
		return buf.Put(marriageMsg.EnvEventTopicStatus, eventProvider)
	})
	if err != nil {
		return Proposal{}, err
	}

	p.log.WithFields(logrus.Fields{
		"transactionId": transactionId,
		"proposalId":    proposalId,
	}).Debug("ProposalCancelled event emitted")

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

	// Use enhanced message buffering for potential future expansion
	err = message.Emit(p.producer)(func(buf *message.Buffer) error {
		// Primary ceremony scheduled event
		eventProvider := CeremonyScheduledEventProvider(
			ceremony.Id(),
			marriageId,
			ceremony.CharacterId1(),
			ceremony.CharacterId2(),
			scheduledAt,
			invitees,
		)
		if err := buf.Put(marriageMsg.EnvEventTopicStatus, eventProvider); err != nil {
			return err
		}

		// Potential for additional events (invitation notifications, etc.)
		// This pattern ensures all related events are emitted together
		return nil
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
		return buf.Put(marriageMsg.EnvEventTopicStatus, eventProvider)
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
		return buf.Put(marriageMsg.EnvEventTopicStatus, eventProvider)
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
func (p *ProcessorImpl) CancelCeremonyAndEmit(transactionId uuid.UUID, ceremonyId uint32, cancelledBy uint32, reason string) (Ceremony, error) {
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
			cancelledBy,
			reason,
		)
		return buf.Put(marriageMsg.EnvEventTopicStatus, eventProvider)
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
func (p *ProcessorImpl) PostponeCeremonyAndEmit(transactionId uuid.UUID, ceremonyId uint32, reason string) (Ceremony, error) {
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
			reason,
		)
		return buf.Put(marriageMsg.EnvEventTopicStatus, eventProvider)
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
			"ceremonyId":     ceremonyId,
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
func (p *ProcessorImpl) RescheduleCeremonyAndEmit(transactionId uuid.UUID, ceremonyId uint32, newScheduledAt time.Time, rescheduledBy uint32) (Ceremony, error) {
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
			rescheduledBy,
		)
		return buf.Put(marriageMsg.EnvEventTopicStatus, eventProvider)
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
func (p *ProcessorImpl) AddInviteeAndEmit(transactionId uuid.UUID, ceremonyId uint32, characterId uint32, addedBy uint32) (Ceremony, error) {
	ceremony, err := p.AddInvitee(ceremonyId, characterId)()
	if err != nil {
		return Ceremony{}, err
	}

	// Emit InviteeAdded event
	return ceremony, message.Emit(p.producer)(func(mb *message.Buffer) error {
		now := time.Now()
		inviteeAddedProvider := InviteeAddedEventProvider(
			ceremony.Id(),
			ceremony.MarriageId(),
			ceremony.CharacterId1(),
			ceremony.CharacterId2(),
			characterId,
			now,
			addedBy,
		)

		return mb.Put(marriageMsg.EnvEventTopicStatus, inviteeAddedProvider)
	})
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
func (p *ProcessorImpl) RemoveInviteeAndEmit(transactionId uuid.UUID, ceremonyId uint32, characterId uint32, removedBy uint32) (Ceremony, error) {
	ceremony, err := p.RemoveInvitee(ceremonyId, characterId)()
	if err != nil {
		return Ceremony{}, err
	}

	// Emit InviteeRemoved event
	return ceremony, message.Emit(p.producer)(func(mb *message.Buffer) error {
		now := time.Now()
		inviteeRemovedProvider := InviteeRemovedEventProvider(
			ceremony.Id(),
			ceremony.MarriageId(),
			ceremony.CharacterId1(),
			ceremony.CharacterId2(),
			characterId,
			now,
			removedBy,
		)

		return mb.Put(marriageMsg.EnvEventTopicStatus, inviteeRemovedProvider)
	})
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

// Divorce divorces a marriage
func (p *ProcessorImpl) Divorce(marriageId uint32, initiatedBy uint32) model.Provider[Marriage] {
	return func() (Marriage, error) {
		p.log.WithFields(logrus.Fields{
			"marriageId":  marriageId,
			"initiatedBy": initiatedBy,
		}).Debug("Processing divorce")

		// Get tenant from context
		t := tenant.MustFromContext(p.ctx)

		// Get the marriage
		marriageProvider := GetMarriageByIdProvider(p.db, p.log)(marriageId, t.Id())
		marriage, err := marriageProvider()
		if err != nil {
			return Marriage{}, err
		}
		if marriage == nil {
			return Marriage{}, errors.New("marriage not found")
		}

		// Check if marriage can be divorced
		if !marriage.CanDivorce() {
			return Marriage{}, errors.New("marriage cannot be divorced")
		}

		// Verify that the initiatedBy character is one of the partners
		if !marriage.IsPartner(initiatedBy) {
			return Marriage{}, errors.New("only married partners can initiate divorce")
		}

		// Divorce the marriage
		divorcedMarriage, err := marriage.Divorce()
		if err != nil {
			return Marriage{}, err
		}

		// Update the marriage in the database
		updateMarriageProvider := UpdateMarriage(p.db, p.log)(divorcedMarriage)
		updatedEntity, err := updateMarriageProvider()
		if err != nil {
			return Marriage{}, err
		}

		// Transform entity to domain model
		result, err := Make(updatedEntity)
		if err != nil {
			return Marriage{}, err
		}

		p.log.WithFields(logrus.Fields{
			"marriageId":  marriageId,
			"initiatedBy": initiatedBy,
		}).Info("Marriage divorced successfully")

		return result, nil
	}
}

// DivorceAndEmit divorces a marriage and emits events
func (p *ProcessorImpl) DivorceAndEmit(transactionId uuid.UUID, marriageId uint32, initiatedBy uint32) (Marriage, error) {
	marriage, err := p.Divorce(marriageId, initiatedBy)()
	if err != nil {
		return Marriage{}, err
	}

	// Emit MarriageDivorced event
	err = message.Emit(p.producer)(func(buf *message.Buffer) error {
		divorcedAt := time.Now()
		if marriage.DivorcedAt() != nil {
			divorcedAt = *marriage.DivorcedAt()
		}
		eventProvider := MarriageDivorcedEventProvider(
			marriageId,
			marriage.CharacterId1(),
			marriage.CharacterId2(),
			divorcedAt,
			initiatedBy,
		)
		return buf.Put(marriageMsg.EnvEventTopicStatus, eventProvider)
	})
	if err != nil {
		return Marriage{}, err
	}

	p.log.WithFields(logrus.Fields{
		"transactionId": transactionId,
		"marriageId":    marriageId,
		"initiatedBy":   initiatedBy,
	}).Debug("MarriageDivorced event emitted")

	return marriage, nil
}

// AdvanceCeremonyState advances a ceremony to the next state
func (p *ProcessorImpl) AdvanceCeremonyState(ceremonyId uint32, nextState string) model.Provider[Ceremony] {
	return func() (Ceremony, error) {
		p.log.WithFields(logrus.Fields{
			"ceremonyId": ceremonyId,
			"nextState":  nextState,
		}).Debug("Advancing ceremony state")

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

		// Apply state transition based on nextState
		var updatedCeremony Ceremony
		switch nextState {
		case "active":
			if !ceremony.CanStart() {
				return Ceremony{}, errors.New("ceremony cannot be started")
			}
			updatedCeremony, err = ceremony.Start()
		case "completed":
			if !ceremony.CanComplete() {
				return Ceremony{}, errors.New("ceremony cannot be completed")
			}
			updatedCeremony, err = ceremony.Complete()
		case "cancelled":
			if !ceremony.CanCancel() {
				return Ceremony{}, errors.New("ceremony cannot be cancelled")
			}
			updatedCeremony, err = ceremony.Cancel()
		case "postponed":
			if !ceremony.CanPostpone() {
				return Ceremony{}, errors.New("ceremony cannot be postponed")
			}
			updatedCeremony, err = ceremony.Postpone()
		default:
			return Ceremony{}, errors.New("invalid ceremony state: " + nextState)
		}

		if err != nil {
			return Ceremony{}, err
		}

		// Update ceremony in database
		updateProvider := UpdateCeremony(p.db, p.log)(ceremonyId, updatedCeremony.ToEntity(), t.Id())
		entity, err := updateProvider()
		if err != nil {
			return Ceremony{}, err
		}

		// Transform entity to domain model
		result, err := MakeCeremony(entity)
		if err != nil {
			return Ceremony{}, err
		}

		p.log.WithFields(logrus.Fields{
			"ceremonyId": ceremonyId,
			"fromState":  ceremony.Status().String(),
			"toState":    result.Status().String(),
		}).Info("Ceremony state advanced successfully")

		return result, nil
	}
}

// AdvanceCeremonyStateAndEmit advances a ceremony state and emits appropriate events
func (p *ProcessorImpl) AdvanceCeremonyStateAndEmit(transactionId uuid.UUID, ceremonyId uint32, nextState string) (Ceremony, error) {
	ceremony, err := p.AdvanceCeremonyState(ceremonyId, nextState)()
	if err != nil {
		return Ceremony{}, err
	}

	// Emit appropriate event based on the new state
	err = message.Emit(p.producer)(func(buf *message.Buffer) error {
		switch nextState {
		case "active":
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
			return buf.Put(marriageMsg.EnvEventTopicStatus, eventProvider)
		case "completed":
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
			return buf.Put(marriageMsg.EnvEventTopicStatus, eventProvider)
		case "cancelled":
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
				0,                    // No specific character ID for state transitions
				"ceremony_cancelled", // Default reason for state transitions
			)
			return buf.Put(marriageMsg.EnvEventTopicStatus, eventProvider)
		case "postponed":
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
				"ceremony_postponed", // Default reason for state transitions
			)
			return buf.Put(marriageMsg.EnvEventTopicStatus, eventProvider)
		default:
			// No event to emit for unknown states
			return nil
		}
	})
	if err != nil {
		return Ceremony{}, err
	}

	p.log.WithFields(logrus.Fields{
		"transactionId": transactionId,
		"ceremonyId":    ceremonyId,
		"nextState":     nextState,
	}).Debug("Ceremony state advanced and event emitted")

	return ceremony, nil
}

// AcceptProposalWithTransactionAndEmit provides full transactional consistency for proposal acceptance
// This method demonstrates enhanced message buffering for complex operations involving multiple database
// changes and event emissions that must all succeed or fail together
func (p *ProcessorImpl) AcceptProposalWithTransactionAndEmit(transactionId uuid.UUID, proposalId uint32) (Marriage, error) {
	// Execute the entire operation within a database transaction
	return p.executeInTransaction(func(txProcessor *ProcessorImpl) (Marriage, error) {
		// Get tenant from context
		t := tenant.MustFromContext(p.ctx)

		// Get the proposal
		proposalProvider := GetProposalByIdProvider(txProcessor.db, txProcessor.log)(proposalId, t.Id())
		proposal, err := proposalProvider()
		if err != nil {
			return Marriage{}, err
		}

		// Check if proposal can be accepted
		if !proposal.CanRespond() {
			return Marriage{}, errors.New("proposal cannot be accepted")
		}

		// Accept the proposal
		acceptedProposal, err := proposal.Accept()
		if err != nil {
			return Marriage{}, err
		}

		// Create marriage and update proposal within the transaction buffer
		return message.EmitWithResult[Marriage, Proposal](txProcessor.producer)(func(buf *message.Buffer) func(Proposal) (Marriage, error) {
			return func(acceptedProposal Proposal) (Marriage, error) {
				// Update the proposal in the database
				updateProposalProvider := UpdateProposal(txProcessor.db, txProcessor.log)(acceptedProposal)
				_, err := updateProposalProvider()
				if err != nil {
					return Marriage{}, err
				}

				// Create the marriage
				marriageProvider := CreateMarriage(txProcessor.db, txProcessor.log)(proposal.ProposerId(), proposal.TargetId(), t.Id())
				marriageEntity, err := marriageProvider()
				if err != nil {
					return Marriage{}, err
				}

				// Transform entity to domain model
				marriage, err := Make(marriageEntity)
				if err != nil {
					return Marriage{}, err
				}

				// Accept the marriage to set it to engaged status
				engagedMarriage, err := marriage.Accept()
				if err != nil {
					return Marriage{}, err
				}

				// Update the marriage in the database
				updateMarriageProvider := UpdateMarriage(txProcessor.db, txProcessor.log)(engagedMarriage)
				updatedEntity, err := updateMarriageProvider()
				if err != nil {
					return Marriage{}, err
				}

				// Transform entity to domain model
				result, err := Make(updatedEntity)
				if err != nil {
					return Marriage{}, err
				}

				// Buffer ProposalAccepted event
				acceptedAt := time.Now()
				proposalAcceptedProvider := ProposalAcceptedEventProvider(
					proposalId,
					result.CharacterId1(),
					result.CharacterId2(),
					acceptedAt,
				)
				if err := buf.Put(marriageMsg.EnvEventTopicStatus, proposalAcceptedProvider); err != nil {
					return Marriage{}, err
				}

				// Buffer MarriageCreated event
				marriedAt := time.Now()
				if result.EngagedAt() != nil {
					marriedAt = *result.EngagedAt()
				}
				marriageCreatedProvider := MarriageCreatedEventProvider(
					result.Id(),
					result.CharacterId1(),
					result.CharacterId2(),
					marriedAt,
				)
				if err := buf.Put(marriageMsg.EnvEventTopicStatus, marriageCreatedProvider); err != nil {
					return Marriage{}, err
				}

				p.log.WithFields(logrus.Fields{
					"transactionId": transactionId,
					"proposalId":    proposalId,
					"marriageId":    result.Id(),
				}).Info("Proposal accepted and marriage created with full transactional consistency")

				return result, nil
			}
		})(acceptedProposal)
	})
}

// executeInTransaction wraps business logic in a database transaction
// This ensures both database operations and message emission are transactionally consistent
func (p *ProcessorImpl) executeInTransaction(operation func(*ProcessorImpl) (Marriage, error)) (Marriage, error) {
	// Begin database transaction
	tx := p.db.Begin()
	if tx.Error != nil {
		return Marriage{}, tx.Error
	}

	// Create a new processor with the transaction DB
	txProcessor := &ProcessorImpl{
		log:                p.log,
		ctx:                p.ctx,
		db:                 tx,
		producer:           p.producer,
		characterProcessor: p.characterProcessor,
	}

	// Execute the operation
	result, err := operation(txProcessor)
	if err != nil {
		// Rollback on error
		tx.Rollback()
		return Marriage{}, err
	}

	// Commit the transaction
	if err := tx.Commit().Error; err != nil {
		return Marriage{}, err
	}

	return result, nil
}

// GetMarriageByCharacter retrieves the active marriage for a character
func (p *ProcessorImpl) GetMarriageByCharacter(characterId uint32) model.Provider[*Marriage] {
	return func() (*Marriage, error) {
		t := tenant.MustFromContext(p.ctx)

		marriageProvider := GetActiveMarriageByCharacterProvider(p.db, p.log)(characterId, t.Id())
		return marriageProvider()
	}
}

// GetMarriageHistory retrieves marriage history for a character
func (p *ProcessorImpl) GetMarriageHistory(characterId uint32) model.Provider[[]Marriage] {
	return func() ([]Marriage, error) {
		t := tenant.MustFromContext(p.ctx)

		historyProvider := GetMarriageHistoryByCharacterProvider(p.db, p.log)(characterId, t.Id())
		return historyProvider()
	}
}

// ExpireProposal marks a proposal as expired
func (p *ProcessorImpl) ExpireProposal(proposalId uint32) model.Provider[Proposal] {
	return func() (Proposal, error) {
		p.log.WithField("proposalId", proposalId).Debug("Expiring proposal")

		// Get tenant from context
		t := tenant.MustFromContext(p.ctx)

		// Get the proposal
		proposalProvider := GetProposalByIdProvider(p.db, p.log)(proposalId, t.Id())
		proposal, err := proposalProvider()
		if err != nil {
			return Proposal{}, err
		}

		// Check if proposal can be expired
		if proposal.Status() != ProposalStatusPending {
			return Proposal{}, errors.New("only pending proposals can be expired")
		}

		// Expire the proposal
		expiredProposal, err := proposal.Expire()
		if err != nil {
			return Proposal{}, err
		}

		// Update the proposal in the database
		updateProposalProvider := UpdateProposal(p.db, p.log)(expiredProposal)
		_, err = updateProposalProvider()
		if err != nil {
			return Proposal{}, err
		}

		p.log.WithField("proposalId", proposalId).Info("Proposal expired successfully")

		return expiredProposal, nil
	}
}

// ExpireProposalAndEmit expires a proposal and emits events
func (p *ProcessorImpl) ExpireProposalAndEmit(transactionId uuid.UUID, proposalId uint32) (Proposal, error) {
	proposal, err := p.ExpireProposal(proposalId)()
	if err != nil {
		return Proposal{}, err
	}

	// Emit ProposalExpired event
	err = message.Emit(p.producer)(func(buf *message.Buffer) error {
		expiredAt := time.Now()
		eventProvider := ProposalExpiredEventProvider(
			proposalId,
			proposal.ProposerId(),
			proposal.TargetId(),
			expiredAt,
		)
		return buf.Put(marriageMsg.EnvEventTopicStatus, eventProvider)
	})
	if err != nil {
		return Proposal{}, err
	}

	p.log.WithFields(logrus.Fields{
		"transactionId": transactionId,
		"proposalId":    proposalId,
	}).Debug("ProposalExpired event emitted")

	return proposal, nil
}

// ProcessExpiredProposals processes all expired proposals for all tenants
func (p *ProcessorImpl) ProcessExpiredProposals() error {
	p.log.Debug("Processing expired proposals")

	// Get tenant from context
	t := tenant.MustFromContext(p.ctx)

	// Get all expired proposals for this tenant
	expiredProposalsProvider := GetExpiredProposalsProvider(p.db, p.log)(t.Id())
	expiredProposals, err := expiredProposalsProvider()
	if err != nil {
		p.log.WithError(err).Error("Failed to retrieve expired proposals")
		return err
	}

	if len(expiredProposals) == 0 {
		p.log.Debug("No expired proposals found")
		return nil
	}

	p.log.WithField("count", len(expiredProposals)).Info("Processing expired proposals")

	// Process each expired proposal
	for _, proposal := range expiredProposals {
		transactionId := uuid.New()
		_, err := p.ExpireProposalAndEmit(transactionId, proposal.Id())
		if err != nil {
			p.log.WithFields(logrus.Fields{
				"proposalId": proposal.Id(),
				"error":      err,
			}).Error("Failed to expire proposal")
			// Continue processing other proposals even if one fails
			continue
		}

		p.log.WithField("proposalId", proposal.Id()).Debug("Successfully expired proposal")
	}

	p.log.WithField("processedCount", len(expiredProposals)).Info("Completed processing expired proposals")
	return nil
}

// ProcessCeremonyTimeouts processes all active ceremonies that may have timed out
func (p *ProcessorImpl) ProcessCeremonyTimeouts() error {
	p.log.Debug("Processing ceremony timeouts")

	// Get tenant from context
	t := tenant.MustFromContext(p.ctx)

	// Get all ceremonies that may have timed out
	timeoutCeremoniesProvider := GetTimeoutCeremoniesProvider(p.db, p.log)(t.Id())
	timeoutCeremonies, err := timeoutCeremoniesProvider()
	if err != nil {
		p.log.WithError(err).Error("Failed to retrieve ceremonies that may have timed out")
		return err
	}

	if len(timeoutCeremonies) == 0 {
		p.log.Debug("No ceremonies found that may have timed out")
		return nil
	}

	p.log.WithField("count", len(timeoutCeremonies)).Info("Processing ceremony timeouts")

	// Process each ceremony that may have timed out
	for _, ceremony := range timeoutCeremonies {
		transactionId := uuid.New()
		_, err := p.PostponeCeremonyAndEmit(transactionId, ceremony.Id(), "timeout_disconnection")
		if err != nil {
			p.log.WithFields(logrus.Fields{
				"ceremonyId": ceremony.Id(),
				"error":      err,
			}).Error("Failed to postpone ceremony due to timeout")
			// Continue processing other ceremonies even if one fails
			continue
		}

		p.log.WithFields(logrus.Fields{
			"ceremonyId":   ceremony.Id(),
			"characterId1": ceremony.CharacterId1(),
			"characterId2": ceremony.CharacterId2(),
		}).Info("Successfully postponed ceremony due to timeout")
	}

	p.log.WithField("processedCount", len(timeoutCeremonies)).Info("Completed processing ceremony timeouts")
	return nil
}

// HandleCharacterDeletion handles automatic divorce when a character is deleted
func (p *ProcessorImpl) HandleCharacterDeletion(characterId uint32) error {
	p.log.WithField("characterId", characterId).Debug("Processing character deletion for marriage cleanup")

	// Get tenant from context
	t := tenant.MustFromContext(p.ctx)

	// Get any active marriage for this character
	marriageProvider := GetActiveMarriageByCharacterProvider(p.db, p.log)(characterId, t.Id())
	marriage, err := marriageProvider()
	if err != nil {
		p.log.WithError(err).WithField("characterId", characterId).Error("Failed to retrieve active marriage for character deletion")
		return err
	}

	// If no active marriage, nothing to do
	if marriage == nil {
		p.log.WithField("characterId", characterId).Debug("No active marriage found for deleted character")
		return nil
	}

	// Mark the marriage as deleted due to character deletion
	now := time.Now()
	builder := marriage.Builder().
		SetStatus(StatusDivorced). // Use divorced status as there's no separate deleted status
		SetDivorcedAt(&now).
		SetUpdatedAt(now)

	// If marriage is not yet married (e.g., still engaged), set a married timestamp
	// This is required by business rules for divorced status
	if marriage.Status() == StatusEngaged && marriage.MarriedAt() == nil {
		builder = builder.SetMarriedAt(&now) // Set to same time as divorce for deleted characters
	}

	deletedMarriage, err := builder.Build()

	if err != nil {
		p.log.WithError(err).WithField("characterId", characterId).Error("Failed to build deleted marriage")
		return err
	}

	// Update the marriage in the database
	updateMarriageProvider := UpdateMarriage(p.db, p.log)(deletedMarriage)
	_, err = updateMarriageProvider()
	if err != nil {
		p.log.WithError(err).WithFields(logrus.Fields{
			"marriageId":  marriage.Id(),
			"characterId": characterId,
		}).Error("Failed to update marriage for character deletion")
		return err
	}

	p.log.WithFields(logrus.Fields{
		"marriageId":  marriage.Id(),
		"characterId": characterId,
	}).Info("Marriage automatically ended due to character deletion")

	return nil
}

// HandleCharacterDeletionAndEmit handles character deletion and emits appropriate events
func (p *ProcessorImpl) HandleCharacterDeletionAndEmit(transactionId uuid.UUID, characterId uint32) error {
	p.log.WithFields(logrus.Fields{
		"characterId":   characterId,
		"transactionId": transactionId,
	}).Debug("Processing character deletion with event emission")

	// Get tenant from context
	t := tenant.MustFromContext(p.ctx)

	// Get any active marriage for this character
	marriageProvider := GetActiveMarriageByCharacterProvider(p.db, p.log)(characterId, t.Id())
	marriage, err := marriageProvider()
	if err != nil {
		p.log.WithError(err).WithField("characterId", characterId).Error("Failed to retrieve active marriage for character deletion")
		return err
	}

	// If no active marriage, nothing to do
	if marriage == nil {
		p.log.WithField("characterId", characterId).Debug("No active marriage found for deleted character")
		return nil
	}

	// Process the deletion with events
	err = message.Emit(p.producer)(func(buf *message.Buffer) error {
		// Mark the marriage as deleted due to character deletion
		now := time.Now()
		builder := marriage.Builder().
			SetStatus(StatusDivorced). // Use divorced status as there's no separate deleted status
			SetDivorcedAt(&now).
			SetUpdatedAt(now)

		// If marriage is not yet married (e.g., still engaged), set a married timestamp
		// This is required by business rules for divorced status
		if marriage.Status() == StatusEngaged && marriage.MarriedAt() == nil {
			builder = builder.SetMarriedAt(&now) // Set to same time as divorce for deleted characters
		}

		deletedMarriage, err := builder.Build()

		if err != nil {
			return err
		}

		// Update the marriage in the database
		updateMarriageProvider := UpdateMarriage(p.db, p.log)(deletedMarriage)
		_, err = updateMarriageProvider()
		if err != nil {
			return err
		}

		// Emit MarriageDeleted event for character deletion
		deletedAt := now
		eventProvider := MarriageDeletedEventProvider(
			marriage.Id(),
			marriage.CharacterId1(),
			marriage.CharacterId2(),
			deletedAt,
			characterId, // The deleted character initiated the deletion
			"character_deleted",
		)
		return buf.Put(marriageMsg.EnvEventTopicStatus, eventProvider)
	})
	if err != nil {
		p.log.WithError(err).WithFields(logrus.Fields{
			"marriageId":  marriage.Id(),
			"characterId": characterId,
		}).Error("Failed to process character deletion")
		return err
	}

	p.log.WithFields(logrus.Fields{
		"transactionId": transactionId,
		"marriageId":    marriage.Id(),
		"characterId":   characterId,
	}).Info("Character deletion processed successfully with events")

	return nil
}
