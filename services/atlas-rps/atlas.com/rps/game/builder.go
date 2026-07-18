package game

import (
	"errors"
	"time"

	"github.com/google/uuid"

	"github.com/Chronicle20/atlas/libs/atlas-constants/channel"
	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

// ModelBuilder provides a fluent API for constructing game.Model instances.
type ModelBuilder struct {
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

// NewModelBuilder creates a new ModelBuilder with required tenant, seeding createdAt.
func NewModelBuilder(t tenant.Model) *ModelBuilder {
	return &ModelBuilder{
		tenant:    t,
		createdAt: time.Now(),
	}
}

// CloneModelBuilder creates a new ModelBuilder initialized from an existing Model.
func CloneModelBuilder(m Model) *ModelBuilder {
	return &ModelBuilder{
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
func (b *ModelBuilder) SetCharacterId(characterId uint32) *ModelBuilder {
	b.characterId = characterId
	return b
}

// SetWorldId sets the world ID.
func (b *ModelBuilder) SetWorldId(worldId world.Id) *ModelBuilder {
	b.worldId = worldId
	return b
}

// SetChannelId sets the channel ID.
func (b *ModelBuilder) SetChannelId(channelId channel.Id) *ModelBuilder {
	b.channelId = channelId
	return b
}

// SetNpcId sets the NPC ID.
func (b *ModelBuilder) SetNpcId(npcId uint32) *ModelBuilder {
	b.npcId = npcId
	return b
}

// SetRung sets the ladder rung.
func (b *ModelBuilder) SetRung(rung int) *ModelBuilder {
	b.rung = rung
	return b
}

// SetStatus sets the session status.
func (b *ModelBuilder) SetStatus(status Status) *ModelBuilder {
	b.status = status
	return b
}

// SetLastThrow sets the last recorded throw.
func (b *ModelBuilder) SetLastThrow(lastThrow Throw) *ModelBuilder {
	b.lastThrow = lastThrow
	return b
}

// SetCreatedAt sets the created-at timestamp.
func (b *ModelBuilder) SetCreatedAt(createdAt time.Time) *ModelBuilder {
	b.createdAt = createdAt
	return b
}

// SetUpdatedAt sets the updated-at timestamp.
func (b *ModelBuilder) SetUpdatedAt(updatedAt time.Time) *ModelBuilder {
	b.updatedAt = updatedAt
	return b
}

// Build validates and constructs the Model. Returns an error if validation fails.
func (b *ModelBuilder) Build() (Model, error) {
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
func (b *ModelBuilder) MustBuild() Model {
	m, err := b.Build()
	if err != nil {
		panic("MustBuild failed: " + err.Error())
	}
	return m
}

// Tenant returns the tenant from the builder.
func (b *ModelBuilder) Tenant() tenant.Model {
	return b.tenant
}

// CharacterId returns the characterId from the builder.
func (b *ModelBuilder) CharacterId() uint32 {
	return b.characterId
}

// WorldId returns the worldId from the builder.
func (b *ModelBuilder) WorldId() world.Id {
	return b.worldId
}

// ChannelId returns the channelId from the builder.
func (b *ModelBuilder) ChannelId() channel.Id {
	return b.channelId
}

// NpcId returns the npcId from the builder.
func (b *ModelBuilder) NpcId() uint32 {
	return b.npcId
}

// Rung returns the rung from the builder.
func (b *ModelBuilder) Rung() int {
	return b.rung
}

// Status returns the status from the builder.
func (b *ModelBuilder) Status() Status {
	return b.status
}

// LastThrow returns the lastThrow from the builder.
func (b *ModelBuilder) LastThrow() Throw {
	return b.lastThrow
}

// CreatedAt returns the createdAt from the builder.
func (b *ModelBuilder) CreatedAt() time.Time {
	return b.createdAt
}

// UpdatedAt returns the updatedAt from the builder.
func (b *ModelBuilder) UpdatedAt() time.Time {
	return b.updatedAt
}
