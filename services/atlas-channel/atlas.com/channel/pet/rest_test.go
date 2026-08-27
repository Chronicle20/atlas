package pet

import (
	"atlas-channel/pet/exclude"
	"reflect"
	"testing"
	"time"
)

// TestTransformRoundTrip confirms Transform is the faithful inverse of
// Extract: every field set by Extract survives a Transform -> Extract round
// trip.
//
// Lead is not asserted: Model has no lead field (Model.Lead() derives
// slot == 0, pet/model.go:77-79), and Extract never reads RestModel.Lead
// (pet/rest.go:49-74), so Transform cannot restore it from Model and does
// not attempt to. Recorded in handwork-notes.md under "Batch channel-d".
func TestTransformRoundTrip(t *testing.T) {
	rm := RestModel{
		Id:         1,
		CashId:     2,
		TemplateId: 3,
		Name:       "Fido",
		Level:      4,
		Closeness:  5,
		Fullness:   6,
		Expiration: time.Date(2026, 3, 4, 5, 6, 7, 0, time.UTC),
		OwnerId:    7,
		Slot:       1,
		X:          8,
		Y:          9,
		Stance:     10,
		FH:         11,
		Excludes: []exclude.RestModel{
			{Id: 20, ItemId: 21},
			{Id: 22, ItemId: 23},
		},
		Flag:       12,
		PurchaseBy: 13,
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
