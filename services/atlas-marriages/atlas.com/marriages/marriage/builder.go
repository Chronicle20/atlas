package marriage

import (
	"errors"
	"time"

	"github.com/google/uuid"
)

// Builder provides fluent construction of Marriage models
type Builder struct {
	id           uint32
	characterId1 uint32
	characterId2 uint32
	status       MarriageStatus
	proposedAt   time.Time
	engagedAt    *time.Time
	marriedAt    *time.Time
	divorcedAt   *time.Time
	tenantId     uuid.UUID
	createdAt    time.Time
	updatedAt    time.Time
}

// NewBuilder creates a new builder with required parameters
func NewBuilder(characterId1, characterId2 uint32, tenantId uuid.UUID) *Builder {
	now := time.Now()
	return &Builder{
		characterId1: characterId1,
		characterId2: characterId2,
		status:       StatusProposed,
		proposedAt:   now,
		tenantId:     tenantId,
		createdAt:    now,
		updatedAt:    now,
	}
}

// SetId sets the marriage ID
func (b *Builder) SetId(id uint32) *Builder {
	b.id = id
	return b
}

// SetStatus sets the marriage status
func (b *Builder) SetStatus(status MarriageStatus) *Builder {
	b.status = status
	return b
}

// SetProposedAt sets the proposal timestamp
func (b *Builder) SetProposedAt(proposedAt time.Time) *Builder {
	b.proposedAt = proposedAt
	return b
}

// SetEngagedAt sets the engagement timestamp
func (b *Builder) SetEngagedAt(engagedAt *time.Time) *Builder {
	b.engagedAt = engagedAt
	return b
}

// SetMarriedAt sets the marriage timestamp
func (b *Builder) SetMarriedAt(marriedAt *time.Time) *Builder {
	b.marriedAt = marriedAt
	return b
}

// SetDivorcedAt sets the divorce timestamp
func (b *Builder) SetDivorcedAt(divorcedAt *time.Time) *Builder {
	b.divorcedAt = divorcedAt
	return b
}

// SetCreatedAt sets the creation timestamp
func (b *Builder) SetCreatedAt(createdAt time.Time) *Builder {
	b.createdAt = createdAt
	return b
}

// SetUpdatedAt sets the last update timestamp
func (b *Builder) SetUpdatedAt(updatedAt time.Time) *Builder {
	b.updatedAt = updatedAt
	return b
}

// Build validates and constructs the final Marriage model
func (b *Builder) Build() (Marriage, error) {
	if b.characterId1 == 0 {
		return Marriage{}, errors.New("character ID 1 is required")
	}
	
	if b.characterId2 == 0 {
		return Marriage{}, errors.New("character ID 2 is required")
	}
	
	if b.characterId1 == b.characterId2 {
		return Marriage{}, errors.New("character cannot marry themselves")
	}
	
	if b.tenantId == uuid.Nil {
		return Marriage{}, errors.New("tenant ID is required")
	}
	
	// Validate state transitions
	if err := b.validateStateTransitions(); err != nil {
		return Marriage{}, err
	}
	
	return Marriage{
		id:           b.id,
		characterId1: b.characterId1,
		characterId2: b.characterId2,
		status:       b.status,
		proposedAt:   b.proposedAt,
		engagedAt:    b.engagedAt,
		marriedAt:    b.marriedAt,
		divorcedAt:   b.divorcedAt,
		tenantId:     b.tenantId,
		createdAt:    b.createdAt,
		updatedAt:    b.updatedAt,
	}, nil
}

// validateStateTransitions validates the consistency of state transitions
func (b *Builder) validateStateTransitions() error {
	switch b.status {
	case StatusProposed:
		if b.engagedAt != nil {
			return errors.New("proposed marriage cannot have engagement timestamp")
		}
		if b.marriedAt != nil {
			return errors.New("proposed marriage cannot have marriage timestamp")
		}
		if b.divorcedAt != nil {
			return errors.New("proposed marriage cannot have divorce timestamp")
		}
	case StatusEngaged:
		if b.engagedAt == nil {
			return errors.New("engaged marriage must have engagement timestamp")
		}
		if b.marriedAt != nil {
			return errors.New("engaged marriage cannot have marriage timestamp")
		}
		if b.divorcedAt != nil {
			return errors.New("engaged marriage cannot have divorce timestamp")
		}
	case StatusMarried:
		if b.engagedAt == nil {
			return errors.New("married marriage must have engagement timestamp")
		}
		if b.marriedAt == nil {
			return errors.New("married marriage must have marriage timestamp")
		}
		if b.divorcedAt != nil {
			return errors.New("married marriage cannot have divorce timestamp")
		}
	case StatusDivorced:
		if b.engagedAt == nil {
			return errors.New("divorced marriage must have engagement timestamp")
		}
		if b.marriedAt == nil {
			return errors.New("divorced marriage must have marriage timestamp")
		}
		if b.divorcedAt == nil {
			return errors.New("divorced marriage must have divorce timestamp")
		}
	case StatusExpired:
		if b.engagedAt != nil {
			return errors.New("expired marriage cannot have engagement timestamp")
		}
		if b.marriedAt != nil {
			return errors.New("expired marriage cannot have marriage timestamp")
		}
		if b.divorcedAt != nil {
			return errors.New("expired marriage cannot have divorce timestamp")
		}
	default:
		return errors.New("invalid marriage status")
	}
	
	return nil
}

