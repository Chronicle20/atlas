package minigame

import (
	"reflect"
	"testing"
)

// TestTransformRoundTrip confirms Transform is the faithful inverse of
// Extract: every field set by Extract survives a Transform -> Extract round
// trip, including the three bool fields (Private, HasPassword, InProgress)
// that the codemod refused to pair automatically.
func TestTransformRoundTrip(t *testing.T) {
	rm := RestModel{
		Id:          1,
		OwnerId:     2,
		RoomType:    1,
		Title:       "Room",
		Private:     true,
		HasPassword: false,
		PieceType:   2,
		Occupancy:   4,
		InProgress:  true,
	}

	m, err := Extract(rm)
	if err != nil {
		t.Fatalf("Extract failed: %v", err)
	}

	rm2, err := Transform(m)
	if err != nil {
		t.Fatalf("Transform failed: %v", err)
	}

	if rm2.Id != rm.Id {
		t.Errorf("Id mismatch. Expected %d, got %d", rm.Id, rm2.Id)
	}

	m2, err := Extract(rm2)
	if err != nil {
		t.Fatalf("Extract (second pass) failed: %v", err)
	}

	if !reflect.DeepEqual(m, m2) {
		t.Errorf("round trip mismatch. Expected %+v, got %+v", m, m2)
	}
}
