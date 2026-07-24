package teleport_rock

import (
	"net/http"
	"testing"

	_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"
)

func TestTransform(t *testing.T) {
	m := NewBuilder().
		SetCharacterId(42).
		SetRegular([]_map.Id{100000000}).
		SetVip([]_map.Id{104040000, 220000000}).
		Build()
	rm, err := Transform(m)
	if err != nil {
		t.Fatalf("Transform: %v", err)
	}
	if rm.GetName() != "teleport-rock-maps" {
		t.Fatalf("resource name: %s", rm.GetName())
	}
	if rm.GetID() != "42" {
		t.Fatalf("id: %s", rm.GetID())
	}
	if len(rm.Regular) != 1 || len(rm.Vip) != 2 {
		t.Fatalf("lists: %+v", rm)
	}
}

func TestTransformIncludesCapacities(t *testing.T) {
	m := NewBuilder().SetCharacterId(42).SetRegular([]_map.Id{100000000}).Build()
	rm, err := Transform(m)
	if err != nil {
		t.Fatalf("Transform: %v", err)
	}
	if rm.RegularCapacity != RegularCapacity || rm.VipCapacity != VipCapacity {
		t.Fatalf("capacities = %d/%d, want %d/%d", rm.RegularCapacity, rm.VipCapacity, RegularCapacity, VipCapacity)
	}
}

func TestStatusForError(t *testing.T) {
	cases := map[error]int{
		ErrMapNotAllowed: http.StatusBadRequest,
		ErrListFull:      http.StatusConflict,
		ErrDuplicate:     http.StatusConflict,
		ErrNotFound:      http.StatusNotFound,
	}
	for err, want := range cases {
		if got := statusForError(err); got != want {
			t.Errorf("statusForError(%v) = %d, want %d", err, got, want)
		}
	}
}
