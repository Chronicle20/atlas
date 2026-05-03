package timer

import (
	"testing"
	"time"

	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func mkRegEntry(t *testing.T, tenantA interface{ /* placeholder */ }) Entry {
	t.Helper()
	f := field.NewBuilder(0, 0, 100000000).SetInstance(uuid.Nil).Build()
	return NewEntryBuilder().
		SetCharacterId(42).
		SetField(f).
		SetForcedReturnMapId(_map.Id(100000201)).
		SetSeconds(600).
		SetToken(uuid.New()).
		SetExpiresAt(time.Now().Add(10 * time.Minute)).
		Build()
}

func TestRegistry_Add_StoresEntry(t *testing.T) {
	tt := mkTenant(t)
	r := NewTestRegistry()
	e := mkRegEntry(t, tt)
	e2 := NewEntryBuilder().
		SetTenant(tt).
		SetCharacterId(e.CharacterId()).
		SetField(e.Field()).
		SetForcedReturnMapId(e.ForcedReturnMapId()).
		SetSeconds(e.Seconds()).
		SetToken(e.Token()).
		SetExpiresAt(e.ExpiresAt()).
		Build()

	require.NoError(t, r.Add(e2))

	got, ok := r.Get(tt, 42)
	require.True(t, ok)
	require.Equal(t, e2.Token(), got.Token())
}

func TestRegistry_Cancel_RemovesEntry(t *testing.T) {
	tt := mkTenant(t)
	r := NewTestRegistry()
	e := NewEntryBuilder().
		SetTenant(tt).
		SetCharacterId(42).
		SetForcedReturnMapId(_map.Id(100000201)).
		SetToken(uuid.New()).
		Build()
	require.NoError(t, r.Add(e))

	got, ok := r.Cancel(tt, 42)
	require.True(t, ok)
	require.Equal(t, e.Token(), got.Token())

	_, ok = r.Get(tt, 42)
	require.False(t, ok, "Cancel must remove the entry")
}

func TestRegistry_Cancel_AbsentIsNoOp(t *testing.T) {
	tt := mkTenant(t)
	r := NewTestRegistry()
	_, ok := r.Cancel(tt, 999)
	require.False(t, ok, "Cancel on absent key returns false")
}

func TestRegistry_Add_ReplacesExistingEntry(t *testing.T) {
	tt := mkTenant(t)
	r := NewTestRegistry()
	first := NewEntryBuilder().SetTenant(tt).SetCharacterId(42).SetToken(uuid.New()).Build()
	second := NewEntryBuilder().SetTenant(tt).SetCharacterId(42).SetToken(uuid.New()).Build()
	require.NoError(t, r.Add(first))
	require.NoError(t, r.Add(second))

	got, ok := r.Get(tt, 42)
	require.True(t, ok)
	require.Equal(t, second.Token(), got.Token(), "second Add overwrites prior entry")
}

func TestRegistry_TenantsIsolated(t *testing.T) {
	t1 := mkTenant(t)
	t2 := mkTenant(t)
	r := NewTestRegistry()
	require.NoError(t, r.Add(NewEntryBuilder().SetTenant(t1).SetCharacterId(42).SetToken(uuid.New()).Build()))
	_, ok := r.Get(t2, 42)
	require.False(t, ok, "Other tenant must not see entry")
}
