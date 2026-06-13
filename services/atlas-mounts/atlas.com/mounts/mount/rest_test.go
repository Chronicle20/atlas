package mount

import (
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestRestModelGetName(t *testing.T) {
	if got := (RestModel{}).GetName(); got != "mounts" {
		t.Fatalf("GetName() = %q, want %q", got, "mounts")
	}
}

func TestTransformAllFields(t *testing.T) {
	tenantId := uuid.New()
	mountId := uuid.New()
	tick := time.Date(2026, time.June, 12, 10, 0, 0, 0, time.UTC)

	m, err := NewModelBuilder(tenantId, 12345, mountId).
		SetLevel(7).
		SetExp(420).
		SetTiredness(33).
		SetLastTirednessTickAt(&tick).
		Build()
	if err != nil {
		t.Fatalf("building model: %v", err)
	}

	rm, err := Transform(m)
	if err != nil {
		t.Fatalf("Transform returned error: %v", err)
	}

	if rm.Id != mountId.String() {
		t.Errorf("Id = %q, want %q", rm.Id, mountId.String())
	}
	if rm.GetID() != mountId.String() {
		t.Errorf("GetID() = %q, want %q", rm.GetID(), mountId.String())
	}
	if rm.CharacterId != 12345 {
		t.Errorf("CharacterId = %d, want %d", rm.CharacterId, 12345)
	}
	if rm.Level != 7 {
		t.Errorf("Level = %d, want %d", rm.Level, 7)
	}
	if rm.Exp != 420 {
		t.Errorf("Exp = %d, want %d", rm.Exp, 420)
	}
	if rm.Tiredness != 33 {
		t.Errorf("Tiredness = %d, want %d", rm.Tiredness, 33)
	}
	if rm.LastTirednessTickAt == nil {
		t.Fatalf("LastTirednessTickAt = nil, want %v", tick)
	}
	if !rm.LastTirednessTickAt.Equal(tick) {
		t.Errorf("LastTirednessTickAt = %v, want %v", *rm.LastTirednessTickAt, tick)
	}
}

func TestTransformNilTick(t *testing.T) {
	m, err := NewModelBuilder(uuid.New(), 1, uuid.New()).Build()
	if err != nil {
		t.Fatalf("building model: %v", err)
	}

	rm, err := Transform(m)
	if err != nil {
		t.Fatalf("Transform returned error: %v", err)
	}

	if rm.LastTirednessTickAt != nil {
		t.Errorf("LastTirednessTickAt = %v, want nil", *rm.LastTirednessTickAt)
	}
	// defaults from the builder
	if rm.Level != 1 || rm.Exp != 0 || rm.Tiredness != 0 {
		t.Errorf("defaults wrong: level=%d exp=%d tiredness=%d", rm.Level, rm.Exp, rm.Tiredness)
	}
}
