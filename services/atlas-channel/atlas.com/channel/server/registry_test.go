package server_test

import (
	"testing"

	"atlas-channel/server"

	"github.com/Chronicle20/atlas/libs/atlas-constants/channel"
	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
	"github.com/Chronicle20/atlas/libs/atlas-tenant"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func mustTenant(t *testing.T) tenant.Model {
	tm, err := tenant.Create(uuid.New(), "GMS", 83, 1)
	require.NoError(t, err)
	return tm
}

func TestRegistry_DeregisterRemoves(t *testing.T) {
	tm := mustTenant(t)
	ch := channel.NewModel(world.Id(1), channel.Id(1))
	m := server.Register(tm, ch, "127.0.0.1", 8585)
	k := server.KeyOf(m)

	r := server.GetRegistry()
	_, ok := r.Get(k)
	require.True(t, ok, "registered entry should be retrievable")
	r.Deregister(k)
	_, ok = r.Get(k)
	require.False(t, ok, "deregistered entry must be gone")
}

func TestRegistry_GetAllReturnsCurrentMembers(t *testing.T) {
	tm := mustTenant(t)
	m1 := server.Register(tm, channel.NewModel(world.Id(2), channel.Id(0)), "10.0.0.1", 8585)
	m2 := server.Register(tm, channel.NewModel(world.Id(2), channel.Id(1)), "10.0.0.2", 8586)

	r := server.GetRegistry()

	got := r.GetAll()
	keys := make(map[server.Key]bool, len(got))
	for _, m := range got {
		keys[server.KeyOf(m)] = true
	}
	require.True(t, keys[server.KeyOf(m1)])
	require.True(t, keys[server.KeyOf(m2)])

	r.Deregister(server.KeyOf(m1))
	got = r.GetAll()
	keys = map[server.Key]bool{}
	for _, m := range got {
		keys[server.KeyOf(m)] = true
	}
	require.False(t, keys[server.KeyOf(m1)], "deregistered m1 must not appear")
	require.True(t, keys[server.KeyOf(m2)], "m2 still present")

	// cleanup so other tests aren't poisoned
	r.Deregister(server.KeyOf(m2))
}
