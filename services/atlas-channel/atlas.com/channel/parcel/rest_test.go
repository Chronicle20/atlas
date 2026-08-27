package parcel

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
	itemId := uint32(555)
	lastNotified := time.Date(2026, 2, 3, 4, 5, 6, 0, time.UTC)
	rm := RestModel{
		Id:                 uuid.New().String(),
		WorldId:            1,
		SenderId:           100,
		SenderAccountId:    200,
		SenderName:         "Sender",
		RecipientId:        300,
		RecipientAccountId: 400,
		Message:            "hello",
		MesoAmount:         5000,
		FeePaid:            250,
		ItemId:             &itemId,
		ItemType:           2,
		Quantity:           10,
		Status:             "PENDING",
		Quick:              true,
		Returned:           false,
		CreatedAt:          time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		ReceivableAt:       time.Date(2026, 1, 2, 0, 0, 0, 0, time.UTC),
		ExpiresAt:          time.Date(2026, 1, 22, 0, 0, 0, 0, time.UTC),
		LastNotified:       &lastNotified,
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
