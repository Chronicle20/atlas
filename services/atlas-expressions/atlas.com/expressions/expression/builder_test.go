package expression

import (
	"testing"
	"time"

	"github.com/Chronicle20/atlas-constants/channel"
	"github.com/Chronicle20/atlas-constants/field"
	_map "github.com/Chronicle20/atlas-constants/map"
	"github.com/Chronicle20/atlas-constants/world"
	"github.com/Chronicle20/atlas-tenant"
	"github.com/stretchr/testify/assert"
)

func TestNewModelBuilder(t *testing.T) {
	ten := setupTestTenant(t)

	b := NewModelBuilder(ten)

	assert.NotNil(t, b)
	assert.Equal(t, ten, b.Tenant())
}

func TestModelBuilder_SetCharacterId(t *testing.T) {
	ten := setupTestTenant(t)

	b := NewModelBuilder(ten).SetCharacterId(1000)

	assert.Equal(t, uint32(1000), b.CharacterId())
}

func TestModelBuilder_SetWorldId(t *testing.T) {
	ten := setupTestTenant(t)

	b := NewModelBuilder(ten).SetWorldId(world.Id(5))

	assert.Equal(t, world.Id(5), b.WorldId())
}

func TestModelBuilder_SetChannelId(t *testing.T) {
	ten := setupTestTenant(t)

	b := NewModelBuilder(ten).SetChannelId(channel.Id(3))

	assert.Equal(t, channel.Id(3), b.ChannelId())
}

func TestModelBuilder_SetMapId(t *testing.T) {
	ten := setupTestTenant(t)

	b := NewModelBuilder(ten).SetMapId(_map.Id(100000000))

	assert.Equal(t, _map.Id(100000000), b.MapId())
}

func TestModelBuilder_SetExpression(t *testing.T) {
	ten := setupTestTenant(t)

	b := NewModelBuilder(ten).SetExpression(7)

	assert.Equal(t, uint32(7), b.Expression())
}

func TestModelBuilder_SetExpiration(t *testing.T) {
	ten := setupTestTenant(t)
	expiration := time.Now().Add(5 * time.Second)

	b := NewModelBuilder(ten).SetExpiration(expiration)

	assert.Equal(t, expiration, b.Expiration())
}

func TestModelBuilder_SetLocation(t *testing.T) {
	ten := setupTestTenant(t)

	f := field.NewBuilder(world.Id(1), channel.Id(2), _map.Id(300000000)).Build()
	b := NewModelBuilder(ten).SetLocation(f)

	assert.Equal(t, world.Id(1), b.WorldId())
	assert.Equal(t, channel.Id(2), b.ChannelId())
	assert.Equal(t, _map.Id(300000000), b.MapId())
}

func TestModelBuilder_FluentChaining(t *testing.T) {
	ten := setupTestTenant(t)
	expiration := time.Now().Add(5 * time.Second)

	b := NewModelBuilder(ten).
		SetCharacterId(1000).
		SetWorldId(world.Id(1)).
		SetChannelId(channel.Id(2)).
		SetMapId(_map.Id(100000000)).
		SetExpression(5).
		SetExpiration(expiration)

	assert.Equal(t, ten, b.Tenant())
	assert.Equal(t, uint32(1000), b.CharacterId())
	assert.Equal(t, world.Id(1), b.WorldId())
	assert.Equal(t, channel.Id(2), b.ChannelId())
	assert.Equal(t, _map.Id(100000000), b.MapId())
	assert.Equal(t, uint32(5), b.Expression())
	assert.Equal(t, expiration, b.Expiration())
}

func TestModelBuilder_Build_Success(t *testing.T) {
	ten := setupTestTenant(t)
	expiration := time.Now().Add(5 * time.Second)

	f := field.NewBuilder(world.Id(1), channel.Id(2), _map.Id(100000000)).Build()
	m, err := NewModelBuilder(ten).
		SetCharacterId(1000).
		SetLocation(f).
		SetExpression(5).
		SetExpiration(expiration).
		Build()

	assert.NoError(t, err)
	assert.Equal(t, ten, m.Tenant())
	assert.Equal(t, uint32(1000), m.CharacterId())
	assert.Equal(t, world.Id(1), m.WorldId())
	assert.Equal(t, channel.Id(2), m.ChannelId())
	assert.Equal(t, _map.Id(100000000), m.MapId())
	assert.Equal(t, uint32(5), m.Expression())
	assert.Equal(t, expiration, m.Expiration())
}

