package trade

import (
	"reflect"
	"testing"

	"github.com/google/uuid"
)

// TestTransformRoundTrip confirms Transform is the faithful inverse of
// Extract: the room id set by Extract survives a Transform -> Extract round
// trip.
func TestTransformRoundTrip(t *testing.T) {
	rm := RestModel{
		Id: uuid.New().String(),
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
