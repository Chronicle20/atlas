package compartment_test

import (
	"context"
	"testing"
	"time"

	"atlas-inventory/asset"
	"atlas-inventory/compartment"
	"atlas-inventory/data/consumable"
	dcp "atlas-inventory/data/consumable/mock"
	"atlas-inventory/kafka/message"

	"github.com/Chronicle20/atlas/libs/atlas-constants/inventory"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
	"github.com/google/uuid"
)

// CanAccommodate mirrors CreateAsset's success condition, and — crucially — is
// merge-aware: a full USE tab does NOT block a stackable reward that fits into an
// existing stack, but does block one that has no stack to merge into.
func TestCanAccommodate(t *testing.T) {
	characterId := uint32(901)

	l := testLogger()
	te := testTenant()
	ctx := tenant.WithContext(context.Background(), te)
	db := testDatabase(t, l)

	mb := message.NewBuffer()

	dcpi := &dcp.ProcessorImpl{}
	dcpi.GetByIdFn = func(itemId uint32) (consumable.Model, error) {
		return consumable.Extract(consumable.RestModel{SlotMax: 100})
	}
	ap := asset.NewProcessor(l, ctx, db).WithConsumableProcessor(dcpi)
	cp := compartment.NewProcessor(l, ctx, db).WithAssetProcessor(ap)

	// EQUIP: capacity 24, empty -> free slots.
	if _, err := cp.Create(mb)(uuid.New(), characterId, inventory.TypeValueEquip, 24); err != nil {
		t.Fatalf("create equip compartment: %v", err)
	}
	// USE: capacity 2, filled to both slots -> full. Slot 1 holds a partial stack
	// of 2000002 (50/100), slot 2 holds 2000003.
	if _, err := cp.Create(mb)(uuid.New(), characterId, inventory.TypeValueUse, 2); err != nil {
		t.Fatalf("create use compartment: %v", err)
	}
	if err := cp.CreateAsset(mb)(uuid.New(), characterId, inventory.TypeValueUse, 2000002, 50, time.Time{}, 0, 0, 0, false); err != nil {
		t.Fatalf("seed 2000002: %v", err)
	}
	if err := cp.CreateAsset(mb)(uuid.New(), characterId, inventory.TypeValueUse, 2000003, 1, time.Time{}, 0, 0, 0, false); err != nil {
		t.Fatalf("seed 2000003: %v", err)
	}

	results, err := cp.CanAccommodate(characterId, []compartment.AccommodationRequest{
		{TemplateId: 2000002, Quantity: 30}, // merges into the 50/100 stack -> fits despite full tab
		{TemplateId: 2000004, Quantity: 30}, // no stack to merge into, tab full -> does not fit
		{TemplateId: 2070000, Quantity: 1},  // throwing star: never merges, tab full -> does not fit
		{TemplateId: 1302000, Quantity: 1},  // equip: EQUIP tab has room -> fits
	})
	if err != nil {
		t.Fatalf("CanAccommodate: %v", err)
	}

	want := map[uint32]bool{2000002: true, 2000004: false, 2070000: false, 1302000: true}
	if len(results) != len(want) {
		t.Fatalf("got %d results, want %d", len(results), len(want))
	}
	for _, r := range results {
		if want[r.TemplateId] != r.Accommodated {
			t.Errorf("item %d: accommodated=%v, want %v", r.TemplateId, r.Accommodated, want[r.TemplateId])
		}
	}
}
