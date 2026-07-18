package inventory

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-constants/item"
)

// equipCompartmentBody is a faithful JSON:API document for the atlas-inventory
// equip compartment endpoint (characters/{id}/inventory/compartments?type=1&include=assets).
// The single compartment carries an `assets` to-many relationship and the asset
// is denormalised under `included`, so decoding it exercises the
// SetReferencedStructs include-unmarshal path (the core of EXT-02). Without that
// path the asset Slot/TemplateId would stay zero and the weapon lookup would
// never match slot -11.
//
// The included asset sits at slot -11 (the main weapon equip slot) and carries
// templateId 1302000 — a one-handed sword: (1302000/10000)%100 == 30, which
// item.GetWeaponType classifies as WeaponTypeOneHandedSword. A silent unmarshal
// bug here would leave the asset at templateId 0 and corrupt the damage ceiling.
const equipCompartmentBody = `{
  "data": {
    "type": "compartments",
    "id": "00000000-0000-0000-0000-000000000001",
    "attributes": {"type": 1, "capacity": 24},
    "relationships": {
      "assets": { "data": [{"type": "assets", "id": "7"}] }
    }
  },
  "included": [{
    "type": "assets",
    "id": "7",
    "attributes": {
      "slot": -11,
      "templateId": 1302000
    }
  }]
}`

// TestGetEquippedWeaponType_ClassifiesIncludedWeapon spins up an httptest server
// returning the equip compartment document above, points the INVENTORY client at
// it via INVENTORY_SERVICE_URL, and asserts the equipped weapon (slot -11) is
// decoded from the `included` array and classified to the expected WeaponType.
func TestGetEquippedWeaponType_ClassifiesIncludedWeapon(t *testing.T) {
	var gotPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path + "?" + r.URL.RawQuery
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(equipCompartmentBody))
	}))
	defer srv.Close()

	// RootUrl("INVENTORY") reads INVENTORY_SERVICE_URL per call; trailing slash
	// so the resource path appends cleanly. t.Setenv auto-restores afterward.
	t.Setenv("INVENTORY_SERVICE_URL", srv.URL+"/")

	p := NewProcessor(logrus.New(), context.Background())

	const characterId uint32 = 12345
	wt, err := p.GetEquippedWeaponType(characterId)
	if err != nil {
		t.Fatalf("GetEquippedWeaponType() error = %v", err)
	}

	if wt != item.WeaponTypeOneHandedSword {
		t.Errorf("WeaponType = %d, want %d (OneHandedSword)", wt, item.WeaponTypeOneHandedSword)
	}

	// Confirm the client built the equip-compartment path (type=1, include=assets).
	wantPath := "/characters/12345/inventory/compartments?type=1&include=assets"
	if gotPath != wantPath {
		t.Errorf("request path = %q, want %q", gotPath, wantPath)
	}
}

// TestGetEquippedWeaponType_NoWeaponEquipped asserts the WeaponTypeNone fallback
// when the compartment has no asset in the weapon slot.
func TestGetEquippedWeaponType_NoWeaponEquipped(t *testing.T) {
	const body = `{
      "data": {
        "type": "compartments",
        "id": "00000000-0000-0000-0000-000000000002",
        "attributes": {"type": 1, "capacity": 24},
        "relationships": {
          "assets": { "data": [{"type": "assets", "id": "9"}] }
        }
      },
      "included": [{
        "type": "assets",
        "id": "9",
        "attributes": {"slot": -1, "templateId": 1002000}
      }]
    }`

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(body))
	}))
	defer srv.Close()

	t.Setenv("INVENTORY_SERVICE_URL", srv.URL+"/")

	p := NewProcessor(logrus.New(), context.Background())

	wt, err := p.GetEquippedWeaponType(777)
	if err != nil {
		t.Fatalf("GetEquippedWeaponType() error = %v", err)
	}
	if wt != item.WeaponTypeNone {
		t.Errorf("WeaponType = %d, want %d (None)", wt, item.WeaponTypeNone)
	}
}
