package definition

import (
	"encoding/json"
	"testing"

	"github.com/google/uuid"
)

func TestModelRoundTripsThroughEntity(t *testing.T) {
	tenantId := uuid.New()
	cfg := json.RawMessage(`{"monsterCount":2}`)

	m, err := NewBuilder("CRIMSON_BALROG", "Crimson Balrog").
		SetId(uuid.New()).
		SetEnabled(false).
		SetConfiguration(cfg).
		Build()
	if err != nil {
		t.Fatalf("Build: %v", err)
	}

	e, err := ToEntity(m, tenantId)
	if err != nil {
		t.Fatalf("ToEntity: %v", err)
	}
	if e.TenantID != tenantId {
		t.Fatalf("tenant not stamped")
	}

	back, err := Make(e)
	if err != nil {
		t.Fatalf("Make: %v", err)
	}
	if back.Id() != m.Id() || back.Type() != "CRIMSON_BALROG" || back.Enabled() {
		t.Fatalf("round trip lost fields: %+v", back)
	}
	if string(back.Configuration()) != string(cfg) {
		t.Fatalf("configuration mangled: %s", back.Configuration())
	}
}

// FR-D2: configuration is opaque to this package. It must survive round trip
// byte-for-byte, including a shape this package has never seen.
func TestConfigurationIsOpaque(t *testing.T) {
	cfg := json.RawMessage(`{"unknownToUs":[1,2,{"deep":true}]}`)
	m, err := NewBuilder("SOMETHING_NEW", "n").SetConfiguration(cfg).Build()
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	e, err := ToEntity(m, uuid.New())
	if err != nil {
		t.Fatalf("ToEntity: %v", err)
	}
	back, err := Make(e)
	if err != nil {
		t.Fatalf("Make: %v", err)
	}
	if string(back.Configuration()) != string(cfg) {
		t.Fatalf("opaque configuration changed: %s", back.Configuration())
	}
}

func TestBuildRejectsEmptyType(t *testing.T) {
	if _, err := NewBuilder("", "n").Build(); err == nil {
		t.Fatalf("expected an error for an empty type")
	}
}
