package party

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/Chronicle20/atlas/libs/atlas-constants/channel"
	_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"
	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
)

func TestExtract(t *testing.T) {
	rm := RestModel{
		Id: 7,
		Members: []MemberRestModel{
			{
				Id:        100,
				WorldId:   world.Id(1),
				ChannelId: channel.Id(1),
				MapId:     _map.Id(100000000),
				Instance:  uuid.MustParse("00000000-0000-0000-0000-000000000001"),
				Online:    true,
			},
			{
				Id:        200,
				WorldId:   world.Id(2),
				ChannelId: channel.Id(3),
				MapId:     _map.Id(200000000),
				Instance:  uuid.Nil,
				Online:    false,
			},
		},
	}

	m, err := Extract(rm)
	require.NoError(t, err)

	require.Equal(t, uint32(7), m.Id())
	require.Equal(t, 2, len(m.Members()))

	require.Equal(t, uint32(100), m.Members()[0].Id())
	require.Equal(t, true, m.Members()[0].Online())
	require.Equal(t, world.Id(1), m.Members()[0].Field().WorldId())
	require.Equal(t, channel.Id(1), m.Members()[0].Field().ChannelId())
	require.Equal(t, _map.Id(100000000), m.Members()[0].Field().MapId())
	require.Equal(t, uuid.MustParse("00000000-0000-0000-0000-000000000001"), m.Members()[0].Field().Instance())

	require.Equal(t, uint32(200), m.Members()[1].Id())
	require.Equal(t, false, m.Members()[1].Online())
	require.Equal(t, world.Id(2), m.Members()[1].Field().WorldId())
	require.Equal(t, channel.Id(3), m.Members()[1].Field().ChannelId())
	require.Equal(t, _map.Id(200000000), m.Members()[1].Field().MapId())
	require.Equal(t, uuid.Nil, m.Members()[1].Field().Instance())
}

func TestExtract_NoMembers(t *testing.T) {
	rm := RestModel{
		Id:      7,
		Members: nil,
	}

	m, err := Extract(rm)
	require.NoError(t, err)

	require.NotNil(t, m.Members())
	require.Equal(t, 0, len(m.Members()))
}
