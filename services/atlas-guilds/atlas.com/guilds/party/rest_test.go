package party

import (
	"reflect"
	"testing"

	"github.com/google/uuid"

	"github.com/Chronicle20/atlas/libs/atlas-constants/channel"
	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	"github.com/Chronicle20/atlas/libs/atlas-constants/job"
	_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"
	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
)

func TestTransformRoundTrip(t *testing.T) {
	t.Run("Member", func(t *testing.T) {
		instance := uuid.New()
		mm := MemberModel{
			id:     42,
			name:   "Bishop",
			level:  120,
			jobId:  job.Id(2000),
			field:  field.NewBuilder(world.Id(1), channel.Id(2), _map.Id(100000000)).SetInstance(instance).Build(),
			online: true,
		}

		rm, err := TransformMember(mm)
		if err != nil {
			t.Fatalf("TransformMember failed: %v", err)
		}

		result, err := ExtractMember(rm)
		if err != nil {
			t.Fatalf("ExtractMember failed: %v", err)
		}

		if !reflect.DeepEqual(result, mm) {
			t.Errorf("round trip mismatch. Expected %+v, got %+v", mm, result)
		}
	})

	t.Run("Party", func(t *testing.T) {
		instance := uuid.New()
		mm := MemberModel{
			id:     42,
			name:   "Bishop",
			level:  120,
			jobId:  job.Id(2000),
			field:  field.NewBuilder(world.Id(1), channel.Id(2), _map.Id(100000000)).SetInstance(instance).Build(),
			online: true,
		}
		m := Model{
			id:       7,
			leaderId: 42,
			members:  []MemberModel{mm},
		}

		rm, err := Transform(m)
		if err != nil {
			t.Fatalf("Transform failed: %v", err)
		}

		result, err := Extract(rm)
		if err != nil {
			t.Fatalf("Extract failed: %v", err)
		}

		if !reflect.DeepEqual(result, m) {
			t.Errorf("round trip mismatch. Expected %+v, got %+v", m, result)
		}
	})
}
