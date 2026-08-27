package door

import (
	"reflect"
	"testing"
	"time"

	"github.com/google/uuid"
)

// TestTransformRoundTrip confirms Transform is the faithful inverse of
// Extract: every field set by Extract survives a Transform -> Extract round
// trip.
func TestTransformRoundTrip(t *testing.T) {
	rm := RestModel{
		Id:               "1",
		AreaDoorId:       100,
		TownDoorId:       200,
		PairId:           300,
		OwnerCharacterId: 400,
		PartyId:          500,
		WorldId:          1,
		ChannelId:        2,
		MapId:            600,
		Instance:         uuid.New(),
		TownMapId:        700,
		Slot:             8,
		TownPortalId:     900,
		AreaX:            10,
		AreaY:            11,
		TownX:            12,
		TownY:            13,
		SkillId:          1000,
		SkillLevel:       14,
		ExpiresAt:        time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC),
	}

	m, err := Extract(rm)
	if err != nil {
		t.Fatalf("Extract failed: %v", err)
	}

	rm2, err := Transform(m)
	if err != nil {
		t.Fatalf("Transform failed: %v", err)
	}

	m2, err := Extract(rm2)
	if err != nil {
		t.Fatalf("Extract (second pass) failed: %v", err)
	}

	if !reflect.DeepEqual(m, m2) {
		t.Errorf("round trip mismatch. Expected %+v, got %+v", m, m2)
	}
}
