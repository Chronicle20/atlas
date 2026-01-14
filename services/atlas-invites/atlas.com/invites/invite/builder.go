package invite

import (
	"errors"
	"time"

	"github.com/Chronicle20/atlas-tenant"
)

// Builder provides a fluent interface for constructing invite Model instances
// with validation. Use NewBuilder() to create a new builder instance.
type Builder struct {
	tenant       tenant.Model
	id           uint32
	inviteType   string
	referenceId  uint32
	originatorId uint32
	targetId     uint32
	worldId      byte
	age          time.Time
}

// NewBuilder creates a new Builder with default values.
func NewBuilder() *Builder {
	return &Builder{
		age: time.Now(),
	}
}

// SetTenant sets the tenant for multi-tenancy isolation.
func (b *Builder) SetTenant(t tenant.Model) *Builder {
	b.tenant = t
	return b
}

// SetId sets the unique invite identifier.
func (b *Builder) SetId(id uint32) *Builder {
	b.id = id
	return b
}

// SetInviteType sets the type of invite (e.g., "BUDDY", "PARTY", "GUILD").
func (b *Builder) SetInviteType(inviteType string) *Builder {
	b.inviteType = inviteType
	return b
}

// SetReferenceId sets the reference ID linking to related entity.
func (b *Builder) SetReferenceId(referenceId uint32) *Builder {
	b.referenceId = referenceId
	return b
}

// SetOriginatorId sets the character ID of the invite sender.
func (b *Builder) SetOriginatorId(originatorId uint32) *Builder {
	b.originatorId = originatorId
	return b
}

// SetTargetId sets the character ID of the invite recipient.
func (b *Builder) SetTargetId(targetId uint32) *Builder {
	b.targetId = targetId
	return b
}

// SetWorldId sets the game world ID.
func (b *Builder) SetWorldId(worldId byte) *Builder {
	b.worldId = worldId
	return b
}

// SetAge sets the creation timestamp of the invite.
func (b *Builder) SetAge(age time.Time) *Builder {
	b.age = age
	return b
}

// Build validates the builder state and returns a Model instance.
// Returns an error if any required fields are missing or invalid.
func (b *Builder) Build() (Model, error) {
	if b.tenant.Id().String() == "00000000-0000-0000-0000-000000000000" {
		return Model{}, errors.New("tenant is required")
	}
	if b.id == 0 {
		return Model{}, errors.New("id is required")
	}
	if b.inviteType == "" {
		return Model{}, errors.New("inviteType is required")
	}
	if b.originatorId == 0 {
		return Model{}, errors.New("originatorId is required")
	}
	if b.targetId == 0 {
		return Model{}, errors.New("targetId is required")
	}

	return Model{
		tenant:       b.tenant,
		id:           b.id,
		inviteType:   b.inviteType,
		referenceId:  b.referenceId,
		originatorId: b.originatorId,
		targetId:     b.targetId,
		worldId:      b.worldId,
		age:          b.age,
	}, nil
}
