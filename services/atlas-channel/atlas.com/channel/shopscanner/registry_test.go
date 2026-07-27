package shopscanner

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

func testTenant(t *testing.T) tenant.Model {
	t.Helper()
	m, err := tenant.Create(uuid.New(), "GMS", 83, 1)
	require.NoError(t, err)
	return m
}

func TestRegistry_LastSearchLifecycle(t *testing.T) {
	r := GetRegistry()
	ta, tb := testTenant(t), testTenant(t)

	_, ok := r.GetLastSearch(ta, 1)
	require.False(t, ok)

	r.SetLastSearch(ta, 1, 2060000)
	e, ok := r.GetLastSearch(ta, 1)
	require.True(t, ok)
	require.Equal(t, uint32(2060000), e.ItemId)

	// overwrite on reuse
	r.SetLastSearch(ta, 1, 1302000)
	e, _ = r.GetLastSearch(ta, 1)
	require.Equal(t, uint32(1302000), e.ItemId)

	// tenant isolation
	_, ok = r.GetLastSearch(tb, 1)
	require.False(t, ok)

	r.ClearCharacter(ta, 1)
	_, ok = r.GetLastSearch(ta, 1)
	require.False(t, ok)
}

func TestRegistry_PendingEntryLifecycle(t *testing.T) {
	r := GetRegistry()
	ta := testTenant(t)
	shopId := uuid.New()

	r.SetPending(ta, 2, PendingEntry{ShopId: shopId, OwnerId: 30001, MapId: _map.Id(910000004)})
	pe, ok := r.GetPending(ta, 2)
	require.True(t, ok)
	require.Equal(t, shopId, pe.ShopId)
	require.Equal(t, uint32(30001), pe.OwnerId)
	require.Equal(t, _map.Id(910000004), pe.MapId)

	r.RemovePending(ta, 2)
	_, ok = r.GetPending(ta, 2)
	require.False(t, ok)
}

func TestRegistry_ClearCharacterClearsBoth(t *testing.T) {
	r := GetRegistry()
	ta := testTenant(t)
	r.SetLastSearch(ta, 3, 2060000)
	r.SetPending(ta, 3, PendingEntry{ShopId: uuid.New(), OwnerId: 30001, MapId: _map.Id(910000004)})
	r.ClearCharacter(ta, 3)
	_, ok1 := r.GetLastSearch(ta, 3)
	_, ok2 := r.GetPending(ta, 3)
	require.False(t, ok1)
	require.False(t, ok2)
}
