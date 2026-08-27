package party

import (
	"reflect"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/Chronicle20/atlas/libs/atlas-constants/channel"
	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	"github.com/Chronicle20/atlas/libs/atlas-constants/job"
	_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"
	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
)

func TestTransformRoundTrip(t *testing.T) {
	t.Run("Transform", func(t *testing.T) {
		instance := uuid.New()
		m := Model{
			id:       9000,
			leaderId: 30001,
			members: []MemberModel{
				{
					id:     30001,
					name:   "Leader",
					level:  120,
					jobId:  job.Id(112),
					field:  field.NewBuilder(world.Id(3), channel.Id(2), _map.Id(910000004)).SetInstance(instance).Build(),
					online: true,
				},
				{
					id:     30002,
					name:   "Member",
					level:  95,
					jobId:  job.Id(200),
					field:  field.NewBuilder(world.Id(3), channel.Id(2), _map.Id(910000004)).SetInstance(instance).Build(),
					online: false,
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
		instance := uuid.New()
		mm := MemberModel{
			id:     30003,
			name:   "Solo",
			level:  200,
			jobId:  job.Id(412),
			field:  field.NewBuilder(world.Id(7), channel.Id(11), _map.Id(104000000)).SetInstance(instance).Build(),
			online: true,
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
