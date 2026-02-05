package expression

import (
	"testing"
	"time"

	"github.com/Chronicle20/atlas-constants/channel"
	"github.com/Chronicle20/atlas-constants/field"
	_map "github.com/Chronicle20/atlas-constants/map"
	"github.com/Chronicle20/atlas-constants/world"
	"github.com/Chronicle20/atlas-tenant"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func setupTestTenant(t *testing.T) tenant.Model {
	t.Helper()
	ten, err := tenant.Create(uuid.New(), "GMS", 83, 1)
	if err != nil {
		t.Fatalf("Failed to create tenant: %v", err)
	}
	return ten
}

func createTestModel(ten tenant.Model) Model {
	return Model{
		tenant:      ten,
		characterId: 1000,
		field:       field.NewBuilder(0, 1, 100000000).Build(),
		expression:  5,
		expiration:  time.Now().Add(5 * time.Second),
	}
}

func TestModel_Tenant(t *testing.T) {
	ten := setupTestTenant(t)
	m := createTestModel(ten)

	assert.Equal(t, ten, m.Tenant())
}

func TestModel_CharacterId(t *testing.T) {
	ten := setupTestTenant(t)
	m := createTestModel(ten)

	assert.Equal(t, uint32(1000), m.CharacterId())
}

func TestModel_WorldId(t *testing.T) {
	ten := setupTestTenant(t)
	m := createTestModel(ten)

	assert.Equal(t, world.Id(0), m.WorldId())
}

func TestModel_ChannelId(t *testing.T) {
	ten := setupTestTenant(t)
	m := createTestModel(ten)

	assert.Equal(t, channel.Id(1), m.ChannelId())
}

func TestModel_MapId(t *testing.T) {
	ten := setupTestTenant(t)
	m := createTestModel(ten)

	assert.Equal(t, _map.Id(100000000), m.MapId())
}

func TestModel_Expression(t *testing.T) {
	ten := setupTestTenant(t)
	m := createTestModel(ten)

	assert.Equal(t, uint32(5), m.Expression())
}

func TestModel_Expiration(t *testing.T) {
	ten := setupTestTenant(t)
	now := time.Now()
	m := Model{
		tenant:      ten,
		characterId: 1000,
		field:       field.NewBuilder(0, 1, 100000000).Build(),
		expression:  5,
		expiration:  now.Add(5 * time.Second),
	}

	// Expiration should be approximately 5 seconds from now
	diff := m.Expiration().Sub(now)
	assert.True(t, diff >= 4*time.Second && diff <= 6*time.Second,
		"Expiration should be approximately 5 seconds from creation")
}

func TestModel_AllAccessors(t *testing.T) {
	ten := setupTestTenant(t)
	expiration := time.Now().Add(5 * time.Second)

	m := Model{
		tenant:      ten,
		characterId: 2000,
		field:       field.NewBuilder(0, 2, 200000000).Build(),
		expression:  10,
		expiration:  expiration,
	}

	// Verify all accessors return expected values
	assert.Equal(t, ten, m.Tenant())
	assert.Equal(t, uint32(2000), m.CharacterId())
	assert.Equal(t, world.Id(1), m.WorldId())
	assert.Equal(t, channel.Id(2), m.ChannelId())
	assert.Equal(t, _map.Id(200000000), m.MapId())
	assert.Equal(t, uint32(10), m.Expression())
	assert.Equal(t, expiration, m.Expiration())
}
