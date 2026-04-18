package clientbound

import (
	"testing"
	"time"

	"github.com/Chronicle20/atlas/libs/atlas-packet/inventory"
	"github.com/Chronicle20/atlas/libs/atlas-packet/model"
	"github.com/Chronicle20/atlas/libs/atlas-packet/test"
	testlog "github.com/sirupsen/logrus/hooks/test"
)

func TestChangeBatchQuantityUpdateRoundTrip(t *testing.T) {
	for _, v := range test.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := test.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			input := NewChangeBatch(false, inventory.NewQuantityUpdateEntry(2, 5, 100))
			output := ChangeBatch{}
			test.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
			if output.Silent() != input.Silent() {
				t.Errorf("silent: got %v, want %v", output.Silent(), input.Silent())
			}
			if len(output.Entries()) != 1 {
				t.Fatalf("entries: got %d, want 1", len(output.Entries()))
			}
			qe, ok := output.Entries()[0].(inventory.QuantityUpdateEntry)
			if !ok {
				t.Fatalf("entry type: got %T, want QuantityUpdateEntry", output.Entries()[0])
			}
			if qe.InventoryType() != 2 {
				t.Errorf("inventoryType: got %v, want 2", qe.InventoryType())
			}
			if qe.Slot() != 5 {
				t.Errorf("slot: got %v, want 5", qe.Slot())
			}
			if qe.Quantity() != 100 {
				t.Errorf("quantity: got %v, want 100", qe.Quantity())
			}
		})
	}
}

func TestChangeBatchMoveRoundTrip(t *testing.T) {
	for _, v := range test.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := test.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			input := NewChangeBatch(true, inventory.NewMoveEntry(2, 3, 7))
			output := ChangeBatch{}
			test.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
			if output.Silent() != true {
				t.Errorf("silent: got %v, want true", output.Silent())
			}
			if len(output.Entries()) != 1 {
				t.Fatalf("entries: got %d, want 1", len(output.Entries()))
			}
			me, ok := output.Entries()[0].(inventory.MoveEntry)
			if !ok {
				t.Fatalf("entry type: got %T, want MoveEntry", output.Entries()[0])
			}
			if me.OldSlot() != 3 {
				t.Errorf("oldSlot: got %v, want 3", me.OldSlot())
			}
			if me.NewSlot() != 7 {
				t.Errorf("newSlot: got %v, want 7", me.NewSlot())
			}
		})
	}
}

func TestChangeBatchMoveEquipRoundTrip(t *testing.T) {
	for _, v := range test.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := test.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			// equip type, newSlot < 0 => addMov = 2
			input := NewChangeBatch(false, inventory.NewMoveEntry(1, 5, -1))
			output := ChangeBatch{}
			test.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
			me := output.Entries()[0].(inventory.MoveEntry)
			if me.OldSlot() != 5 {
				t.Errorf("oldSlot: got %v, want 5", me.OldSlot())
			}
			if me.NewSlot() != -1 {
				t.Errorf("newSlot: got %v, want -1", me.NewSlot())
			}
		})
	}
}

func TestChangeBatchRemoveRoundTrip(t *testing.T) {
	for _, v := range test.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := test.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			input := NewChangeBatch(false, inventory.NewRemoveEntry(2, 3))
			output := ChangeBatch{}
			test.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
			re := output.Entries()[0].(inventory.RemoveEntry)
			if re.InventoryType() != 2 {
				t.Errorf("inventoryType: got %v, want 2", re.InventoryType())
			}
			if re.Slot() != 3 {
				t.Errorf("slot: got %v, want 3", re.Slot())
			}
		})
	}
}

