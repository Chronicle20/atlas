package compartment_test

import (
	"atlas-inventory/asset"
	"atlas-inventory/compartment"
	"atlas-inventory/data/tradeability"
	"atlas-inventory/kafka/message"
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"

	tradeabilityMock "atlas-inventory/data/tradeability/mock"

	af "github.com/Chronicle20/atlas/libs/atlas-constants/asset"
	"github.com/Chronicle20/atlas/libs/atlas-constants/inventory"
	"github.com/Chronicle20/atlas/libs/atlas-constants/item"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

// TestApplyAssetKarmaMarksAnUntradeableEquip is the happy path: an untradeable,
// karma-applicable equip gains the EQUIP karma bit (0x10) and nothing else.
func TestApplyAssetKarmaMarksAnUntradeableEquip(t *testing.T) {
	characterId := uint32(501)
	templateId := uint32(1002357) // Zakum Helmet

	l := testLogger()
	te := testTenant()
	ctx := tenant.WithContext(context.Background(), te)
	db := testDatabase(t, l)

	mb := message.NewBuffer()

	ap := asset.NewProcessor(l, ctx, db)
	tm := &tradeabilityMock.ProcessorMock{
		GetFunc: func(inventoryType inventory.Type, tid item.Id) (tradeability.Model, error) {
			return tradeability.NewModel(true, 1), nil
		},
	}
	cp := compartment.NewProcessor(l, ctx, db).WithAssetProcessor(ap).WithTradeabilityProcessor(tm)

	c, err := cp.Create(mb)(uuid.New(), characterId, inventory.TypeValueEquip, 24)
	if err != nil {
		t.Fatalf("Failed to create compartment: %v", err)
	}

	slot := int16(3)
	m := asset.NewBuilder(c.Id(), templateId).SetSlot(slot).SetCreatedAt(time.Now()).AddFlag(af.FlagUntradeable).Build()
	if _, err := ap.CreateFromModel(mb)(uuid.New(), characterId, m); err != nil {
		t.Fatalf("Failed to create asset: %v", err)
	}

	if err := cp.ApplyAssetKarma(mb)(uuid.New(), characterId, inventory.TypeValueEquip, slot, 0, false); err != nil {
		t.Fatalf("ApplyAssetKarma returned unexpected error: %v", err)
	}

	a, err := ap.GetBySlot(c.Id(), slot)
	if err != nil {
		t.Fatalf("Failed to reload asset: %v", err)
	}
	if !af.HasFlag(a.Flag(), af.FlagKarmaEquip) {
		t.Fatal("expected the EQUIP karma bit (0x10) to be set")
	}
	if af.HasFlag(a.Flag(), af.FlagKarmaUse) && !af.HasFlag(a.Flag(), af.FlagSpikes) {
		t.Fatal("the BUNDLE karma bit (0x02 = FlagSpikes on an equip) was written; wrong bit")
	}
	if !af.HasFlag(a.Flag(), af.FlagUntradeable) {
		t.Fatal("FlagUntradeable was disturbed")
	}
}

// TestApplyAssetKarmaRefusesIneligibleTarget is the FR-6.4 re-assertion: the
// channel's gates are advisory across a service boundary.
func TestApplyAssetKarmaRefusesIneligibleTarget(t *testing.T) {
	characterId := uint32(502)
	templateId := uint32(1002357)

	l := testLogger()
	te := testTenant()
	ctx := tenant.WithContext(context.Background(), te)
	db := testDatabase(t, l)

	mb := message.NewBuffer()

	ap := asset.NewProcessor(l, ctx, db)
	tm := &tradeabilityMock.ProcessorMock{
		GetFunc: func(inventoryType inventory.Type, tid item.Id) (tradeability.Model, error) {
			return tradeability.NewModel(true, 0), nil // tradeAvailable == 0
		},
	}
	cp := compartment.NewProcessor(l, ctx, db).WithAssetProcessor(ap).WithTradeabilityProcessor(tm)

	c, err := cp.Create(mb)(uuid.New(), characterId, inventory.TypeValueEquip, 24)
	if err != nil {
		t.Fatalf("Failed to create compartment: %v", err)
	}

	slot := int16(3)
	m := asset.NewBuilder(c.Id(), templateId).SetSlot(slot).SetCreatedAt(time.Now()).AddFlag(af.FlagUntradeable).Build()
	if _, err := ap.CreateFromModel(mb)(uuid.New(), characterId, m); err != nil {
		t.Fatalf("Failed to create asset: %v", err)
	}

	if err := cp.ApplyAssetKarma(mb)(uuid.New(), characterId, inventory.TypeValueEquip, slot, 0, false); err == nil {
		t.Fatal("expected ApplyAssetKarma to refuse a target with tradeAvailable == 0")
	}

	a, err := ap.GetBySlot(c.Id(), slot)
	if err != nil {
		t.Fatalf("Failed to reload asset: %v", err)
	}
	if af.HasFlag(a.Flag(), af.FlagKarmaEquip) {
		t.Fatal("a refused apply still mutated the flag")
	}
}

// TestApplyAssetKarmaRefusesAlreadyMarked is FR-6.7: a redelivered command must
// not silently consume a second scissors against an already-marked item.
func TestApplyAssetKarmaRefusesAlreadyMarked(t *testing.T) {
	characterId := uint32(503)
	templateId := uint32(1002357)

	l := testLogger()
	te := testTenant()
	ctx := tenant.WithContext(context.Background(), te)
	db := testDatabase(t, l)

	mb := message.NewBuffer()

	ap := asset.NewProcessor(l, ctx, db)
	tm := &tradeabilityMock.ProcessorMock{
		GetFunc: func(inventoryType inventory.Type, tid item.Id) (tradeability.Model, error) {
			return tradeability.NewModel(true, 1), nil
		},
	}
	cp := compartment.NewProcessor(l, ctx, db).WithAssetProcessor(ap).WithTradeabilityProcessor(tm)

	c, err := cp.Create(mb)(uuid.New(), characterId, inventory.TypeValueEquip, 24)
	if err != nil {
		t.Fatalf("Failed to create compartment: %v", err)
	}

	slot := int16(3)
	m := asset.NewBuilder(c.Id(), templateId).SetSlot(slot).SetCreatedAt(time.Now()).AddFlag(af.FlagUntradeable).AddFlag(af.FlagKarmaEquip).Build()
	if _, err := ap.CreateFromModel(mb)(uuid.New(), characterId, m); err != nil {
		t.Fatalf("Failed to create asset: %v", err)
	}

	if err := cp.ApplyAssetKarma(mb)(uuid.New(), characterId, inventory.TypeValueEquip, slot, 0, false); err == nil {
		t.Fatal("expected ApplyAssetKarma to refuse an already-marked asset")
	}
}

// TestApplyAssetKarmaRefusesLockedTarget mirrors client gate 1
// (GW_ItemSlotEquip::IsProtectedItem, gms_v83 @0x4E9506).
func TestApplyAssetKarmaRefusesLockedTarget(t *testing.T) {
	characterId := uint32(504)
	templateId := uint32(1002357)

	l := testLogger()
	te := testTenant()
	ctx := tenant.WithContext(context.Background(), te)
	db := testDatabase(t, l)

	mb := message.NewBuffer()

	ap := asset.NewProcessor(l, ctx, db)
	tm := &tradeabilityMock.ProcessorMock{
		GetFunc: func(inventoryType inventory.Type, tid item.Id) (tradeability.Model, error) {
			return tradeability.NewModel(true, 1), nil
		},
	}
	cp := compartment.NewProcessor(l, ctx, db).WithAssetProcessor(ap).WithTradeabilityProcessor(tm)

	c, err := cp.Create(mb)(uuid.New(), characterId, inventory.TypeValueEquip, 24)
	if err != nil {
		t.Fatalf("Failed to create compartment: %v", err)
	}

	slot := int16(3)
	m := asset.NewBuilder(c.Id(), templateId).SetSlot(slot).SetCreatedAt(time.Now()).AddFlag(af.FlagUntradeable).AddFlag(af.FlagLock).Build()
	if _, err := ap.CreateFromModel(mb)(uuid.New(), characterId, m); err != nil {
		t.Fatalf("Failed to create asset: %v", err)
	}

	if err := cp.ApplyAssetKarma(mb)(uuid.New(), characterId, inventory.TypeValueEquip, slot, 0, false); err == nil {
		t.Fatal("expected ApplyAssetKarma to refuse a Sealing-Lock'd asset")
	}
}

// TestApplyAssetKarmaRefusesAlreadyTradeableTarget is gate 4: karma exists to
// unlock an UNTRADEABLE item; marking a tradeable one consumes the scissors for
// nothing.
func TestApplyAssetKarmaRefusesAlreadyTradeableTarget(t *testing.T) {
	characterId := uint32(505)
	templateId := uint32(1002357)

	l := testLogger()
	te := testTenant()
	ctx := tenant.WithContext(context.Background(), te)
	db := testDatabase(t, l)

	mb := message.NewBuffer()

	ap := asset.NewProcessor(l, ctx, db)
	tm := &tradeabilityMock.ProcessorMock{
		GetFunc: func(inventoryType inventory.Type, tid item.Id) (tradeability.Model, error) {
			return tradeability.NewModel(false, 1), nil
		},
	}
	cp := compartment.NewProcessor(l, ctx, db).WithAssetProcessor(ap).WithTradeabilityProcessor(tm)

	c, err := cp.Create(mb)(uuid.New(), characterId, inventory.TypeValueEquip, 24)
	if err != nil {
		t.Fatalf("Failed to create compartment: %v", err)
	}

	slot := int16(3)
	m := asset.NewBuilder(c.Id(), templateId).SetSlot(slot).SetCreatedAt(time.Now()).Build()
	if _, err := ap.CreateFromModel(mb)(uuid.New(), characterId, m); err != nil {
		t.Fatalf("Failed to create asset: %v", err)
	}

	if err := cp.ApplyAssetKarma(mb)(uuid.New(), characterId, inventory.TypeValueEquip, slot, 0, false); err == nil {
		t.Fatal("expected ApplyAssetKarma to refuse an already-tradeable asset")
	}
}

// TestApplyAssetKarmaRefusesPet is OQ-5: the pet karma bit aliases FlagLock.
// The tradeability lookup happens BEFORE the pet gate (compartment.ApplyAssetKarma
// resolves item data ahead of calling asset.ApplyKarma), so the mock must
// succeed here or the test would pass for the wrong reason (a failed lookup),
// not because the target is a pet. The assertion checks the error text names
// the pet-class gate specifically.
func TestApplyAssetKarmaRefusesPet(t *testing.T) {
	characterId := uint32(506)
	templateId := uint32(5000000)

	l := testLogger()
	te := testTenant()
	ctx := tenant.WithContext(context.Background(), te)
	db := testDatabase(t, l)

	mb := message.NewBuffer()

	ap := asset.NewProcessor(l, ctx, db)
	tm := &tradeabilityMock.ProcessorMock{
		GetFunc: func(inventoryType inventory.Type, tid item.Id) (tradeability.Model, error) {
			return tradeability.NewModel(true, 1), nil
		},
	}
	cp := compartment.NewProcessor(l, ctx, db).WithAssetProcessor(ap).WithTradeabilityProcessor(tm)

	c, err := cp.Create(mb)(uuid.New(), characterId, inventory.TypeValueCash, 24)
	if err != nil {
		t.Fatalf("Failed to create compartment: %v", err)
	}

	slot := int16(3)
	m := asset.NewBuilder(c.Id(), templateId).SetSlot(slot).SetCreatedAt(time.Now()).Build()
	if _, err := ap.CreateFromModel(mb)(uuid.New(), characterId, m); err != nil {
		t.Fatalf("Failed to create asset: %v", err)
	}

	err = cp.ApplyAssetKarma(mb)(uuid.New(), characterId, inventory.TypeValueCash, slot, 0, false)
	if err == nil {
		t.Fatal("expected ApplyAssetKarma to refuse a pet-class target")
	}
	if !strings.Contains(err.Error(), "pet") {
		t.Fatalf("expected the refusal to name the pet-class gate, got: %v", err)
	}
}

// TestApplyAssetKarmaRefusesUnreadableItemData: a failed lookup is a refusal,
// never a permissive default.
func TestApplyAssetKarmaRefusesUnreadableItemData(t *testing.T) {
	characterId := uint32(507)
	templateId := uint32(1002357)

	l := testLogger()
	te := testTenant()
	ctx := tenant.WithContext(context.Background(), te)
	db := testDatabase(t, l)

	mb := message.NewBuffer()

	ap := asset.NewProcessor(l, ctx, db)
	tm := &tradeabilityMock.ProcessorMock{
		GetFunc: func(inventoryType inventory.Type, tid item.Id) (tradeability.Model, error) {
			return tradeability.Model{}, errors.New("boom")
		},
	}
	cp := compartment.NewProcessor(l, ctx, db).WithAssetProcessor(ap).WithTradeabilityProcessor(tm)

	c, err := cp.Create(mb)(uuid.New(), characterId, inventory.TypeValueEquip, 24)
	if err != nil {
		t.Fatalf("Failed to create compartment: %v", err)
	}

	slot := int16(3)
	m := asset.NewBuilder(c.Id(), templateId).SetSlot(slot).SetCreatedAt(time.Now()).AddFlag(af.FlagUntradeable).Build()
	if _, err := ap.CreateFromModel(mb)(uuid.New(), characterId, m); err != nil {
		t.Fatalf("Failed to create asset: %v", err)
	}

	if err := cp.ApplyAssetKarma(mb)(uuid.New(), characterId, inventory.TypeValueEquip, slot, 0, false); err == nil {
		t.Fatal("expected ApplyAssetKarma to refuse when item data is unreadable")
	}
}

// TestApplyAssetKarmaClearRemovesTheBit is the compensation path.
func TestApplyAssetKarmaClearRemovesTheBit(t *testing.T) {
	characterId := uint32(508)
	templateId := uint32(1002357)

	l := testLogger()
	te := testTenant()
	ctx := tenant.WithContext(context.Background(), te)
	db := testDatabase(t, l)

	mb := message.NewBuffer()

	ap := asset.NewProcessor(l, ctx, db)
	tm := &tradeabilityMock.ProcessorMock{
		GetFunc: func(inventoryType inventory.Type, tid item.Id) (tradeability.Model, error) {
			return tradeability.NewModel(true, 1), nil
		},
	}
	cp := compartment.NewProcessor(l, ctx, db).WithAssetProcessor(ap).WithTradeabilityProcessor(tm)

	c, err := cp.Create(mb)(uuid.New(), characterId, inventory.TypeValueEquip, 24)
	if err != nil {
		t.Fatalf("Failed to create compartment: %v", err)
	}

	slot := int16(3)
	m := asset.NewBuilder(c.Id(), templateId).SetSlot(slot).SetCreatedAt(time.Now()).AddFlag(af.FlagUntradeable).AddFlag(af.FlagKarmaEquip).Build()
	if _, err := ap.CreateFromModel(mb)(uuid.New(), characterId, m); err != nil {
		t.Fatalf("Failed to create asset: %v", err)
	}

	if err := cp.ApplyAssetKarma(mb)(uuid.New(), characterId, inventory.TypeValueEquip, slot, 0, true); err != nil {
		t.Fatalf("clear returned unexpected error: %v", err)
	}

	a, err := ap.GetBySlot(c.Id(), slot)
	if err != nil {
		t.Fatalf("Failed to reload asset: %v", err)
	}
	if af.HasFlag(a.Flag(), af.FlagKarmaEquip) {
		t.Fatal("expected the karma bit to be cleared")
	}
	if !af.HasFlag(a.Flag(), af.FlagUntradeable) {
		t.Fatal("clearing karma disturbed FlagUntradeable")
	}
}
