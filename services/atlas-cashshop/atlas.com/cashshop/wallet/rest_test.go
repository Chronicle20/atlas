package wallet

import (
	"github.com/google/uuid"
	"testing"
)

func TestTransform(t *testing.T) {
	// Create a wallet model using direct struct creation (within package)
	id := uuid.New()
	accountId := uint32(12345)
	credit := uint32(1000)
	points := uint32(500)
	prepaid := uint32(250)

	m := Model{
		id:        id,
		accountId: accountId,
		credit:    credit,
		points:    points,
		prepaid:   prepaid,
	}

	// Transform to REST model
	rm, err := Transform(m)
	if err != nil {
		t.Fatalf("Transform failed: %v", err)
	}

	// Verify all fields are correctly transformed
	if rm.Id != id {
		t.Errorf("Id mismatch: expected %v, got %v", id, rm.Id)
	}
	if rm.AccountId != accountId {
		t.Errorf("AccountId mismatch: expected %d, got %d", accountId, rm.AccountId)
	}
	if rm.Credit != credit {
		t.Errorf("Credit mismatch: expected %d, got %d", credit, rm.Credit)
	}
	if rm.Points != points {
		t.Errorf("Points mismatch: expected %d, got %d", points, rm.Points)
	}
	if rm.Prepaid != prepaid {
		t.Errorf("Prepaid mismatch: expected %d, got %d", prepaid, rm.Prepaid)
	}
}

func TestExtract(t *testing.T) {
	// Create a REST model
	id := uuid.New()
	accountId := uint32(12345)
	credit := uint32(1000)
	points := uint32(500)
	prepaid := uint32(250)

	rm := RestModel{
		Id:        id,
		AccountId: accountId,
		Credit:    credit,
		Points:    points,
		Prepaid:   prepaid,
	}

	// Extract to domain model
	m, err := Extract(rm)
	if err != nil {
		t.Fatalf("Extract failed: %v", err)
	}

	// Verify all fields are correctly extracted
	if m.Id() != id {
		t.Errorf("Id mismatch: expected %v, got %v", id, m.Id())
	}
	if m.AccountId() != accountId {
		t.Errorf("AccountId mismatch: expected %d, got %d", accountId, m.AccountId())
	}
	if m.Credit() != credit {
		t.Errorf("Credit mismatch: expected %d, got %d", credit, m.Credit())
	}
	if m.Points() != points {
		t.Errorf("Points mismatch: expected %d, got %d", points, m.Points())
	}
	if m.Prepaid() != prepaid {
		t.Errorf("Prepaid mismatch: expected %d, got %d", prepaid, m.Prepaid())
	}
}

func TestTransformExtractRoundTrip(t *testing.T) {
	// Create a wallet model
	id := uuid.New()
	accountId := uint32(12345)
	credit := uint32(1000)
	points := uint32(500)
	prepaid := uint32(250)

	original := Model{
		id:        id,
		accountId: accountId,
		credit:    credit,
		points:    points,
		prepaid:   prepaid,
	}

	// Transform to REST model
	rm, err := Transform(original)
	if err != nil {
		t.Fatalf("Transform failed: %v", err)
	}

	// Extract back to domain model
	result, err := Extract(rm)
	if err != nil {
		t.Fatalf("Extract failed: %v", err)
	}

	// Verify round-trip preserves all values
	if original.Id() != result.Id() {
		t.Errorf("Id mismatch after round-trip: expected %v, got %v", original.Id(), result.Id())
	}
	if original.AccountId() != result.AccountId() {
		t.Errorf("AccountId mismatch after round-trip: expected %d, got %d", original.AccountId(), result.AccountId())
	}
	if original.Credit() != result.Credit() {
		t.Errorf("Credit mismatch after round-trip: expected %d, got %d", original.Credit(), result.Credit())
	}
	if original.Points() != result.Points() {
		t.Errorf("Points mismatch after round-trip: expected %d, got %d", original.Points(), result.Points())
	}
	if original.Prepaid() != result.Prepaid() {
		t.Errorf("Prepaid mismatch after round-trip: expected %d, got %d", original.Prepaid(), result.Prepaid())
	}
}

func TestRestModelGetName(t *testing.T) {
	rm := RestModel{}
	expected := "wallets"
	if rm.GetName() != expected {
		t.Errorf("GetName mismatch: expected %s, got %s", expected, rm.GetName())
	}
}

func TestRestModelGetID(t *testing.T) {
	id := uuid.New()
	rm := RestModel{Id: id}
	expected := id.String()
	if rm.GetID() != expected {
		t.Errorf("GetID mismatch: expected %s, got %s", expected, rm.GetID())
	}
}

func TestRestModelSetID(t *testing.T) {
	rm := &RestModel{}
	id := uuid.New()
	err := rm.SetID(id.String())
	if err != nil {
		t.Fatalf("SetID failed: %v", err)
	}
	if rm.Id != id {
		t.Errorf("SetID mismatch: expected %v, got %v", id, rm.Id)
	}
}

func TestRestModelSetIDInvalid(t *testing.T) {
	rm := &RestModel{}
	err := rm.SetID("not-a-valid-uuid")
	if err == nil {
		t.Error("SetID should fail for invalid UUID")
	}
}
