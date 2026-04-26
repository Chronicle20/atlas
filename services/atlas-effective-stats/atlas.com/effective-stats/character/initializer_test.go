package character

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"atlas-effective-stats/stat"

	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus/hooks/test"
)

// TestFetchEquipmentBonuses_HydratesEquipmentStats stubs atlas-inventory with
// a JSON:API document containing one equipped Medal-style asset (slot -49,
// templateId 1142107, hp 47, mp 50). It then calls fetchEquipmentBonuses and
// asserts the returned []stat.Bonus includes the expected equipment:42
// entries for TypeMaxHp and TypeMaxMp.
//
// This is the integration-level regression net for the bug where the
// CompartmentRestModel only kept asset IDs, leaving Slot==0 and starving
// the IsEquipped() gate.
func TestFetchEquipmentBonuses_HydratesEquipmentStats(t *testing.T) {
	const doc = `{
      "data": {
        "type": "compartments",
        "id": "00000000-0000-0000-0000-000000000001",
        "attributes": {"type": 1, "capacity": 24},
        "relationships": {
          "assets": { "data": [{"type": "assets", "id": "42"}] }
        }
      },
      "included": [{
        "type": "assets",
        "id": "42",
        "attributes": {
          "slot": -49,
          "templateId": 1142107,
          "expiration": "0001-01-01T00:00:00Z",
          "ownerId": 0,
          "strength": 0,
          "dexterity": 0,
          "intelligence": 0,
          "luck": 0,
          "hp": 47,
          "mp": 50,
          "weaponAttack": 0,
          "magicAttack": 0,
          "weaponDefense": 0,
          "magicDefense": 0,
          "accuracy": 0,
          "avoidability": 0,
          "hands": 0,
          "speed": 0,
          "jump": 0
        }
      }]
    }`

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/vnd.api+json")
		_, _ = w.Write([]byte(doc))
	}))
	defer srv.Close()

	// RootUrl("INVENTORY") reads INVENTORY_SERVICE_URL and concatenates the
	// path template directly. The trailing slash keeps the resulting URL valid
	// (httptest.Server.URL has no trailing slash by default).
	t.Setenv("INVENTORY_SERVICE_URL", srv.URL+"/")

	l, _ := test.NewNullLogger()
	ten, err := tenant.Create(uuid.New(), "GMS", 83, 1)
	if err != nil {
		t.Fatalf("tenant.Create() error = %v", err)
	}
	ctx := tenant.WithContext(context.Background(), ten)

	bonuses, err := fetchEquipmentBonuses(l, ctx, 12)
	if err != nil {
		t.Fatalf("fetchEquipmentBonuses() error = %v", err)
	}

	var sawHp, sawMp bool
	for _, b := range bonuses {
		if b.Source() != "equipment:42" {
			t.Errorf("bonus source = %q, want %q", b.Source(), "equipment:42")
		}
		if b.StatType() == stat.TypeMaxHp && b.Amount() == 47 {
			sawHp = true
		}
		if b.StatType() == stat.TypeMaxMp && b.Amount() == 50 {
			sawMp = true
		}
	}
	if !sawHp {
		t.Errorf("missing TypeMaxHp+47 bonus; got %d bonuses: %#v", len(bonuses), bonuses)
	}
	if !sawMp {
		t.Errorf("missing TypeMaxMp+50 bonus; got %d bonuses: %#v", len(bonuses), bonuses)
	}
}