// ProposalBuilder provides fluent construction of Proposal models
type ProposalBuilder struct {
	id             uint32
	proposerId     uint32
	targetId       uint32
	status         ProposalStatus
	proposedAt     time.Time
	respondedAt    *time.Time
	expiresAt      time.Time
	rejectionCount uint32
	cooldownUntil  *time.Time
	tenantId       uuid.UUID
	createdAt      time.Time
	updatedAt      time.Time
}

// NewProposalBuilder creates a new builder with required parameters
func NewProposalBuilder(proposerId, targetId uint32, tenantId uuid.UUID) *ProposalBuilder {
	now := time.Now()
	return &ProposalBuilder{
		proposerId:     proposerId,
		targetId:       targetId,
		status:         ProposalStatusPending,
		proposedAt:     now,
		expiresAt:      now.Add(ProposalExpiryDuration),
		rejectionCount: 0,
		tenantId:       tenantId,
		createdAt:      now,
		updatedAt:      now,
	}
}

// SetId sets the proposal ID
func (b *ProposalBuilder) SetId(id uint32) *ProposalBuilder {
	b.id = id
	return b
}

// SetStatus sets the proposal status
func (b *ProposalBuilder) SetStatus(status ProposalStatus) *ProposalBuilder {
	b.status = status
	return b
}

// SetProposedAt sets the proposal timestamp
func (b *ProposalBuilder) SetProposedAt(proposedAt time.Time) *ProposalBuilder {
	b.proposedAt = proposedAt
	return b
}

// SetRespondedAt sets the response timestamp
func (b *ProposalBuilder) SetRespondedAt(respondedAt *time.Time) *ProposalBuilder {
	b.respondedAt = respondedAt
	return b
}

// SetExpiresAt sets the expiry timestamp
func (b *ProposalBuilder) SetExpiresAt(expiresAt time.Time) *ProposalBuilder {
	b.expiresAt = expiresAt
	return b
}

// SetRejectionCount sets the rejection count
func (b *ProposalBuilder) SetRejectionCount(rejectionCount uint32) *ProposalBuilder {
	b.rejectionCount = rejectionCount
	return b
}

// SetCooldownUntil sets the cooldown timestamp
func (b *ProposalBuilder) SetCooldownUntil(cooldownUntil *time.Time) *ProposalBuilder {
	b.cooldownUntil = cooldownUntil
	return b
}

// SetCreatedAt sets the creation timestamp
func (b *ProposalBuilder) SetCreatedAt(createdAt time.Time) *ProposalBuilder {
	b.createdAt = createdAt
	return b
}

// SetUpdatedAt sets the last update timestamp
func (b *ProposalBuilder) SetUpdatedAt(updatedAt time.Time) *ProposalBuilder {
	b.updatedAt = updatedAt
	return b
}

// Build validates and constructs the final Proposal model
func (b *ProposalBuilder) Build() (Proposal, error) {
	if b.proposerId == 0 {
		return Proposal{}, errors.New("proposer ID is required")
	}
	
	if b.targetId == 0 {
		return Proposal{}, errors.New("target ID is required")
	}
	
	if b.proposerId == b.targetId {
		return Proposal{}, errors.New("character cannot propose to themselves")
	}
	
	if b.tenantId == uuid.Nil {
		return Proposal{}, errors.New("tenant ID is required")
	}
	
	if b.expiresAt.Before(b.proposedAt) {
		return Proposal{}, errors.New("expiry time cannot be before proposal time")
	}
	
	// Validate state transitions
	if err := b.validateProposalStateTransitions(); err != nil {
		return Proposal{}, err
	}
	
	return Proposal{
		id:             b.id,
		proposerId:     b.proposerId,
		targetId:       b.targetId,
		status:         b.status,
		proposedAt:     b.proposedAt,
		respondedAt:    b.respondedAt,
		expiresAt:      b.expiresAt,
		rejectionCount: b.rejectionCount,
		cooldownUntil:  b.cooldownUntil,
		tenantId:       b.tenantId,
		createdAt:      b.createdAt,
		updatedAt:      b.updatedAt,
	}, nil
}

