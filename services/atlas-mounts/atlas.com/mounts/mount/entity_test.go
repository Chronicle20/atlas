package mount

import (
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestMakeRoundTrips(t *testing.T) {
	tenantId := uuid.New()
	id := uuid.New()
	tick := time.Now().UTC().Truncate(time.Second)

	e := Entity{
		TenantId:            tenantId,
		CharacterId:         42,
		Id:                  id,
		Level:               5,
		Exp:                 120,
		Tiredness:           33,
		LastTirednessTickAt: &tick,
	}

	m, err := Make(e)
	if err != nil {
		t.Fatalf("Make returned error: %v", err)
	}

	if m.TenantId() != tenantId {
		t.Errorf("TenantId = %v, want %v", m.TenantId(), tenantId)
	}
	if m.CharacterId() != 42 {
		t.Errorf("CharacterId = %d, want 42", m.CharacterId())
	}
	if m.Id() != id {
		t.Errorf("Id = %v, want %v", m.Id(), id)
	}
	if m.Level() != 5 {
		t.Errorf("Level = %d, want 5", m.Level())
	}
	if m.Exp() != 120 {
		t.Errorf("Exp = %d, want 120", m.Exp())
	}
	if m.Tiredness() != 33 {
		t.Errorf("Tiredness = %d, want 33", m.Tiredness())
	}
	if m.LastTirednessTickAt() == nil || !m.LastTirednessTickAt().Equal(tick) {
		t.Errorf("LastTirednessTickAt = %v, want %v", m.LastTirednessTickAt(), tick)
	}
}

func TestMakeNilTick(t *testing.T) {
	e := Entity{
		TenantId:    uuid.New(),
		CharacterId: 7,
		Id:          uuid.New(),
		Level:       1,
	}

	m, err := Make(e)
	if err != nil {
		t.Fatalf("Make returned error: %v", err)
	}
	if m.LastTirednessTickAt() != nil {
		t.Errorf("LastTirednessTickAt = %v, want nil", m.LastTirednessTickAt())
	}
}
