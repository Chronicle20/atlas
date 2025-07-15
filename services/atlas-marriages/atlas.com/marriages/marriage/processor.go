package marriage

import (
	"context"
	"errors"
	"time"

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

		// Create new proposal
		proposal, err := NewProposalBuilder(proposerId, targetId, t.Id()).Build()
		if err != nil {
			return Proposal{}, err
		}

		// Save to database
		entity := proposal.ToProposalEntity()
		if err := p.db.Create(&entity).Error; err != nil {
			return Proposal{}, err
		}

		// Update proposal with generated ID
		savedProposal, err := NewProposalBuilder(proposerId, targetId, t.Id()).
			SetId(entity.ID).
			SetCreatedAt(entity.CreatedAt).
			SetUpdatedAt(entity.UpdatedAt).
			Build()
		if err != nil {
			return Proposal{}, err
		}

		p.log.WithFields(logrus.Fields{
			"proposalId": savedProposal.Id(),
			"proposerId": proposerId,
			"targetId":   targetId,
		}).Info("Marriage proposal created successfully")

		return savedProposal, nil
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
		proposerMarriage, err := p.getActiveMarriage(proposerId)
		if err != nil {
			return false, err
		}
		if proposerMarriage != nil {
			return false, nil
		}

		// Check if target is already married or engaged
		targetMarriage, err := p.getActiveMarriage(targetId)
		if err != nil {
			return false, err
		}
		if targetMarriage != nil {
			return false, nil
		}

		// Check if target already has a pending proposal
		existingProposal, err := p.GetActiveProposal(proposerId, targetId)()
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
		
		var lastProposal ProposalEntity
		err := p.db.Where("proposer_id = ? AND tenant_id = ?", proposerId, t.Id()).
			Order("created_at DESC").
			First(&lastProposal).Error
		
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return true, nil // No previous proposals, cooldown doesn't apply
			}
			return false, err
		}

		// Check if global cooldown period has passed
		cooldownEnd := lastProposal.CreatedAt.Add(GlobalCooldownDuration)
		return time.Now().After(cooldownEnd), nil
	}
}

// CheckPerTargetCooldown checks if the proposer is in cooldown period for specific target
func (p *ProcessorImpl) CheckPerTargetCooldown(proposerId, targetId uint32) model.Provider[bool] {
	return func() (bool, error) {
		t := tenant.MustFromContext(p.ctx)
		
		var lastProposal ProposalEntity
		err := p.db.Where("proposer_id = ? AND target_id = ? AND tenant_id = ?", proposerId, targetId, t.Id()).
			Order("created_at DESC").
			First(&lastProposal).Error
		
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return true, nil // No previous proposals to this target
			}
			return false, err
		}

		// If last proposal was rejected, check cooldown
		if lastProposal.Status == ProposalStatusRejected && lastProposal.CooldownUntil != nil {
			return time.Now().After(*lastProposal.CooldownUntil), nil
		}

		// If last proposal expired, apply initial cooldown
		if lastProposal.Status == ProposalStatusExpired {
			cooldownEnd := lastProposal.UpdatedAt.Add(InitialPerTargetCooldown)
			return time.Now().After(cooldownEnd), nil
		}

		return true, nil
	}
}

// GetActiveProposal retrieves an active proposal between two characters
func (p *ProcessorImpl) GetActiveProposal(proposerId, targetId uint32) model.Provider[*Proposal] {
	return func() (*Proposal, error) {
		t := tenant.MustFromContext(p.ctx)
		
		var entity ProposalEntity
		err := p.db.Where("proposer_id = ? AND target_id = ? AND tenant_id = ? AND status = ?", 
			proposerId, targetId, t.Id(), ProposalStatusPending).
			First(&entity).Error
		
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return nil, nil
			}
			return nil, err
		}

		proposal, err := MakeProposal(entity)
		if err != nil {
			return nil, err
		}

		// Check if proposal has expired
		if proposal.IsExpired() {
			return nil, nil
		}

		return &proposal, nil
	}
}

// GetPendingProposalsByCharacter retrieves all pending proposals for a character (sent or received)
func (p *ProcessorImpl) GetPendingProposalsByCharacter(characterId uint32) model.Provider[[]Proposal] {
	return func() ([]Proposal, error) {
		t := tenant.MustFromContext(p.ctx)
		
		var entities []ProposalEntity
		err := p.db.Where("(proposer_id = ? OR target_id = ?) AND tenant_id = ? AND status = ?", 
			characterId, characterId, t.Id(), ProposalStatusPending).
			Order("created_at DESC").
			Find(&entities).Error
		
		if err != nil {
			return nil, err
		}

		proposals := make([]Proposal, 0, len(entities))
		for _, entity := range entities {
			proposal, err := MakeProposal(entity)
			if err != nil {
				return nil, err
			}
			
			// Only include non-expired proposals
			if !proposal.IsExpired() {
				proposals = append(proposals, proposal)
			}
		}

		return proposals, nil
	}
}

// GetProposalHistory retrieves the history of proposals between two characters
func (p *ProcessorImpl) GetProposalHistory(proposerId, targetId uint32) model.Provider[[]Proposal] {
	return func() ([]Proposal, error) {
		t := tenant.MustFromContext(p.ctx)
		
		var entities []ProposalEntity
		err := p.db.Where("proposer_id = ? AND target_id = ? AND tenant_id = ?", 
			proposerId, targetId, t.Id()).
			Order("created_at DESC").
			Find(&entities).Error
		
		if err != nil {
			return nil, err
		}

		proposals := make([]Proposal, 0, len(entities))
		for _, entity := range entities {
			proposal, err := MakeProposal(entity)
			if err != nil {
				return nil, err
			}
			proposals = append(proposals, proposal)
		}

		return proposals, nil
	}
}

// getActiveMarriage is a helper function to check if a character has an active marriage
func (p *ProcessorImpl) getActiveMarriage(characterId uint32) (*Marriage, error) {
	t := tenant.MustFromContext(p.ctx)
	
	var entity Entity
	err := p.db.Where("(character_id1 = ? OR character_id2 = ?) AND tenant_id = ? AND status IN (?)", 
		characterId, characterId, t.Id(), []MarriageStatus{StatusProposed, StatusEngaged, StatusMarried}).
		First(&entity).Error
	
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}

	marriage, err := Make(entity)
	if err != nil {
		return nil, err
	}

	return &marriage, nil
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