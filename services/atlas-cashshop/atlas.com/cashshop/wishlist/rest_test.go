package wishlist

import (
	"testing"

	"github.com/google/uuid"
)

func TestTransform(t *testing.T) {
	// Create a wishlist model using direct struct creation (within package)
	id := uuid.New()
	characterId := uint32(12345)
	serialNumber := uint32(5000001)

	m := Model{
		id:           id,
		characterId:  characterId,
		serialNumber: serialNumber,
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
	if rm.CharacterId != characterId {
		t.Errorf("CharacterId mismatch: expected %d, got %d", characterId, rm.CharacterId)
	}
	if rm.SerialNumber != serialNumber {
		t.Errorf("SerialNumber mismatch: expected %d, got %d", serialNumber, rm.SerialNumber)
	}
}

func TestExtract(t *testing.T) {
	// Create a REST model
	id := uuid.New()
	characterId := uint32(12345)
	serialNumber := uint32(5000001)

	rm := RestModel{
		Id:           id,
		CharacterId:  characterId,
		SerialNumber: serialNumber,
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
	if m.CharacterId() != characterId {
		t.Errorf("CharacterId mismatch: expected %d, got %d", characterId, m.CharacterId())
	}
	if m.SerialNumber() != serialNumber {
		t.Errorf("SerialNumber mismatch: expected %d, got %d", serialNumber, m.SerialNumber())
	}
}

func TestTransformExtractRoundTrip(t *testing.T) {
	// Create a wishlist model
	id := uuid.New()
	characterId := uint32(12345)
	serialNumber := uint32(5000001)

	original := Model{
		id:           id,
		characterId:  characterId,
		serialNumber: serialNumber,
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
	if original.CharacterId() != result.CharacterId() {
		t.Errorf("CharacterId mismatch after round-trip: expected %d, got %d", original.CharacterId(), result.CharacterId())
	}
	if original.SerialNumber() != result.SerialNumber() {
		t.Errorf("SerialNumber mismatch after round-trip: expected %d, got %d", original.SerialNumber(), result.SerialNumber())
	}
}

func TestRestModelGetName(t *testing.T) {
	rm := RestModel{}
	expected := "items"
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
