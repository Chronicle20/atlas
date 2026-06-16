package character_test

import (
	"atlas-channel/asset"
	"atlas-channel/character"
	"atlas-channel/compartment"
	"atlas-channel/inventory"
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	inventory2 "github.com/Chronicle20/atlas/libs/atlas-constants/inventory"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
)

// TestPetAssetEnrichmentDecorator_PreservesEquipmentLook is a regression guard
// for the "naked character when a pet is in the cash compartment" bug.
//
// InventoryDecorator runs first and, via Model.SetInventory, extracts the worn
// (negative-slot) equips into the equipment LOOK while leaving the equipable
// compartment holding only positive-slot bag items. PetAssetEnrichmentDecorator
// must NOT re-run the look-rebuilding Model.SetInventory afterwards: doing so
// re-derives the look from the already-stripped equipable compartment, wiping
// every worn piece and rendering the avatar naked. The decorator only takes the
// look-clobbering path when a pet is present in the cash compartment, which is
// why this regression is latent until a character owns a pet.
func TestPetAssetEnrichmentDecorator_PreservesEquipmentLook(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/vnd.api+json")
		_, _ = w.Write([]byte(`{"data":[{"type":"pets","id":"1","attributes":{"templateId":5000029,"name":"Pet","level":1,"closeness":0,"fullness":79,"ownerId":1,"slot":-1}}]}`))
	}))
	defer srv.Close()
	t.Setenv("PETS_SERVICE_URL", srv.URL+"/api/")

	tm, _ := tenant.Create(uuid.New(), "GMS", 83, 1)
	ctx := tenant.WithContext(context.Background(), tm)
	p := character.NewProcessor(logrus.New(), ctx)

	const charId = uint32(1)
	equipCompId := uuid.New()
	cashCompId := uuid.New()

	// A worn hat (negative slot) — this should land in the equipment look.
	wornHat := asset.NewBuilder(equipCompId, 1002357).SetId(7).SetSlot(-1).SetQuantity(1).MustBuild()
	equipComp := compartment.NewBuilder(equipCompId, charId, inventory2.TypeValueEquip, 24).
		AddAsset(wornHat).MustBuild()

	// A cash pet asset (templateId 5000029 → cash, petId>0 → IsPet()).
	petAsset := asset.NewBuilder(cashCompId, 5000029).SetId(15).SetSlot(1).SetPetId(1).SetQuantity(1).MustBuild()
	cashComp := compartment.NewBuilder(cashCompId, charId, inventory2.TypeValueCash, 24).
		AddAsset(petAsset).MustBuild()

	useComp := compartment.NewBuilder(uuid.New(), charId, inventory2.TypeValueUse, 24).MustBuild()
	setupComp := compartment.NewBuilder(uuid.New(), charId, inventory2.TypeValueSetup, 24).MustBuild()
	etcComp := compartment.NewBuilder(uuid.New(), charId, inventory2.TypeValueETC, 24).MustBuild()

	rawInv := inventory.NewBuilder(charId).
		SetEquipable(equipComp).
		SetConsumable(useComp).
		SetSetup(setupComp).
		SetEtc(etcComp).
		SetCash(cashComp).
		MustBuild()

	// Mimic InventoryDecorator: SetInventory builds the look from worn items.
	m := createTestCharacter(charId, "Tester", 10).SetInventory(rawInv)

	// Sanity: the look was built before enrichment.
	if hat, ok := m.Equipment().Get("hat"); !ok || hat.Equipable == nil {
		t.Fatalf("precondition failed: hat look not built by SetInventory (ok=%v)", ok)
	}

	got := p.PetAssetEnrichmentDecorator(m)

	// The worn hat must still be in the look after enrichment.
	hat, ok := got.Equipment().Get("hat")
	if !ok || hat.Equipable == nil {
		t.Errorf("equipment look was clobbered: hat.Equipable=nil (ok=%v) — character renders naked", ok)
	} else if hat.Equipable.TemplateId() != 1002357 {
		t.Errorf("hat template = %d, want 1002357", hat.Equipable.TemplateId())
	}

	// The pet asset should be enriched with its live name.
	cashAssets := got.Inventory().Cash().Assets()
	if len(cashAssets) != 1 {
		t.Fatalf("cash assets = %d, want 1", len(cashAssets))
	}
	if cashAssets[0].PetName() != "Pet" {
		t.Errorf("pet name = %q, want %q", cashAssets[0].PetName(), "Pet")
	}
	// PetSlot must mirror the live pet's slot (-1 = despawned). PetSpawnHandle
	// relies on this to decide spawn vs despawn; without enrichment it defaults
	// to 0 and the handler always despawns.
	if cashAssets[0].PetSlot() != -1 {
		t.Errorf("pet slot = %d, want -1 (from live pet)", cashAssets[0].PetSlot())
	}
}