// validateProposalStateTransitions validates the consistency of proposal state transitions
func (b *ProposalBuilder) validateProposalStateTransitions() error {
	switch b.status {
	case ProposalStatusPending:
		if b.respondedAt != nil {
			return errors.New("pending proposal cannot have response timestamp")
		}
	case ProposalStatusAccepted:
		if b.respondedAt == nil {
			return errors.New("accepted proposal must have response timestamp")
		}
	case ProposalStatusRejected:
		if b.respondedAt == nil {
			return errors.New("rejected proposal must have response timestamp")
		}
		if b.cooldownUntil == nil {
			return errors.New("rejected proposal must have cooldown timestamp")
		}
	case ProposalStatusExpired:
		if b.respondedAt != nil {
			return errors.New("expired proposal cannot have response timestamp")
		}
	case ProposalStatusCancelled:
		if b.respondedAt != nil {
			return errors.New("cancelled proposal cannot have response timestamp")
		}
	default:
		return errors.New("invalid proposal status")
	}
	
	return nil
}

// CeremonyBuilder provides fluent construction of Ceremony models
type CeremonyBuilder struct {
	id           uint32
	marriageId   uint32
	characterId1 uint32
	characterId2 uint32
	status       CeremonyStatus
	scheduledAt  time.Time
	startedAt    *time.Time
	completedAt  *time.Time
	cancelledAt  *time.Time
	postponedAt  *time.Time
	invitees     []uint32
	tenantId     uuid.UUID
	createdAt    time.Time
	updatedAt    time.Time
}

// NewCeremonyBuilder creates a new builder with required parameters
func NewCeremonyBuilder(marriageId, characterId1, characterId2 uint32, tenantId uuid.UUID) *CeremonyBuilder {
	now := time.Now()
	return &CeremonyBuilder{
		marriageId:   marriageId,
		characterId1: characterId1,
		characterId2: characterId2,
		status:       CeremonyStatusScheduled,
		scheduledAt:  now,
		invitees:     make([]uint32, 0),
		tenantId:     tenantId,
		createdAt:    now,
		updatedAt:    now,
	}
}

// SetId sets the ceremony ID
func (b *CeremonyBuilder) SetId(id uint32) *CeremonyBuilder {
	b.id = id
	return b
}

// SetMarriageId sets the marriage ID
func (b *CeremonyBuilder) SetMarriageId(marriageId uint32) *CeremonyBuilder {
	b.marriageId = marriageId
	return b
}

// SetStatus sets the ceremony status
func (b *CeremonyBuilder) SetStatus(status CeremonyStatus) *CeremonyBuilder {
	b.status = status
	return b
}

// SetScheduledAt sets the scheduled timestamp
func (b *CeremonyBuilder) SetScheduledAt(scheduledAt time.Time) *CeremonyBuilder {
	b.scheduledAt = scheduledAt
	return b
}

// SetStartedAt sets the started timestamp
func (b *CeremonyBuilder) SetStartedAt(startedAt *time.Time) *CeremonyBuilder {
	b.startedAt = startedAt
	return b
}

// SetCompletedAt sets the completed timestamp
func (b *CeremonyBuilder) SetCompletedAt(completedAt *time.Time) *CeremonyBuilder {
	b.completedAt = completedAt
	return b
}

// SetCancelledAt sets the cancelled timestamp
func (b *CeremonyBuilder) SetCancelledAt(cancelledAt *time.Time) *CeremonyBuilder {
	b.cancelledAt = cancelledAt
	return b
}

// SetPostponedAt sets the postponed timestamp
func (b *CeremonyBuilder) SetPostponedAt(postponedAt *time.Time) *CeremonyBuilder {
	b.postponedAt = postponedAt
	return b
}

// SetInvitees sets the invitees list
func (b *CeremonyBuilder) SetInvitees(invitees []uint32) *CeremonyBuilder {
	// Copy the slice to maintain immutability
	b.invitees = make([]uint32, len(invitees))
	copy(b.invitees, invitees)
	return b
}

// SetCreatedAt sets the creation timestamp
func (b *CeremonyBuilder) SetCreatedAt(createdAt time.Time) *CeremonyBuilder {
	b.createdAt = createdAt
	return b
}

