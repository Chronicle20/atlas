package marriage

import (
	"context"
	"errors"

	"github.com/Chronicle20/atlas-model/model"
	"github.com/Chronicle20/atlas-tenant"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

// EligibilityRequirement represents minimum level requirement for marriage
const EligibilityRequirement = 10

// Processor interface defines the proposal processing operations
type Processor interface {
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
}

// ProcessorImpl implements the Processor interface
type ProcessorImpl struct {
	log      logrus.FieldLogger
	ctx      context.Context
	db       *gorm.DB
	producer ProducerFunction
}

// ProducerFunction type for Kafka message production
type ProducerFunction func(token string) model.Provider[[]byte]

// NewProcessor creates a new processor instance
func NewProcessor(log logrus.FieldLogger, ctx context.Context, db *gorm.DB) Processor {
	return &ProcessorImpl{
		log:      log,
		ctx:      ctx,
		db:       db,
		producer: nil, // Will be implemented when Kafka producer is added
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

// ProposeAndEmit creates a proposal and emits events (placeholder for future Kafka integration)
func (p *ProcessorImpl) ProposeAndEmit(transactionId uuid.UUID, proposerId, targetId uint32) (Proposal, error) {
	proposal, err := p.Propose(proposerId, targetId)()
	if err != nil {
		return Proposal{}, err
	}

	// TODO: Emit ProposalCreated event when Kafka producer is implemented
	p.log.WithFields(logrus.Fields{
		"transactionId": transactionId,
		"proposalId":    proposal.Id(),
	}).Debug("Would emit ProposalCreated event")

	return proposal, nil
}

// CheckEligibility checks if a character meets the minimum level requirement
func (p *ProcessorImpl) CheckEligibility(characterId uint32) model.Provider[bool] {
	return func() (bool, error) {
		// TODO: In a real implementation, this would query the character service
		// For now, we'll assume all characters are eligible (level 10+)
		p.log.WithField("characterId", characterId).Debug("Checking character eligibility")
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