func TestModelBuilder_Build_MissingTenant(t *testing.T) {
	emptyTenant := tenant.Model{}
	expiration := time.Now().Add(5 * time.Second)

	_, err := NewModelBuilder(emptyTenant).
		SetCharacterId(1000).
		SetExpiration(expiration).
		Build()

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "tenant is required")
}

func TestModelBuilder_Build_MissingCharacterId(t *testing.T) {
	ten := setupTestTenant(t)
	expiration := time.Now().Add(5 * time.Second)

	_, err := NewModelBuilder(ten).
		SetExpiration(expiration).
		Build()

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "characterId is required")
}

func TestModelBuilder_Build_MissingExpiration(t *testing.T) {
	ten := setupTestTenant(t)

	_, err := NewModelBuilder(ten).
		SetCharacterId(1000).
		Build()

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "expiration is required")
}

func TestModelBuilder_MustBuild_Success(t *testing.T) {
	ten := setupTestTenant(t)
	expiration := time.Now().Add(5 * time.Second)

	m := NewModelBuilder(ten).
		SetCharacterId(1000).
		SetExpiration(expiration).
		MustBuild()

	assert.Equal(t, ten, m.Tenant())
	assert.Equal(t, uint32(1000), m.CharacterId())
}

func TestModelBuilder_MustBuild_Panics(t *testing.T) {
	emptyTenant := tenant.Model{}

	assert.Panics(t, func() {
		NewModelBuilder(emptyTenant).
			SetCharacterId(1000).
			SetExpiration(time.Now().Add(5 * time.Second)).
			MustBuild()
	})
}

func TestCloneModelBuilder(t *testing.T) {
	ten := setupTestTenant(t)
	expiration := time.Now().Add(5 * time.Second)

	f := field.NewBuilder(world.Id(1), channel.Id(2), _map.Id(100000000)).Build()
	original := NewModelBuilder(ten).
		SetCharacterId(1000).
		SetLocation(f).
		SetExpression(5).
		SetExpiration(expiration).
		MustBuild()

	b := CloneModelBuilder(original)

	assert.Equal(t, original.Tenant(), b.Tenant())
	assert.Equal(t, original.CharacterId(), b.CharacterId())
	assert.Equal(t, original.WorldId(), b.WorldId())
	assert.Equal(t, original.ChannelId(), b.ChannelId())
	assert.Equal(t, original.MapId(), b.MapId())
	assert.Equal(t, original.Expression(), b.Expression())
	assert.Equal(t, original.Expiration(), b.Expiration())
}

func TestCloneModelBuilder_Modify(t *testing.T) {
	ten := setupTestTenant(t)
	expiration := time.Now().Add(5 * time.Second)

	f := field.NewBuilder(world.Id(1), channel.Id(2), _map.Id(100000000)).Build()
	original := NewModelBuilder(ten).
		SetCharacterId(1000).
		SetLocation(f).
		SetExpression(5).
		SetExpiration(expiration).
		MustBuild()

	// Clone and modify only the expression
	modified := CloneModelBuilder(original).
		SetExpression(10).
		MustBuild()

	// Original should be unchanged (immutability)
	assert.Equal(t, uint32(5), original.Expression())

	// Modified should have new expression but same other fields
	assert.Equal(t, uint32(10), modified.Expression())
	assert.Equal(t, original.CharacterId(), modified.CharacterId())
	assert.Equal(t, original.WorldId(), modified.WorldId())
	assert.Equal(t, original.ChannelId(), modified.ChannelId())
	assert.Equal(t, original.MapId(), modified.MapId())
}

func TestModelBuilder_ZeroExpressionIsValid(t *testing.T) {
	ten := setupTestTenant(t)
	expiration := time.Now().Add(5 * time.Second)

	// Expression 0 is valid (means "revert to default")
	m, err := NewModelBuilder(ten).
		SetCharacterId(1000).
		SetExpression(0).
		SetExpiration(expiration).
		Build()

	assert.NoError(t, err)
	assert.Equal(t, uint32(0), m.Expression())
}
