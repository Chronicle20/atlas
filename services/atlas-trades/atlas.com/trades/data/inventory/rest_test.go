package inventory

import (
	"testing"

	"github.com/jtumidanski/api2go/jsonapi"

	"github.com/Chronicle20/atlas/libs/atlas-constants/asset"
	"github.com/Chronicle20/atlas/libs/atlas-constants/inventory"
)

// compartmentDocument is a compartment as atlas-inventory serialises it: the
// assets live in `included`, referenced from the compartment's `relationships`.
const compartmentDocument = `{
  "data": {
    "type": "compartments",
    "id": "6f1c9a52-3d2a-4a4e-9c4e-1b2c3d4e5f60",
    "attributes": {"type": 2, "capacity": 24},
    "relationships": {
      "assets": {"data": [{"type": "assets", "id": "7001"}, {"type": "assets", "id": "7002"}]}
    }
  },
  "included": [
    {"type": "assets", "id": "7001", "attributes": {"slot": 1, "templateId": 2000000, "quantity": 100, "flag": 8}},
    {"type": "assets", "id": "7002", "attributes": {"slot": -11, "templateId": 1302000, "quantity": 1, "flag": 0}}
  ]
}`

// TestExtractCarriesAssetAttributes pins the whole reason this reader exists.
// The compartment resource carries its assets as a RELATIONSHIP, so without
// SetToManyReferenceIDs the decode fails outright and without
// SetReferencedStructs each asset decodes to a bare id with slot, quantity and
// — most dangerously — FLAG all zero. A zero flag reads as "no untradeable
// bit", which would make every restricted item pass FR-4.1.
func TestExtractCarriesAssetAttributes(t *testing.T) {
	var rm RestModel
	if err := jsonapi.Unmarshal([]byte(compartmentDocument), &rm); err != nil {
		t.Fatalf("unmarshal compartment: %v", err)
	}
	m, err := Extract(rm)
	if err != nil {
		t.Fatalf("extract: %v", err)
	}

	if m.Type() != inventory.TypeValueUse {
		t.Errorf("type: got %d, want %d", m.Type(), inventory.TypeValueUse)
	}
	if m.Capacity() != 24 {
		t.Errorf("capacity: got %d, want 24", m.Capacity())
	}
	if got := len(m.Assets()); got != 2 {
		t.Fatalf("assets: got %d, want 2", got)
	}

	a, ok := m.FindBySlot(1)
	if !ok {
		t.Fatal("FindBySlot(1) missed the asset in slot 1")
	}
	if a.Id() != asset.Id(7001) || a.TemplateId() != 2000000 || a.Quantity() != 100 {
		t.Errorf("asset in slot 1: got %+v, want asset 7001 of template 2000000 x100", a)
	}
	if a.Flag() != uint16(asset.FlagUntradeable) {
		t.Errorf("asset flag: got %d, want the untradeable bit %d", a.Flag(), uint16(asset.FlagUntradeable))
	}
}

// TestFindBySlotResolvesANegativeSlot pins that equipped assets — which live in
// the EQUIP compartment at negative positions — are addressable. The staging
// path relies on reading them in order to REFUSE them (FR-4.4).
func TestFindBySlotResolvesANegativeSlot(t *testing.T) {
	var rm RestModel
	if err := jsonapi.Unmarshal([]byte(compartmentDocument), &rm); err != nil {
		t.Fatalf("unmarshal compartment: %v", err)
	}
	m, err := Extract(rm)
	if err != nil {
		t.Fatalf("extract: %v", err)
	}
	a, ok := m.FindBySlot(-11)
	if !ok {
		t.Fatal("FindBySlot(-11) missed the equipped asset")
	}
	if a.TemplateId() != 1302000 {
		t.Errorf("equipped asset template: got %d, want 1302000", a.TemplateId())
	}
}

// TestFindBySlotMissesAnEmptySlot pins that an unoccupied slot is reported
// absent rather than as a zero-valued asset.
func TestFindBySlotMissesAnEmptySlot(t *testing.T) {
	var rm RestModel
	if err := jsonapi.Unmarshal([]byte(compartmentDocument), &rm); err != nil {
		t.Fatalf("unmarshal compartment: %v", err)
	}
	m, err := Extract(rm)
	if err != nil {
		t.Fatalf("extract: %v", err)
	}
	if _, ok := m.FindBySlot(9); ok {
		t.Error("FindBySlot(9) reported an asset in an empty slot")
	}
}

// TestAssetsIsNotWritableThroughTheGetter pins that a caller cannot reach into
// the compartment's asset list through the returned slice.
func TestAssetsIsNotWritableThroughTheGetter(t *testing.T) {
	m := Model{assets: []Asset{NewAsset(1, 1, 2000000, 5, 0)}}
	escaped := m.Assets()
	escaped[0] = NewAsset(9, 9, 9, 9, 9)
	if m.Assets()[0].Id() != 1 {
		t.Error("writing through Assets() mutated the compartment")
	}
}
