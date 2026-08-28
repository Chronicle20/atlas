package expression

import (
	"errors"
	"time"

	"github.com/google/uuid"

	"github.com/Chronicle20/atlas/libs/atlas-constants/channel"
	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"
	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

// Builder provides a fluent API for constructing expression.Model instances.
type Builder struct {
	tenant      tenant.Model
	characterId uint32
	field       field.Model
	expression  uint32
	expiration  time.Time
}

// NewBuilder creates a new Builder with required tenant.
func NewBuilder(t tenant.Model) *Builder {
	return &Builder{
		tenant: t,
	}
}

// CloneBuilder creates a new Builder initialized from an existing Model.
func CloneBuilder(m Model) *Builder {
	return &Builder{
		tenant:      m.Tenant(),
		characterId: m.CharacterId(),
		field:       m.Field(),
		expression:  m.Expression(),
		expiration:  m.Expiration(),
	}
}

// SetCharacterId sets the character ID.
func (b *Builder) SetCharacterId(characterId uint32) *Builder {
	b.characterId = characterId
	return b
}

// SetWorldId sets the world ID.
func (b *Builder) SetWorldId(worldId world.Id) *Builder {
	b.field = b.field.Clone().SetWorldId(worldId).Build()
	return b
}

// SetChannelId sets the channel ID.
func (b *Builder) SetChannelId(channelId channel.Id) *Builder {
	b.field = b.field.Clone().SetChannelId(channelId).Build()
	return b
}

// SetMapId sets the map ID.
func (b *Builder) SetMapId(mapId _map.Id) *Builder {
	b.field = b.field.Clone().SetMapId(mapId).Build()
	return b
}

// SetInstance sets the instance UUID.
func (b *Builder) SetInstance(instance uuid.UUID) *Builder {
	b.field = b.field.Clone().SetInstance(instance).Build()
	return b
}

// SetExpression sets the expression value.
func (b *Builder) SetExpression(expression uint32) *Builder {
	b.expression = expression
	return b
}

// SetExpiration sets the expiration time.
func (b *Builder) SetExpiration(expiration time.Time) *Builder {
	b.expiration = expiration
	return b
}

// SetLocation sets worldId, channelId, mapId, and instance together.
func (b *Builder) SetLocation(field field.Model) *Builder {
	b.field = field
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
	if b.expiration.IsZero() {
		return Model{}, errors.New("expiration is required")
	}
	return Model{
		tenant:      b.tenant,
		characterId: b.characterId,
		field:       b.field,
		expression:  b.expression,
		expiration:  b.expiration,
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
	return b.field.WorldId()
}

// ChannelId returns the channelId from the builder.
func (b *Builder) ChannelId() channel.Id {
	return b.field.ChannelId()
}

// MapId returns the mapId from the builder.
func (b *Builder) MapId() _map.Id {
	return b.field.MapId()
}

// Instance returns the instance from the builder.
func (b *Builder) Instance() uuid.UUID {
	return b.field.Instance()
}

// Expression returns the expression from the builder.
func (b *Builder) Expression() uint32 {
	return b.expression
}

// Expiration returns the expiration from the builder.
func (b *Builder) Expiration() time.Time {
	return b.expiration
}
