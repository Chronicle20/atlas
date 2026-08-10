package trade

import (
	"errors"
	"math"
	"testing"

	"github.com/Chronicle20/atlas/libs/atlas-constants/asset"
	"github.com/Chronicle20/atlas/libs/atlas-constants/inventory"
)

// TestRestrictionsRejectUntradeableFlags pins FR-4.1 against both flags.
func TestRestrictionsRejectUntradeableFlags(t *testing.T) {
	for name, flag := range map[string]asset.Flag{
		"untradeable":      asset.FlagUntradeable,
		"mergeUntradeable": asset.FlagMergeUntradeable,
	} {
		t.Run(name, func(t *testing.T) {
			err := checkRestrictions(assetView{Flags: uint16(flag)}, itemDataView{}, byte(inventory.TypeValueUse))
			if !errors.Is(err, errUntradeableFlag) {
				t.Fatalf("checkRestrictions: got %v, want %v", err, errUntradeableFlag)
			}
		})
	}
}

// TestRestrictionsRejectTradeBlock pins FR-4.2.
func TestRestrictionsRejectTradeBlock(t *testing.T) {
	err := checkRestrictions(assetView{}, itemDataView{TradeBlock: true}, byte(inventory.TypeValueUse))
	if !errors.Is(err, errTradeBlock) {
		t.Fatalf("checkRestrictions: got %v, want %v", err, errTradeBlock)
	}
}

// TestRestrictionsRejectUnreadableItemData pins the PRD's explicit rule: a
// missing flag must NOT be read as "tradeable". An atlas-data LOOKUP FAILURE
// (not a false value) is a refusal (design §7).
func TestRestrictionsRejectUnreadableItemData(t *testing.T) {
	err := checkRestrictions(assetView{}, itemDataView{Unreadable: true}, byte(inventory.TypeValueUse))
	if !errors.Is(err, errItemDataUnknown) {
		t.Fatalf("checkRestrictions: got %v, want %v", err, errItemDataUnknown)
	}
}

// TestRestrictionsRejectEquippedSource pins FR-4.4. Worn equipment lives in the
// EQUIP compartment at a NEGATIVE slot — there is no separate EQUIPPED
// inventory.Type — so the source slot is the discriminator.
func TestRestrictionsRejectEquippedSource(t *testing.T) {
	err := checkRestrictions(assetView{SourceSlot: -11}, itemDataView{}, byte(inventory.TypeValueEquip))
	if !errors.Is(err, errEquipped) {
		t.Fatalf("checkRestrictions: got %v, want %v", err, errEquipped)
	}
}

// TestRestrictionsAcceptAnUnequippedEquip guards the equipped rule against
// over-reach: an equip item sitting in the bag has a positive slot and is
// perfectly tradeable.
func TestRestrictionsAcceptAnUnequippedEquip(t *testing.T) {
	if err := checkRestrictions(assetView{SourceSlot: 4}, itemDataView{}, byte(inventory.TypeValueEquip)); err != nil {
		t.Fatalf("checkRestrictions refused an unequipped equip: %v", err)
	}
}

// TestRestrictionsRejectUnstageableCompartments pins FR-4.3 and the signed-type
// boundary together. inventory.Type is an int8 and this codebase models exactly
// five compartments, so a QUEST compartment — on any client version that has one
// — is not among them and is refused, as is any byte above 127 that would
// otherwise arrive as a negative type.
func TestRestrictionsRejectUnstageableCompartments(t *testing.T) {
	for name, source := range map[string]byte{
		"zero":        0,
		"beyond cash": byte(inventory.TypeValueCash) + 1,
		"above int8":  200,
		"all bits":    math.MaxUint8,
		// 0x80 is the first byte that a naive inventory.Type(b) conversion
		// would turn into a NEGATIVE compartment (-128) rather than rejecting.
		"first negative int8": 0x80,
	} {
		t.Run(name, func(t *testing.T) {
			err := checkRestrictions(assetView{}, itemDataView{}, source)
			if !errors.Is(err, errUnknownInventory) {
				t.Fatalf("checkRestrictions(source=%d): got %v, want %v", source, err, errUnknownInventory)
			}
		})
	}
}

// TestRestrictionsAcceptEveryKnownCompartment guards against an over-broad
// compartment rule: all five shared inventory types must stage.
func TestRestrictionsAcceptEveryKnownCompartment(t *testing.T) {
	for _, it := range inventory.Types {
		if err := checkRestrictions(assetView{}, itemDataView{}, byte(it)); err != nil {
			t.Errorf("checkRestrictions refused compartment %d: %v", it, err)
		}
	}
}

// TestRestrictionsAcceptAPlainTradeableItem guards against an over-broad rule.
func TestRestrictionsAcceptAPlainTradeableItem(t *testing.T) {
	if err := checkRestrictions(assetView{}, itemDataView{}, byte(inventory.TypeValueUse)); err != nil {
		t.Fatalf("staging refused a plain tradeable item: %v", err)
	}
}

// TestRestrictionsAcceptAnUnrelatedFlag pins that the untradeable rule tests the
// two untradeable bits specifically, not "any flag at all" — a locked or cold
// item is still tradeable.
func TestRestrictionsAcceptAnUnrelatedFlag(t *testing.T) {
	flags := uint16(asset.FlagLock) | uint16(asset.FlagCold)
	if err := checkRestrictions(assetView{Flags: flags}, itemDataView{}, byte(inventory.TypeValueUse)); err != nil {
		t.Fatalf("checkRestrictions refused an item carrying only unrelated flags: %v", err)
	}
}

// TestStageableInventoryTypeRoundTrips pins the decode boundary itself: each of
// the five compartment bytes decodes to its own shared type.
func TestStageableInventoryTypeRoundTrips(t *testing.T) {
	for _, want := range inventory.Types {
		got, ok := stageableInventoryType(byte(want))
		if !ok || got != want {
			t.Errorf("stageableInventoryType(%d) = (%d, %t), want (%d, true)", byte(want), got, ok, want)
		}
	}
}
