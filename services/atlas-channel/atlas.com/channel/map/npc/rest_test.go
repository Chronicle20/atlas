package npc

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	"github.com/jtumidanski/api2go/jsonapi"
	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

func newTestTenant(t *testing.T) tenant.Model {
	t.Helper()
	tm, err := tenant.Create(uuid.New(), "GMS", 83, 1)
	if err != nil {
		t.Fatalf("tenant: %v", err)
	}
	return tm
}

// TestRestModel_Unmarshal asserts a scripted NPC's attributes round-trip
// through a JSON:API document decode, mirroring atlas-maps' own
// map/npc.RestModel field-for-field.
func TestRestModel_Unmarshal(t *testing.T) {
	body := []byte(`{
		"data": {
			"type": "npcs",
			"id": "12345",
			"attributes": {
				"npcId": 9010000,
				"x": 250,
				"y": 300,
				"fh": 12
			}
		}
	}`)

	var rm RestModel
	if err := jsonapi.Unmarshal(body, &rm); err != nil {
		t.Fatalf("jsonapi.Unmarshal: %v", err)
	}
	if rm.Id != "12345" {
		t.Errorf("Id = %q, want %q", rm.Id, "12345")
	}
	if rm.NpcId != 9010000 || rm.X != 250 || rm.Y != 300 || rm.Fh != 12 {
		t.Fatalf("attributes mismatch: %+v", rm)
	}

	m, err := Extract(rm)
	if err != nil {
		t.Fatalf("Extract: %v", err)
	}
	if m.UniqueId() != 12345 || m.NpcId() != 9010000 || m.X() != 250 || m.Y() != 300 || m.Fh() != 12 {
		t.Fatalf("Extract round-trip mismatch: %+v", m)
	}
}

// TestExtract_InvalidId asserts Extract surfaces a non-numeric Id as an
// error instead of silently coercing it to 0 -- atlas-maps' own uniqueId
// is always numeric, so a decode failure here means the two services have
// drifted.
func TestExtract_InvalidId(t *testing.T) {
	if _, err := Extract(RestModel{Id: "not-a-number"}); err == nil {
		t.Fatal("Extract: expected an error for a non-numeric Id, got nil")
	}
}

// TestBuilder_RoundTrip asserts the Builder produces a Model whose
// accessors reflect exactly the values set on it.
func TestBuilder_RoundTrip(t *testing.T) {
	m := NewBuilder(12345, 9010000).SetPosition(250, 300, 12).Build()
	if m.UniqueId() != 12345 || m.NpcId() != 9010000 || m.X() != 250 || m.Y() != 300 || m.Fh() != 12 {
		t.Fatalf("Build mismatch: %+v", m)
	}
}

// TestForEachInMap_RequestsInstanceScopedPath asserts InMapModelProvider
// (via ForEachInMap) requests atlas-maps' instance-scoped path template
// (requests.go's Resource) against the field's world/channel/map/instance,
// and that a character entering a field with an already-placed scripted
// NPC receives it -- the regression task-BC2 exists to prevent.
func TestForEachInMap_RequestsInstanceScopedPath(t *testing.T) {
	inst := uuid.MustParse("11111111-2222-3333-4444-555555555555")
	var seen string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		seen = r.URL.Path
		w.Header().Set("Content-Type", "application/vnd.api+json")
		_, _ = w.Write([]byte(`{
			"data": [
				{
					"type": "npcs",
					"id": "12345",
					"attributes": {
						"npcId": 9010000,
						"x": 250,
						"y": 300,
						"fh": 12
					}
				}
			]
		}`))
	}))
	defer srv.Close()
	defer SetBaseURLForTest(srv.URL)()

	tm := newTestTenant(t)
	ctx := tenant.WithContext(context.Background(), tm)

	f := field.NewBuilder(0, 1, 108010600).SetInstance(inst).Build()

	var seenNpcs []Model
	err := NewProcessor(logrus.New(), ctx).ForEachInMap(f, func(m Model) error {
		seenNpcs = append(seenNpcs, m)
		return nil
	})
	if err != nil {
		t.Fatalf("ForEachInMap: %v", err)
	}
	if len(seenNpcs) != 1 {
		t.Fatalf("seenNpcs = %d models, want 1", len(seenNpcs))
	}
	if seenNpcs[0].UniqueId() != 12345 || seenNpcs[0].NpcId() != 9010000 {
		t.Errorf("seenNpcs[0] = %+v, want uniqueId 12345 / npcId 9010000", seenNpcs[0])
	}

	want := fmt.Sprintf("/api/worlds/0/channels/1/maps/108010600/instances/%s/npcs", inst.String())
	if seen != want {
		t.Errorf("requested %q, want %q", seen, want)
	}
}
