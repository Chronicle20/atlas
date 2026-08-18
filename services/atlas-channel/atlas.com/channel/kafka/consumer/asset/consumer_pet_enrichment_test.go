package asset

import (
	asset2 "atlas-channel/kafka/message/asset"
	"atlas-channel/pet"
	model2 "atlas-channel/socket/model"
	"encoding/json"
	"go/ast"
	"go/parser"
	"go/token"
	"testing"

	"github.com/sirupsen/logrus"
)

// petReviveUpdatedEvent is the asset UPDATED event atlas-inventory emits when a
// Water of Life revive extends a pet asset's expiration, captured verbatim from
// atlas-channel on atlas-pr-1360 (2026-08-16 11:16:48.741Z).
const petReviveUpdatedEvent = `{
  "transactionId": "e3f5b4c4-70fc-4777-8822-c5ad04b46092",
  "characterId": 1,
  "compartmentId": "02c90e6c-915b-42cc-a829-803b4aea3586",
  "assetId": 14,
  "templateId": 5000012,
  "slot": 1,
  "type": "UPDATED",
  "body": {
    "expiration": "2026-11-14T11:16:48.541181601Z",
    "createdAt": "2026-08-15T21:29:23.419974Z",
    "quantity": 1,
    "cashId": "1437245515603478011",
    "commodityId": 60000024,
    "purchaseBy": 1,
    "petId": 1
  }
}`

// TestAssetUpdatedEventEnrichesPetBlock pins the fix for a bug that reached a
// live client: handleAssetUpdatedEvent built the asset straight from the event
// body and announced an InventoryChange add without calling enrichPetAsset, the
// only add path in this file that skipped it.
//
// The asset status events carry no pet fields, so the encoded GW_ItemSlotPet
// block came out with a serial falling back to the Atlas pet id (1 instead of
// the cash serial 1437245515603478011), a blank name and zeroed level, closeness
// and fullness. The block is the correct LENGTH, so nothing desyncs and the
// client accepts it — then can no longer bind its inventory slot to the spawned
// pet (CPet::GetItemSlot). Observed on atlas-pr-1360: pet commands stopped being
// sent at all, and despawning the pet closed the client.
func TestAssetUpdatedEventEnrichesPetBlock(t *testing.T) {
	const wantSerial = uint64(1437245515603478011)

	var e asset2.StatusEvent[asset2.UpdatedStatusEventBody]
	if err := json.Unmarshal([]byte(petReviveUpdatedEvent), &e); err != nil {
		t.Fatalf("unmarshal event: %v", err)
	}

	pm, err := pet.NewModelBuilder(1, wantSerial, 5000012, "Pet").
		SetLevel(1).
		SetCloseness(37).
		SetFullness(100).
		SetSlot(0).
		Build()
	if err != nil {
		t.Fatalf("build pet: %v", err)
	}
	fetch := func(petId uint32) (pet.Model, error) {
		if petId != 1 {
			t.Fatalf("enrichment asked for pet [%d], want [1]", petId)
		}
		return pm, nil
	}

	raw := buildAssetFromUpdatedBody(e)

	// The regression itself: without enrichment the wire serial is the Atlas pet
	// id, not the cash serial the client knows the pet by.
	if got := model2.NewAsset(true, raw).PetSerialNumber(); got == wantSerial {
		t.Fatalf("unenriched asset already carries serial [%d]; this test no longer pins anything", got)
	}

	a := enrichPetAssetWith(logrus.New(), fetch, raw)
	enc := model2.NewAsset(true, a)

	if got := enc.PetSerialNumber(); got != wantSerial {
		t.Errorf("pet serial = %d, want %d", got, wantSerial)
	}
	if got := a.PetName(); got != "Pet" {
		t.Errorf("pet name = %q, want %q", got, "Pet")
	}
	if got := a.PetLevel(); got != 1 {
		t.Errorf("pet level = %d, want 1", got)
	}
	if got := a.Closeness(); got != 37 {
		t.Errorf("closeness = %d, want 37", got)
	}
	if got := a.Fullness(); got != 100 {
		t.Errorf("fullness = %d, want 100", got)
	}
	if got := a.PetDeadDate(); !got.Equal(pm.Expiration()) {
		t.Errorf("dead date = %s, want %s", got, pm.Expiration())
	}
}

// TestEnrichPetAssetLeavesNonPetsAlone keeps the enrichment inert for the
// ordinary items that dominate the UPDATED stream.
func TestEnrichPetAssetLeavesNonPetsAlone(t *testing.T) {
	var e asset2.StatusEvent[asset2.UpdatedStatusEventBody]
	if err := json.Unmarshal([]byte(petReviveUpdatedEvent), &e); err != nil {
		t.Fatalf("unmarshal event: %v", err)
	}
	e.TemplateId = 2000000 // a red potion: not a pet
	e.Body.PetId = 0

	fetch := func(petId uint32) (pet.Model, error) {
		t.Fatalf("enrichment fetched pet [%d] for a non-pet asset", petId)
		return pet.Model{}, nil
	}

	a := buildAssetFromUpdatedBody(e)
	if got := enrichPetAssetWith(logrus.New(), fetch, a); got.PetName() != "" {
		t.Errorf("non-pet asset gained pet data: %q", got.PetName())
	}
}

// TestEveryAddEntryEnrichesPets is the guard that would have caught the bug the
// test above pins. The enrichment is a call the author of a new add path has to
// remember, and forgetting it is silent: the packet still encodes, still has the
// right length, and only a live client shows the damage. This walks the
// consumer's AST and fails any function that builds an InventoryChange add entry
// without also enriching the asset.
func TestEveryAddEntryEnrichesPets(t *testing.T) {
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, "consumer.go", nil, 0)
	if err != nil {
		t.Fatalf("parse consumer.go: %v", err)
	}

	calls := func(n ast.Node, sel string) bool {
		found := false
		ast.Inspect(n, func(n ast.Node) bool {
			c, ok := n.(*ast.CallExpr)
			if !ok {
				return true
			}
			if id, ok := c.Fun.(*ast.Ident); ok && id.Name == sel {
				found = true
			}
			if s, ok := c.Fun.(*ast.SelectorExpr); ok && s.Sel.Name == sel {
				found = true
			}
			return !found
		})
		return found
	}

	checked := 0
	for _, d := range f.Decls {
		fd, ok := d.(*ast.FuncDecl)
		if !ok || !calls(fd, "NewAddEntry") {
			continue
		}
		checked++
		if !calls(fd, "enrichPetAsset") {
			t.Errorf("%s builds an InventoryChange add entry without enrichPetAsset; a pet asset "+
				"sent from there loses its serial, name, level, closeness and fullness", fd.Name.Name)
		}
	}
	if checked == 0 {
		t.Fatal("found no NewAddEntry call sites in consumer.go; the guard is not looking at anything")
	}
}
