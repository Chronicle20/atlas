package party

import (
	"errors"
	"github.com/google/uuid"
)

// Builder provides fluent construction of party models
type Builder struct {
	tenantId *uuid.UUID
	id       *uint32
	leaderId *uint32
	members  []uint32
}

// NewBuilder creates a new builder with required parameters
func NewBuilder(tenantId uuid.UUID, id uint32, leaderId uint32) *Builder {
	return &Builder{
		tenantId: &tenantId,
		id:       &id,
		leaderId: &leaderId,
		members:  make([]uint32, 0),
	}
}

// SetMembers sets the party members
func (b *Builder) SetMembers(members []uint32) *Builder {
	b.members = make([]uint32, len(members))
	copy(b.members, members)
	return b
}

// Build validates invariants and constructs the final immutable model
func (b *Builder) Build() (Model, error) {
	if b.tenantId == nil {
		return Model{}, errors.New("tenant ID is required")
	}
	if b.id == nil {
		return Model{}, errors.New("party ID is required")
	}
	if b.leaderId == nil {
		return Model{}, errors.New("leader ID is required")
	}
	if *b.id == 0 {
		return Model{}, errors.New("party ID must be greater than 0")
	}
	if *b.leaderId == 0 {
		return Model{}, errors.New("leader ID must be greater than 0")
	}

	// Validate that leader is in members list if members are provided
	if len(b.members) > 0 {
		leaderInMembers := false
		for _, member := range b.members {
			if member == *b.leaderId {
				leaderInMembers = true
				break
			}
		}
		if !leaderInMembers {
			return Model{}, errors.New("leader must be a member of the party")
		}
	}

	// Validate no duplicate members
	memberSet := make(map[uint32]bool)
	for _, member := range b.members {
		if member == 0 {
			return Model{}, errors.New("member ID must be greater than 0")
		}
		if memberSet[member] {
			return Model{}, errors.New("duplicate member ID found")
		}
		memberSet[member] = true
	}

	return Model{
		tenantId: *b.tenantId,
		id:       *b.id,
		leaderId: *b.leaderId,
		members:  b.members,
	}, nil
}

// Builder returns a builder initialized with the current model's values
func (m Model) Builder() *Builder {
	return &Builder{
		tenantId: &m.tenantId,
		id:       &m.id,
		leaderId: &m.leaderId,
		members:  append([]uint32{}, m.members...),
	}
}