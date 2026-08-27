package wish

import (
	"reflect"
	"testing"
	"time"
)

// TestTransformRoundTrip confirms Transform is the faithful inverse of
// Extract: every Model field set by Extract survives a Transform -> Extract
// round trip. ExpiresAt is a *time.Time on both sides (the codemod's SKIP
// reason); both the nil (never expires) and non-nil cases are covered, and
// the non-nil case confirms Transform copies the pointed-to value rather
// than aliasing the Model's pointer. RestModel.CreatedAt has no Model
// counterpart, so it is not part of the round trip and is asserted to come
// back as its zero value.
func TestTransformRoundTrip(t *testing.T) {
	base := RestModel{
		Id:            "1",
		WorldId:       0,
		Serial:        1001,
		CharacterId:   2002,
		ItemId:        4001000,
		ListingSerial: 3003,
		Price:         100000,
		Count:         1,
	}

	t.Run("nil ExpiresAt", func(t *testing.T) {
		rm := base
		rm.ExpiresAt = nil

		m, err := Extract(rm)
		if err != nil {
			t.Fatalf("Extract failed: %v", err)
		}

		rm2, err := Transform(m)
		if err != nil {
			t.Fatalf("Transform failed: %v", err)
		}

		if rm2.Id != rm.Id {
			t.Errorf("Id mismatch. Expected %s, got %s", rm.Id, rm2.Id)
		}
		if rm2.ExpiresAt != nil {
			t.Errorf("expected nil ExpiresAt to survive round trip, got %v", rm2.ExpiresAt)
		}
		if !rm2.CreatedAt.IsZero() {
			t.Errorf("expected CreatedAt to be zero value (no Model counterpart), got %v", rm2.CreatedAt)
		}

		m2, err := Extract(rm2)
		if err != nil {
			t.Fatalf("Extract (second pass) failed: %v", err)
		}
		if !reflect.DeepEqual(m, m2) {
			t.Errorf("round trip mismatch. Expected %+v, got %+v", m, m2)
		}
	})

	t.Run("non-nil ExpiresAt", func(t *testing.T) {
		expiresAt := time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)
		rm := base
		rm.ExpiresAt = &expiresAt

		m, err := Extract(rm)
		if err != nil {
			t.Fatalf("Extract failed: %v", err)
		}

		rm2, err := Transform(m)
		if err != nil {
			t.Fatalf("Transform failed: %v", err)
		}

		if rm2.Id != rm.Id {
			t.Errorf("Id mismatch. Expected %s, got %s", rm.Id, rm2.Id)
		}
		if rm2.ExpiresAt == nil || !rm2.ExpiresAt.Equal(expiresAt) {
			t.Errorf("expected ExpiresAt %v, got %v", expiresAt, rm2.ExpiresAt)
		}
		if rm2.ExpiresAt == rm.ExpiresAt {
			t.Errorf("expected Transform to copy ExpiresAt into a new pointer, not alias the source")
		}

		m2, err := Extract(rm2)
		if err != nil {
			t.Fatalf("Extract (second pass) failed: %v", err)
		}
		if !reflect.DeepEqual(m, m2) {
			t.Errorf("round trip mismatch. Expected %+v, got %+v", m, m2)
		}
	})
}
