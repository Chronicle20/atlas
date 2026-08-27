package party

import (
	"reflect"
	"testing"

	"github.com/Chronicle20/atlas/libs/atlas-constants/channel"
	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
)

func TestTransformRoundTrip(t *testing.T) {
	t.Run("Party", func(t *testing.T) {
		m := Model{
			id:       1001,
			leaderId: 2002,
			members: []MemberModel{
				{id: 3003, worldId: world.Id(1), channelId: channel.Id(2)},
			},
		}

		rm, err := Transform(m)
		if err != nil {
			t.Fatalf("Transform returned error: %v", err)
		}

		got, err := Extract(rm)
		if err != nil {
			t.Fatalf("Extract returned error: %v", err)
		}

		if !reflect.DeepEqual(got, m) {
			t.Fatalf("round trip mismatch: got %+v, want %+v", got, m)
		}
	})

	t.Run("Member", func(t *testing.T) {
		m := MemberModel{
			id:        4004,
			worldId:   world.Id(5),
			channelId: channel.Id(6),
		}

		rm := TransformMember(m)

		got, err := ExtractMember(rm)
		if err != nil {
			t.Fatalf("ExtractMember returned error: %v", err)
		}

		if !reflect.DeepEqual(got, m) {
			t.Fatalf("round trip mismatch: got %+v, want %+v", got, m)
		}
	})
}
