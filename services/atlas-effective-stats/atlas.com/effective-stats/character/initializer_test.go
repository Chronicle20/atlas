package character

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"atlas-effective-stats/external/data/equipment"
	"atlas-effective-stats/stat"

	"github.com/Chronicle20/atlas/libs/atlas-constants/channel"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus/hooks/test"
)

// TestFetchEquippedSnapshots_HydratesEquipmentStats stubs atlas-inventory with
// a JSON:API document containing one equipped Medal-style asset (slot -49,
// templateId 1142107, hp 47, mp 50). It then calls fetchEquippedSnapshots and
// asserts the returned snapshots include the expected equipment:42 bonuses
// for TypeMaxHp and TypeMaxMp.
//
// This is the integration-level regression net for the bug where the
// CompartmentRestModel only kept asset IDs, leaving Slot==0 and starving
// the IsEquipped() gate.
func TestFetchEquippedSnapshots_HydratesEquipmentStats(t *testing.T) {
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

	snapshots, err := fetchEquippedSnapshots(l, ctx, 12)
	if err != nil {
		t.Fatalf("fetchEquippedSnapshots() error = %v", err)
	}
	if len(snapshots) != 1 {
		t.Fatalf("snapshots count = %d, want 1", len(snapshots))
	}
	if snapshots[0].AssetId() != 42 {
		t.Errorf("asset id = %d, want 42", snapshots[0].AssetId())
	}
	if snapshots[0].TemplateId() != 1142107 {
		t.Errorf("template id = %d, want 1142107", snapshots[0].TemplateId())
	}

	bonuses := snapshots[0].Bonuses()
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

// TestInitializeCharacter_DropsUnqualifiedOverall_Diagnosis is the
// integration-level reproduction of the PRD §4.1 case: a level-30 Magician
// with LUK 39 has equipped a Pole Arm (templateId 1052095) whose template
// requires LUK 40. The asset's MP +50 bonus must NOT show up in the final
// Computed.MaxMp() and MUST NOT appear in Bonuses() because the qualification
// gate dropped the asset.
func TestInitializeCharacter_DropsUnqualifiedOverall_Diagnosis(t *testing.T) {
	setupTestRegistry(t)
	t.Cleanup(equipment.ResetCacheForTest)
	l, ctx, _ := createTestContext()

	stubServers := startInitializerStubs(t, stubConfig{
		character: stubCharacter{
			level: 30, jobId: 200,
			str: 4, dex: 25, intl: 4, luk: 39,
			maxHp: 1430, maxMp: 6330,
		},
		equipped: []stubEquipped{{
			assetId: 42, templateId: 1052095, slot: -5, mp: 50,
		}},
		equipmentReqs: map[uint32]equipmentReqs{
			1052095: {reqLuk: 40},
		},
	})
	t.Cleanup(stubServers.Close)
	stubServers.PointEnv(t)

	if err := InitializeCharacter(l, ctx, 12345, channel.NewModel(0, 0)); err != nil {
		t.Fatalf("InitializeCharacter: %v", err)
	}
	m, err := GetRegistry().Get(ctx, 12345)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if m.Computed().MaxMp() != 6330 {
		t.Errorf("MaxMp = %d, want 6330 (overall +50 should be dropped)", m.Computed().MaxMp())
	}
	for _, b := range m.Bonuses() {
		if b.Source() == "equipment:42" {
			t.Errorf("Bonuses() unexpectedly contains equipment:42 entry: %+v", b)
		}
	}
}

// TestInitializeCharacter_KeepsQualifiedOverall is the positive companion to
// the diagnosis case: bumping the wearer's LUK from 39 to 40 satisfies
// reqLuk=40, so the +50 MP overall MUST be included in Computed.MaxMp().
func TestInitializeCharacter_KeepsQualifiedOverall(t *testing.T) {
	setupTestRegistry(t)
	t.Cleanup(equipment.ResetCacheForTest)
	l, ctx, _ := createTestContext()

	stubServers := startInitializerStubs(t, stubConfig{
		character: stubCharacter{
			level: 30, jobId: 200,
			str: 4, dex: 25, intl: 4, luk: 40,
			maxHp: 1430, maxMp: 6330,
		},
		equipped: []stubEquipped{{
			assetId: 42, templateId: 1052095, slot: -5, mp: 50,
		}},
		equipmentReqs: map[uint32]equipmentReqs{
			1052095: {reqLuk: 40},
		},
	})
	t.Cleanup(stubServers.Close)
	stubServers.PointEnv(t)

	if err := InitializeCharacter(l, ctx, 12345, channel.NewModel(0, 0)); err != nil {
		t.Fatalf("InitializeCharacter: %v", err)
	}
	m, _ := GetRegistry().Get(ctx, 12345)
	if m.Computed().MaxMp() != 6380 {
		t.Errorf("MaxMp = %d, want 6380", m.Computed().MaxMp())
	}
}