// SetUpdatedAt sets the last update timestamp
func (b *CeremonyBuilder) SetUpdatedAt(updatedAt time.Time) *CeremonyBuilder {
	b.updatedAt = updatedAt
	return b
}

// Build validates and constructs the final Ceremony model
func (b *CeremonyBuilder) Build() (Ceremony, error) {
	if b.marriageId == 0 {
		return Ceremony{}, errors.New("marriage ID is required")
	}
	
	if b.characterId1 == 0 {
		return Ceremony{}, errors.New("character ID 1 is required")
	}
	
	if b.characterId2 == 0 {
		return Ceremony{}, errors.New("character ID 2 is required")
	}
	
	if b.characterId1 == b.characterId2 {
		return Ceremony{}, errors.New("character cannot have ceremony with themselves")
	}
	
	if b.tenantId == uuid.Nil {
		return Ceremony{}, errors.New("tenant ID is required")
	}
	
	if len(b.invitees) > MaxInvitees {
		return Ceremony{}, errors.New("too many invitees")
	}
	
	// Validate that invitees don't include the partners themselves
	for _, invitee := range b.invitees {
		if invitee == b.characterId1 || invitee == b.characterId2 {
			return Ceremony{}, errors.New("partners cannot be invitees")
		}
	}
	
	// Check for duplicate invitees
	inviteeMap := make(map[uint32]bool)
	for _, invitee := range b.invitees {
		if inviteeMap[invitee] {
			return Ceremony{}, errors.New("duplicate invitee")
		}
		inviteeMap[invitee] = true
	}
	
	// Validate state transitions
	if err := b.validateCeremonyStateTransitions(); err != nil {
		return Ceremony{}, err
	}
	
	// Copy invitees to maintain immutability
	invitees := make([]uint32, len(b.invitees))
	copy(invitees, b.invitees)
	
	return Ceremony{
		id:           b.id,
		marriageId:   b.marriageId,
		characterId1: b.characterId1,
		characterId2: b.characterId2,
		status:       b.status,
		scheduledAt:  b.scheduledAt,
		startedAt:    b.startedAt,
		completedAt:  b.completedAt,
		cancelledAt:  b.cancelledAt,
		postponedAt:  b.postponedAt,
		invitees:     invitees,
		tenantId:     b.tenantId,
		createdAt:    b.createdAt,
		updatedAt:    b.updatedAt,
	}, nil
}

// validateCeremonyStateTransitions validates the consistency of ceremony state transitions
func (b *CeremonyBuilder) validateCeremonyStateTransitions() error {
	switch b.status {
	case CeremonyStatusScheduled:
		if b.startedAt != nil {
			return errors.New("scheduled ceremony cannot have started timestamp")
		}
		if b.completedAt != nil {
			return errors.New("scheduled ceremony cannot have completed timestamp")
		}
		if b.cancelledAt != nil {
			return errors.New("scheduled ceremony cannot have cancelled timestamp")
		}
		if b.postponedAt != nil {
			return errors.New("scheduled ceremony cannot have postponed timestamp")
		}
	case CeremonyStatusActive:
		if b.startedAt == nil {
			return errors.New("active ceremony must have started timestamp")
		}
		if b.completedAt != nil {
			return errors.New("active ceremony cannot have completed timestamp")
		}
		if b.cancelledAt != nil {
			return errors.New("active ceremony cannot have cancelled timestamp")
		}
	case CeremonyStatusCompleted:
		if b.startedAt == nil {
			return errors.New("completed ceremony must have started timestamp")
		}
		if b.completedAt == nil {
			return errors.New("completed ceremony must have completed timestamp")
		}
		if b.cancelledAt != nil {
			return errors.New("completed ceremony cannot have cancelled timestamp")
		}
	case CeremonyStatusCancelled:
		if b.cancelledAt == nil {
			return errors.New("cancelled ceremony must have cancelled timestamp")
		}
		if b.completedAt != nil {
			return errors.New("cancelled ceremony cannot have completed timestamp")
		}
	case CeremonyStatusPostponed:
		if b.postponedAt == nil {
			return errors.New("postponed ceremony must have postponed timestamp")
		}
		if b.completedAt != nil {
			return errors.New("postponed ceremony cannot have completed timestamp")
		}
		if b.cancelledAt != nil {
			return errors.New("postponed ceremony cannot have cancelled timestamp")
		}
	default:
		return errors.New("invalid ceremony status")
	}
	
	return nil
}