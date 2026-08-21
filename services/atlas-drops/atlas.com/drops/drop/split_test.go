package drop

import (
	"atlas-drops/party"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/Chronicle20/atlas/libs/atlas-constants/channel"
	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"
	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
)

func TestSplitMeso(t *testing.T) {
	instA := uuid.MustParse("00000000-0000-0000-0000-00000000000a")
	instB := uuid.MustParse("00000000-0000-0000-0000-00000000000b")
	f := field.NewBuilder(world.Id(1), channel.Id(1), _map.Id(100000000)).SetInstance(instA).Build()
	member := func(id uint32, mf field.Model, online bool) party.MemberModel {
		return party.NewMemberBuilder().SetId(id).SetField(mf).SetOnline(online).Build()
	}
	onField := func(id uint32) party.MemberModel { return member(id, f, true) }

	tests := []struct {
		name     string
		meso     uint32
		pickerId uint32
		members  []party.MemberModel
		expected []Recipient
	}{
		{
			name:     "no party",
			meso:     100,
			pickerId: 10,
			members:  nil,
			expected: []Recipient{{10, 100, true}},
		},
		{
			name:     "empty member list",
			meso:     100,
			pickerId: 10,
			members:  []party.MemberModel{},
			expected: []Recipient{{10, 100, true}},
		},
		{
			name:     "party of one is the picker",
			meso:     100,
			pickerId: 10,
			members:  []party.MemberModel{onField(10)},
			expected: []Recipient{{10, 100, true}},
		},
		{
			name:     "party of three all eligible",
			meso:     100,
			pickerId: 10,
			members:  []party.MemberModel{onField(10), onField(20), onField(30)},
			expected: []Recipient{{10, 33, true}, {20, 33, false}, {30, 33, false}},
		},
		{
			name:     "offline member excluded",
			meso:     100,
			pickerId: 10,
			members:  []party.MemberModel{onField(10), member(20, f, false)},
			expected: []Recipient{{10, 100, true}},
		},
		{
			name:     "different world excluded",
			meso:     100,
			pickerId: 10,
			members: []party.MemberModel{
				onField(10),
				member(20, field.NewBuilder(world.Id(2), channel.Id(1), _map.Id(100000000)).SetInstance(instA).Build(), true),
			},
			expected: []Recipient{{10, 100, true}},
		},
		{
			name:     "different channel excluded",
			meso:     100,
			pickerId: 10,
			members: []party.MemberModel{
				onField(10),
				member(20, field.NewBuilder(world.Id(1), channel.Id(2), _map.Id(100000000)).SetInstance(instA).Build(), true),
			},
			expected: []Recipient{{10, 100, true}},
		},
		{
			name:     "different map excluded",
			meso:     100,
			pickerId: 10,
			members: []party.MemberModel{
				onField(10),
				member(20, field.NewBuilder(world.Id(1), channel.Id(1), _map.Id(200000000)).SetInstance(instA).Build(), true),
			},
			expected: []Recipient{{10, 100, true}},
		},
		{
			name:     "different instance excluded",
			meso:     100,
			pickerId: 10,
			members: []party.MemberModel{
				onField(10),
				member(20, field.NewBuilder(world.Id(1), channel.Id(1), _map.Id(100000000)).SetInstance(instB).Build(), true),
			},
			expected: []Recipient{{10, 100, true}},
		},
		{
			name:     "duplicate member ids collapsed",
			meso:     100,
			pickerId: 10,
			members:  []party.MemberModel{onField(20), onField(20)},
			expected: []Recipient{{10, 50, true}, {20, 50, false}},
		},
		{
			name:     "remainder discarded",
			meso:     100,
			pickerId: 10,
			members:  []party.MemberModel{onField(20), onField(30)},
			expected: []Recipient{{10, 33, true}, {20, 33, false}, {30, 33, false}},
		},
		{
			name:     "meso less than recipient count",
			meso:     2,
			pickerId: 10,
			members:  []party.MemberModel{onField(20), onField(30)},
			expected: []Recipient{{10, 0, true}, {20, 0, false}, {30, 0, false}},
		},
		{
			name:     "picker included despite offline own record",
			meso:     100,
			pickerId: 10,
			members:  []party.MemberModel{member(10, f, false), onField(20)},
			expected: []Recipient{{10, 50, true}, {20, 50, false}},
		},
		{
			name:     "picker included despite stale field on own record",
			meso:     100,
			pickerId: 10,
			members: []party.MemberModel{
				member(10, field.NewBuilder(world.Id(1), channel.Id(1), _map.Id(200000000)).SetInstance(instA).Build(), true),
				onField(20),
			},
			expected: []Recipient{{10, 50, true}, {20, 50, false}},
		},
		{
			name:     "sorted by character id",
			meso:     90,
			pickerId: 25,
			members:  []party.MemberModel{onField(30), onField(20)},
			expected: []Recipient{{20, 30, false}, {25, 30, true}, {30, 30, false}},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := splitMeso(f, tc.meso, tc.pickerId, tc.members)
			require.Equal(t, tc.expected, got)
		})
	}
}

func TestSplitMeso_ExactlyOnePicker(t *testing.T) {
	instA := uuid.MustParse("00000000-0000-0000-0000-00000000000a")
	f := field.NewBuilder(world.Id(1), channel.Id(1), _map.Id(100000000)).SetInstance(instA).Build()
	member := func(id uint32, mf field.Model, online bool) party.MemberModel {
		return party.NewMemberBuilder().SetId(id).SetField(mf).SetOnline(online).Build()
	}
	onField := func(id uint32) party.MemberModel { return member(id, f, true) }

	got := splitMeso(f, 100, 10, []party.MemberModel{onField(10), onField(20), onField(30)})

	n := 0
	for _, r := range got {
		if r.Picker {
			n++
		}
	}
	require.Equal(t, 1, n)
}

func TestSplitMeso_RemainderIsDiscarded(t *testing.T) {
	instA := uuid.MustParse("00000000-0000-0000-0000-00000000000a")
	f := field.NewBuilder(world.Id(1), channel.Id(1), _map.Id(100000000)).SetInstance(instA).Build()
	member := func(id uint32, mf field.Model, online bool) party.MemberModel {
		return party.NewMemberBuilder().SetId(id).SetField(mf).SetOnline(online).Build()
	}
	onField := func(id uint32) party.MemberModel { return member(id, f, true) }

	got := splitMeso(f, 100, 10, []party.MemberModel{onField(10), onField(20), onField(30)})

	var total uint32
	for _, r := range got {
		total += r.Amount
	}
	require.Equal(t, uint32(99), total)
}
