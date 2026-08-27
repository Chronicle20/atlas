package game

import (
	"errors"
	"time"

	"github.com/google/uuid"

	"github.com/Chronicle20/atlas/libs/atlas-constants/channel"
	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

// Builder provides a fluent API for constructing game.Model instances.
type Builder struct {
	tenant      tenant.Model
	characterId uint32
	worldId     world.Id
	channelId   channel.Id
	npcId       uint32
	rung        int
	status      Status
	lastThrow   Throw
	createdAt   time.Time
	updatedAt   time.Time
}

// NewBuilder creates a new Builder with required tenant, seeding createdAt.
func NewBuilder(t tenant.Model) *Builder {
	return &Builder{
		tenant:    t,
		createdAt: time.Now(),
	}
}

// CloneBuilder creates a new Builder initialized from an existing Model.
func CloneBuilder(m Model) *Builder {
	return &Builder{
		tenant:      m.Tenant(),
		characterId: m.CharacterId(),
		worldId:     m.WorldId(),
		channelId:   m.ChannelId(),
		npcId:       m.NpcId(),
		rung:        m.Rung(),
		status:      m.Status(),
		lastThrow:   m.LastThrow(),
		createdAt:   m.CreatedAt(),
		updatedAt:   m.UpdatedAt(),
	}
}

// SetCharacterId sets the character ID.
func (b *Builder) SetCharacterId(characterId uint32) *Builder {
	b.characterId = characterId
	return b
}

// SetWorldId sets the world ID.
func (b *Builder) SetWorldId(worldId world.Id) *Builder {
	b.worldId = worldId
	return b
}

// SetChannelId sets the channel ID.
func (b *Builder) SetChannelId(channelId channel.Id) *Builder {
	b.channelId = channelId
	return b
}

// SetNpcId sets the NPC ID.
func (b *Builder) SetNpcId(npcId uint32) *Builder {
	b.npcId = npcId
	return b
}

// SetRung sets the ladder rung.
func (b *Builder) SetRung(rung int) *Builder {
	b.rung = rung
	return b
}

// SetStatus sets the session status.
func (b *Builder) SetStatus(status Status) *Builder {
	b.status = status
	return b
}

// SetLastThrow sets the last recorded throw.
func (b *Builder) SetLastThrow(lastThrow Throw) *Builder {
	b.lastThrow = lastThrow
	return b
}

// SetCreatedAt sets the created-at timestamp.
func (b *Builder) SetCreatedAt(createdAt time.Time) *Builder {
	b.createdAt = createdAt
	return b
}

// SetUpdatedAt sets the updated-at timestamp.
func (b *Builder) SetUpdatedAt(updatedAt time.Time) *Builder {
	b.updatedAt = updatedAt
	return b
}

// Build validates and constructs the Model. Returns an error if validation fails.
func (b *Builder) Build() (Model, error) {
	if b.tenant.Id() == uuid.Nil {
		return Model{}, errors.New("tenant is required")
	}
	if b.characterId == 0 {
		return Model{}, errors.New("characterId is required")
	}
	b.updatedAt = time.Now()
	return Model{
		tenant:      b.tenant,
		characterId: b.characterId,
		worldId:     b.worldId,
		channelId:   b.channelId,
		npcId:       b.npcId,
		rung:        b.rung,
		status:      b.status,
		lastThrow:   b.lastThrow,
		createdAt:   b.createdAt,
		updatedAt:   b.updatedAt,
	}, nil
}

// MustBuild builds the model and panics if validation fails.
// Use this only when building from a known-valid source (e.g., cloning an existing model).
func (b *Builder) MustBuild() Model {
	m, err := b.Build()
	if err != nil {
		panic("MustBuild failed: " + err.Error())
	}
	return m
}

// Tenant returns the tenant from the builder.
func (b *Builder) Tenant() tenant.Model {
	return b.tenant
}

// CharacterId returns the characterId from the builder.
func (b *Builder) CharacterId() uint32 {
	return b.characterId
}

// WorldId returns the worldId from the builder.
func (b *Builder) WorldId() world.Id {
	return b.worldId
}

// ChannelId returns the channelId from the builder.
func (b *Builder) ChannelId() channel.Id {
	return b.channelId
}

// NpcId returns the npcId from the builder.
func (b *Builder) NpcId() uint32 {
	return b.npcId
}

// Rung returns the rung from the builder.
func (b *Builder) Rung() int {
	return b.rung
}

// Status returns the status from the builder.
func (b *Builder) Status() Status {
	return b.status
}

// LastThrow returns the lastThrow from the builder.
func (b *Builder) LastThrow() Throw {
	return b.lastThrow
}

// CreatedAt returns the createdAt from the builder.
func (b *Builder) CreatedAt() time.Time {
	return b.createdAt
}

// UpdatedAt returns the updatedAt from the builder.
func (b *Builder) UpdatedAt() time.Time {
	return b.updatedAt
}
