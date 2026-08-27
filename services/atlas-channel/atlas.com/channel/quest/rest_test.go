package quest

import (
	"reflect"
	"testing"
	"time"
)

// TestTransformRoundTrip confirms Transform is the faithful inverse of
// Extract: every field set by Extract survives a Transform -> Extract round
// trip.
func TestTransformRoundTrip(t *testing.T) {
	rm := RestModel{
		Id:             1,
		CharacterId:    2,
		QuestId:        3,
		State:          StateStarted,
		StartedAt:      time.Date(2026, 1, 1, 1, 1, 1, 0, time.UTC),
		CompletedAt:    time.Date(2026, 2, 2, 2, 2, 2, 0, time.UTC),
		ExpirationTime: time.Date(2026, 3, 3, 3, 3, 3, 0, time.UTC),
		CompletedCount: 4,
		ForfeitCount:   5,
		Progress: []ProgressRestModel{
			{Id: 6, InfoNumber: 7, Progress: "p1"},
			{Id: 8, InfoNumber: 9, Progress: "p2"},
		},
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
