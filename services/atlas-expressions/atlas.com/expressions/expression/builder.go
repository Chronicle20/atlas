package expression

import (
	"errors"
	"time"

	"github.com/Chronicle20/atlas-constants/channel"
	_map "github.com/Chronicle20/atlas-constants/map"
	"github.com/Chronicle20/atlas-constants/world"
	"github.com/Chronicle20/atlas-tenant"
	"github.com/google/uuid"
)

// ModelBuilder provides a fluent API for constructing expression.Model instances.
type ModelBuilder struct {
	tenant      tenant.Model
	characterId uint32
	worldId     world.Id
	channelId   channel.Id
	mapId       _map.Id
	expression  uint32
	expiration  time.Time
}

// NewModelBuilder creates a new ModelBuilder with required tenant.
func NewModelBuilder(t tenant.Model) *ModelBuilder {
	return &ModelBuilder{
		tenant: t,
	}
}

// CloneModelBuilder creates a new ModelBuilder initialized from an existing Model.
func CloneModelBuilder(m Model) *ModelBuilder {
	return &ModelBuilder{
		tenant:      m.Tenant(),
		characterId: m.CharacterId(),
		worldId:     m.WorldId(),
		channelId:   m.ChannelId(),
		mapId:       m.MapId(),
		expression:  m.Expression(),
		expiration:  m.Expiration(),
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

// SetMapId sets the map ID.
func (b *ModelBuilder) SetMapId(mapId _map.Id) *ModelBuilder {
	b.mapId = mapId
	return b
}

// SetExpression sets the expression value.
func (b *ModelBuilder) SetExpression(expression uint32) *ModelBuilder {
	b.expression = expression
	return b
}

// SetExpiration sets the expiration time.
func (b *ModelBuilder) SetExpiration(expiration time.Time) *ModelBuilder {
	b.expiration = expiration
	return b
}

// SetLocation sets worldId, channelId, and mapId together.
func (b *ModelBuilder) SetLocation(worldId world.Id, channelId channel.Id, mapId _map.Id) *ModelBuilder {
	b.worldId = worldId
	b.channelId = channelId
	b.mapId = mapId
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
	if b.expiration.IsZero() {
		return Model{}, errors.New("expiration is required")
	}
	return Model{
		tenant:      b.tenant,
		characterId: b.characterId,
		worldId:     b.worldId,
		channelId:   b.channelId,
		mapId:       b.mapId,
		expression:  b.expression,
		expiration:  b.expiration,
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

// MapId returns the mapId from the builder.
func (b *ModelBuilder) MapId() _map.Id {
	return b.mapId
}

// Expression returns the expression from the builder.
func (b *ModelBuilder) Expression() uint32 {
	return b.expression
}

// Expiration returns the expiration from the builder.
func (b *ModelBuilder) Expiration() time.Time {
	return b.expiration
}
