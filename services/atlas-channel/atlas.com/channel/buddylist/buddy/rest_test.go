package buddy

import (
	"reflect"
	"testing"
)

// TestTransformRoundTrip confirms Transform is the faithful inverse of
// Extract: every field set by Extract survives a Transform -> Extract round
// trip.
func TestTransformRoundTrip(t *testing.T) {
	rm := RestModel{
		CharacterId:   1,
		Group:         "Default",
		CharacterName: "Bob",
		ChannelId:     2,
		InShop:        true,
		Pending:       false,
	}

	m, err := Extract(rm)
	if err != nil {
		t.Fatalf("Extract failed: %v", err)
	}

	rm2, err := Transform(m)
	if err != nil {
		t.Fatalf("Transform failed: %v", err)
	}

	if rm2.CharacterId != rm.CharacterId {
		t.Errorf("CharacterId mismatch. Expected %d, got %d", rm.CharacterId, rm2.CharacterId)
	}

	m2, err := Extract(rm2)
	if err != nil {
		t.Fatalf("Extract (second pass) failed: %v", err)
	}

	if !reflect.DeepEqual(m, m2) {
		t.Errorf("round trip mismatch. Expected %+v, got %+v", m, m2)
	}
}
