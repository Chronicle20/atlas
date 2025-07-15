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