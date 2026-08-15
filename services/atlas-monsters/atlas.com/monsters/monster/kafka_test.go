package monster

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
)

func TestStatusEventEnvelopeEchoesProvenance(t *testing.T) {
	f := field.NewBuilder(1, 4, 200090010).Build()
	m := NewMonster(f, 7, 8150000, 0, 0, 0, 5, 0, 100, 0, SpawnSourceTypeEvent, "occ-1")

	e := statusEventFromField(m.Field(), m.UniqueId(), m.MonsterId(), EventMonsterStatusKilled,
		statusEventKilledBody{}, m.SpawnSourceType(), m.SpawnSourceId())

	b, err := json.Marshal(e)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if !strings.Contains(string(b), `"spawnSourceType":"EVENT"`) ||
		!strings.Contains(string(b), `"spawnSourceId":"occ-1"`) {
		t.Fatalf("provenance not echoed: %s", b)
	}
}

func TestStatusEventEnvelopeOmitsEmptyProvenance(t *testing.T) {
	f := field.NewBuilder(1, 4, 200090010).Build()
	e := statusEventFromField(f, 7, 8150000, EventMonsterStatusCreated,
		statusEventCreatedBody{}, "", "")

	b, err := json.Marshal(e)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if strings.Contains(string(b), "spawnSource") {
		t.Fatalf("expected no provenance keys, got: %s", b)
	}
}