func TestChangeBatchAddStackableRoundTrip(t *testing.T) {
	asset := model.NewAsset(true, 0, 2000000, time.Time{}).
		SetStackableInfo(5, 0, 0)
	for _, v := range test.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := test.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			input := NewChangeBatch(false, inventory.NewAddEntry(2, 5, asset))
			output := ChangeBatch{}
			test.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
			ae := output.Entries()[0].(inventory.AddEntry)
			if ae.InventoryType() != 2 {
				t.Errorf("inventoryType: got %v, want 2", ae.InventoryType())
			}
			if ae.Slot() != 5 {
				t.Errorf("slot: got %v, want 5", ae.Slot())
			}
			if ae.Asset().TemplateId() != 2000000 {
				t.Errorf("templateId: got %v, want 2000000", ae.Asset().TemplateId())
			}
		})
	}
}

// TestChangeBatchAddStackableZeroPositionContract pins the wire-format invariant
// that an AddEntry's asset body is encoded with zeroPosition=true. The entry
// header already emits WriteInt16(slot); a zeroPosition=false body would
// double-encode the slot (extra 1 byte for stackable, extra 2 bytes for equip)
// and misalign the client decoder. This was the regression introduced in
// 6db749769 and manifested as a client crash on item gain.
func TestChangeBatchAddStackableZeroPositionContract(t *testing.T) {
	l, _ := testlog.NewNullLogger()
	ctx := test.CreateContext("GMS", 83, 1)
	goodAsset := model.NewAsset(true, 7, 2000000, time.Time{}).SetStackableInfo(5, 0, 0)
	badAsset := model.NewAsset(false, 7, 2000000, time.Time{}).SetStackableInfo(5, 0, 0)

	good := NewChangeBatch(false, inventory.NewAddEntry(2, 7, goodAsset)).Encode(l, ctx)(nil)
	bad := NewChangeBatch(false, inventory.NewAddEntry(2, 7, badAsset)).Encode(l, ctx)(nil)

	if diff := len(bad) - len(good); diff != 1 {
		t.Fatalf("stackable zeroPosition=false should add exactly 1 slot byte, got diff of %d", diff)
	}
}

func TestChangeBatchAddEquipmentZeroPositionContract(t *testing.T) {
	l, _ := testlog.NewNullLogger()
	ctx := test.CreateContext("GMS", 83, 1)
	goodAsset := model.NewAsset(true, 1, 1302000, time.Time{}).
		SetEquipmentStats(0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0).
		SetEquipmentMeta(7, 0, 0, 0, 0, 0)
	badAsset := model.NewAsset(false, 1, 1302000, time.Time{}).
		SetEquipmentStats(0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0).
		SetEquipmentMeta(7, 0, 0, 0, 0, 0)

	good := NewChangeBatch(false, inventory.NewAddEntry(1, 1, goodAsset)).Encode(l, ctx)(nil)
	bad := NewChangeBatch(false, inventory.NewAddEntry(1, 1, badAsset)).Encode(l, ctx)(nil)

	if diff := len(bad) - len(good); diff != 2 {
		t.Fatalf("equip zeroPosition=false should add a 2-byte slot prefix on GMS v83, got diff of %d", diff)
	}
}

func TestChangeBatchMultipleEntriesRoundTrip(t *testing.T) {
	asset := model.NewAsset(true, 0, 2000000, time.Time{}).
		SetStackableInfo(10, 0, 0)
	for _, v := range test.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := test.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			input := NewChangeBatch(false,
				inventory.NewAddEntry(2, 5, asset),
				inventory.NewQuantityUpdateEntry(2, 3, 50),
				inventory.NewRemoveEntry(2, 7),
			)
			output := ChangeBatch{}
			test.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
			if len(output.Entries()) != 3 {
				t.Fatalf("entries: got %d, want 3", len(output.Entries()))
			}
			if _, ok := output.Entries()[0].(inventory.AddEntry); !ok {
				t.Errorf("entry[0]: got %T, want AddEntry", output.Entries()[0])
			}
			if _, ok := output.Entries()[1].(inventory.QuantityUpdateEntry); !ok {
				t.Errorf("entry[1]: got %T, want QuantityUpdateEntry", output.Entries()[1])
			}
			if _, ok := output.Entries()[2].(inventory.RemoveEntry); !ok {
				t.Errorf("entry[2]: got %T, want RemoveEntry", output.Entries()[2])
			}
		})
	}
}
