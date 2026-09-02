package inventory

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	"github.com/jtumidanski/api2go/jsonapi"
	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-constants/inventory"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

func testCtx(t *testing.T) context.Context {
	t.Helper()
	tm, err := tenant.Register(uuid.New(), "GMS", 83, 1)
	if err != nil {
		t.Fatalf("tenant: %v", err)
	}
	return tenant.WithContext(context.Background(), tm)
}

// TestRestModel_Decode asserts the JSON:API decode of a captured
// characters/{id}/inventory payload: the inventory -> compartments ->
// assets include chain decodes into the equip compartment's assets.
func TestRestModel_Decode(t *testing.T) {
	payload := []byte(`{
		"data": {
			"type": "inventories",
			"id": "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa",
			"attributes": {"characterId": 1001},
			"relationships": {
				"compartments": {
					"data": [
						{"type": "compartments", "id": "bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb"}
					]
				}
			}
		},
		"included": [
			{
				"type": "compartments",
				"id": "bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb",
				"attributes": {"type": 1},
				"relationships": {
					"assets": {
						"data": [
							{"type": "assets", "id": "1"},
							{"type": "assets", "id": "2"}
						]
					}
				}
			},
			{"type": "assets", "id": "1", "attributes": {"slot": -1, "templateId": 1302000}},
			{"type": "assets", "id": "2", "attributes": {"slot": -101, "templateId": 1040002}}
		]
	}`)

	var rm RestModel
	if err := jsonapi.Unmarshal(payload, &rm); err != nil {
		t.Fatalf("Unmarshal returned error: %v", err)
	}

	m, err := Extract(rm)
	if err != nil {
		t.Fatalf("Extract returned error: %v", err)
	}

	if m.CharacterId() != 1001 {
		t.Errorf("CharacterId() = %d, want 1001", m.CharacterId())
	}

	equip := m.Equipment()
	if len(equip) != 2 {
		t.Fatalf("len(Equipment()) = %d, want 2", len(equip))
	}

	bySlot := make(map[int16]uint32, len(equip))
	for _, a := range equip {
		bySlot[a.Slot()] = a.TemplateId()
	}
	if bySlot[-1] != 1302000 {
		t.Errorf("bySlot[-1] = %d, want 1302000", bySlot[-1])
	}
	if bySlot[-101] != 1040002 {
		t.Errorf("bySlot[-101] = %d, want 1040002", bySlot[-101])
	}
}

// TestRestModel_Decode_NonEquipCompartmentDropped asserts a non-equip
// compartment (e.g. use) does not surface through Equipment().
func TestRestModel_Decode_NonEquipCompartmentDropped(t *testing.T) {
	payload := []byte(`{
		"data": {
			"type": "inventories",
			"id": "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa",
			"attributes": {"characterId": 1001},
			"relationships": {
				"compartments": {
					"data": [
						{"type": "compartments", "id": "cccccccc-cccc-cccc-cccc-cccccccccccc"}
					]
				}
			}
		},
		"included": [
			{
				"type": "compartments",
				"id": "cccccccc-cccc-cccc-cccc-cccccccccccc",
				"attributes": {"type": 2},
				"relationships": {
					"assets": {
						"data": [
							{"type": "assets", "id": "3"}
						]
					}
				}
			},
			{"type": "assets", "id": "3", "attributes": {"slot": 1, "templateId": 2000000}}
		]
	}`)

	var rm RestModel
	if err := jsonapi.Unmarshal(payload, &rm); err != nil {
		t.Fatalf("Unmarshal returned error: %v", err)
	}

	m, err := Extract(rm)
	if err != nil {
		t.Fatalf("Extract returned error: %v", err)
	}

	if len(m.Equipment()) != 0 {
		t.Errorf("len(Equipment()) = %d, want 0", len(m.Equipment()))
	}
	if got, want := inventory.TypeValueEquip, inventory.Type(1); got != want {
		t.Fatalf("test setup: TypeValueEquip = %d, want %d", got, want)
	}
}

// TestGetByCharacterId_HttpDecode exercises the real HTTP round-trip: a
// httptest.NewServer serves a captured characters/{id}/inventory JSON:API
// fixture (with its included compartments/assets chain) and
// Processor.GetByCharacterId decodes it through the actual
// requests.GetRequest path, not a hand-built RestModel.
func TestGetByCharacterId_HttpDecode(t *testing.T) {
	wantPath := "/characters/1001/inventory"
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != wantPath {
			t.Errorf("unexpected path %s, want %s", r.URL.Path, wantPath)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"data": {
				"type": "inventories",
				"id": "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa",
				"attributes": {"characterId": 1001},
				"relationships": {
					"compartments": {
						"data": [{"type": "compartments", "id": "bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb"}]
					}
				}
			},
			"included": [
				{
					"type": "compartments",
					"id": "bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb",
					"attributes": {"type": 1},
					"relationships": {
						"assets": {"data": [{"type": "assets", "id": "1"}]}
					}
				},
				{"type": "assets", "id": "1", "attributes": {"slot": -1, "templateId": 1302000}}
			]
		}`))
	}))
	defer srv.Close()
	t.Setenv("INVENTORY_SERVICE_URL", srv.URL+"/")

	m, err := NewProcessor(logrus.New(), testCtx(t)).GetByCharacterId(1001)
	if err != nil {
		t.Fatalf("GetByCharacterId returned error: %v", err)
	}
	if m.CharacterId() != 1001 {
		t.Errorf("CharacterId() = %d, want 1001", m.CharacterId())
	}
	equip := m.Equipment()
	if len(equip) != 1 || equip[0].Slot() != -1 || equip[0].TemplateId() != 1302000 {
		t.Errorf("Equipment() = %+v, want one slot=-1 templateId=1302000 entry", equip)
	}
}
