package environment

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/Chronicle20/atlas/libs/atlas-constants/channel"
	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"
	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

func newTestField() field.Model {
	return field.NewBuilder(world.Id(0), channel.Id(1), _map.Id(910010000)).SetInstance(uuid.Nil).Build()
}

func newTestKey(t *testing.T) FieldKey {
	ten, err := tenant.Create(uuid.New(), "GMS", 83, 1)
	require.NoError(t, err)
	return FieldKey{Tenant: ten, Field: newTestField()}
}

func TestRegistrySetAppendsNewEntry(t *testing.T) {
	key := newTestKey(t)
	r := getRegistry()

	r.Set(key, ObjectEntry{Kind: field.ObjectKindObstacle, Name: "a", State: 1})
	r.Set(key, ObjectEntry{Kind: field.ObjectKindEnvironment, Name: "b", State: 2})

	got := r.Get(key)
	require.Equal(t, []ObjectEntry{
		{Kind: field.ObjectKindObstacle, Name: "a", State: 1},
		{Kind: field.ObjectKindEnvironment, Name: "b", State: 2},
	}, got)
}

func TestRegistrySetReplacesInPlace(t *testing.T) {
	key := newTestKey(t)
	r := getRegistry()

	r.Set(key, ObjectEntry{Kind: field.ObjectKindObstacle, Name: "a", State: 1})
	r.Set(key, ObjectEntry{Kind: field.ObjectKindEnvironment, Name: "b", State: 2})
	r.Set(key, ObjectEntry{Kind: field.ObjectKindObstacle, Name: "a", State: 7})

	got := r.Get(key)
	require.Len(t, got, 2)
	require.Equal(t, ObjectEntry{Kind: field.ObjectKindObstacle, Name: "a", State: 7}, got[0])
	require.Equal(t, ObjectEntry{Kind: field.ObjectKindEnvironment, Name: "b", State: 2}, got[1])
}

func TestRegistrySetSameNameDifferentKindAreDistinct(t *testing.T) {
	key := newTestKey(t)
	r := getRegistry()

	r.Set(key, ObjectEntry{Kind: field.ObjectKindObstacle, Name: "a", State: 1})
	r.Set(key, ObjectEntry{Kind: field.ObjectKindEnvironment, Name: "a", State: 2})

	got := r.Get(key)
	require.Len(t, got, 2)
}

func TestRegistryGetReturnsCopy(t *testing.T) {
	key := newTestKey(t)
	r := getRegistry()

	r.Set(key, ObjectEntry{Kind: field.ObjectKindObstacle, Name: "a", State: 1})

	got := r.Get(key)
	got[0].State = 99

	again := r.Get(key)
	require.Equal(t, uint32(1), again[0].State)
}

func TestRegistryGetUntrackedReturnsEmptyNotNil(t *testing.T) {
	key := newTestKey(t)
	r := getRegistry()

	got := r.Get(key)
	require.Len(t, got, 0)
	require.NotNil(t, got)
}

func TestRegistryTenantIsolation(t *testing.T) {
	tenA, err := tenant.Create(uuid.New(), "GMS", 83, 1)
	require.NoError(t, err)
	tenB, err := tenant.Create(uuid.New(), "GMS", 83, 1)
	require.NoError(t, err)
	f := newTestField()

	keyA := FieldKey{Tenant: tenA, Field: f}
	keyB := FieldKey{Tenant: tenB, Field: f}

	r := getRegistry()
	r.Set(keyA, ObjectEntry{Kind: field.ObjectKindEnvironment, Name: "a", State: 1})

	require.Len(t, r.Get(keyB), 0)
	require.Len(t, r.Get(keyA), 1)
}

func TestRegistryDeleteRemovesKey(t *testing.T) {
	key := newTestKey(t)
	r := getRegistry()

	r.Set(key, ObjectEntry{Kind: field.ObjectKindObstacle, Name: "a", State: 1})
	r.Set(key, ObjectEntry{Kind: field.ObjectKindEnvironment, Name: "b", State: 2})

	r.Delete(key)

	require.Len(t, r.Get(key), 0)
}
