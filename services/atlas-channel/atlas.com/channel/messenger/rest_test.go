package messenger

import (
	"reflect"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/Chronicle20/atlas/libs/atlas-constants/channel"
	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
)

func TestTransformRoundTrip(t *testing.T) {
	t.Run("Transform", func(t *testing.T) {
		m := Model{
			id: 1000,
			members: []MemberModel{
				{
					id:        30001,
					name:      "Bishop",
					worldId:   world.Id(3),
					channelId: channel.Id(2),
					online:    true,
					slot:      1,
				},
				{
					id:        30002,
					name:      "Priest",
					worldId:   world.Id(5),
					channelId: channel.Id(4),
					online:    false,
					slot:      2,
				},
			},
		}

		rm, err := Transform(m)
		require.NoError(t, err)

		got, err := Extract(rm)
		require.NoError(t, err)

		if !reflect.DeepEqual(got, m) {
			t.Fatalf("round trip mismatch:\n got  = %+v\n want = %+v", got, m)
		}
	})

	t.Run("Member", func(t *testing.T) {
		mm := MemberModel{
			id:        30003,
			name:      "Cleric",
			worldId:   world.Id(1),
			channelId: channel.Id(9),
			online:    true,
			slot:      3,
		}

		rm, err := TransformMember(mm)
		require.NoError(t, err)

		got, err := ExtractMember(rm)
		require.NoError(t, err)

		if !reflect.DeepEqual(got, mm) {
			t.Fatalf("round trip mismatch:\n got  = %+v\n want = %+v", got, mm)
		}
	})
}
