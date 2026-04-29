package mist

import (
	"errors"
	"testing"
	"time"

	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func newTestMistRegistry() *Registry {
	return &Registry{perTenant: map[string]*tenantBucket{}}
}

func mkRegTenant() tenant.Model {
	t, _ := tenant.Create(uuid.New(), "GMS", 83, 1)
	return t
}

func TestRegistry_Add_GetByField(t *testing.T) {
	r := newTestMistRegistry()
	tt := mkRegTenant()
	f := field.NewBuilder(0, 0, 100000000).SetInstance(uuid.Nil).Build()
	id := uuid.New()
	m := NewBuilder(id, f).SetOrigin(0, 0).SetBounds(-1, -1, 1, 1).SetDuration(time.Minute).Build()

	require.NoError(t, r.Add(tt, m))
	got := r.GetByField(tt, f)
	require.Len(t, got, 1)
	require.Equal(t, id, got[0].Id())
}

func TestRegistry_Remove_ReturnsRemovedMist(t *testing.T) {
	r := newTestMistRegistry()
	tt := mkRegTenant()
	f := field.NewBuilder(0, 0, 100000000).SetInstance(uuid.Nil).Build()
	id := uuid.New()
	m := NewBuilder(id, f).SetOrigin(0, 0).SetBounds(-1, -1, 1, 1).SetDuration(time.Minute).Build()
	_ = r.Add(tt, m)

	removed, err := r.Remove(tt, id)
	require.NoError(t, err)
	require.Equal(t, id, removed.Id())
	require.Empty(t, r.GetByField(tt, f))
}

func TestRegistry_GetByField_DistinguishesInstances(t *testing.T) {
	r := newTestMistRegistry()
	tt := mkRegTenant()
	f1 := field.NewBuilder(0, 0, 100000000).SetInstance(uuid.MustParse("aaaaaaaa-0000-0000-0000-000000000001")).Build()
	f2 := field.NewBuilder(0, 0, 100000000).SetInstance(uuid.MustParse("bbbbbbbb-0000-0000-0000-000000000002")).Build()
	mistA := NewBuilder(uuid.New(), f1).SetOrigin(0, 0).SetBounds(-1, -1, 1, 1).SetDuration(time.Minute).Build()
	_ = r.Add(tt, mistA)

	require.Len(t, r.GetByField(tt, f1), 1)
	require.Len(t, r.GetByField(tt, f2), 0, "different instance UUID — no overlap")
}

func TestRegistry_AllByTenant_ReturnsAcrossFields(t *testing.T) {
	r := newTestMistRegistry()
	tt := mkRegTenant()
	f1 := field.NewBuilder(0, 0, 100000000).SetInstance(uuid.Nil).Build()
	f2 := field.NewBuilder(0, 0, 200000000).SetInstance(uuid.Nil).Build()
	_ = r.Add(tt, NewBuilder(uuid.New(), f1).SetDuration(time.Minute).Build())
	_ = r.Add(tt, NewBuilder(uuid.New(), f2).SetDuration(time.Minute).Build())

	require.Len(t, r.AllByTenant(tt), 2)
}

func TestRegistry_Add_DuplicateId_ReturnsError(t *testing.T) {
	r := newTestMistRegistry()
	tt := mkRegTenant()
	f := field.NewBuilder(0, 0, 100000000).SetInstance(uuid.Nil).Build()
	id := uuid.New()
	m := NewBuilder(id, f).SetOrigin(0, 0).SetBounds(-1, -1, 1, 1).SetDuration(time.Minute).Build()

	require.NoError(t, r.Add(tt, m))
	err := r.Add(tt, m)
	require.Error(t, err)
	require.True(t, errors.Is(err, ErrAlreadyExists))
}

func TestRegistry_GetTenants_ReturnsAllAddedTenants(t *testing.T) {
	r := newTestMistRegistry()
	t1 := mkRegTenant()
	t2 := mkRegTenant()
	f := field.NewBuilder(0, 0, 100000000).SetInstance(uuid.Nil).Build()
	_ = r.Add(t1, NewBuilder(uuid.New(), f).SetDuration(time.Minute).Build())
	_ = r.Add(t2, NewBuilder(uuid.New(), f).SetDuration(time.Minute).Build())
	require.Len(t, r.GetTenants(), 2)
}
