package server_test

import (
	"testing"

	"atlas-channel/server"

	"github.com/Chronicle20/atlas/libs/atlas-constants/channel"
	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestKey_Equality(t *testing.T) {
	id := uuid.New()
	a := server.Key{TenantId: id, WorldId: world.Id(1), ChannelId: channel.Id(2)}
	b := server.Key{TenantId: id, WorldId: world.Id(1), ChannelId: channel.Id(2)}
	require.Equal(t, a, b)
}

func TestKey_UsableAsMapKey(t *testing.T) {
	id := uuid.New()
	k := server.Key{TenantId: id, WorldId: world.Id(1), ChannelId: channel.Id(2)}
	m := map[server.Key]int{k: 42}
	require.Equal(t, 42, m[server.Key{TenantId: id, WorldId: world.Id(1), ChannelId: channel.Id(2)}])
}
