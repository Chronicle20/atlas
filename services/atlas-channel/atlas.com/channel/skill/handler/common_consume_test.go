package handler

import (
	"atlas-channel/asset"
	"atlas-channel/character"
	"atlas-channel/compartment"
	"atlas-channel/consumable"
	"atlas-channel/data/skill/effect"
	"atlas-channel/inventory"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-constants/channel"
	charcon "github.com/Chronicle20/atlas/libs/atlas-constants/character"
	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	inventoryconst "github.com/Chronicle20/atlas/libs/atlas-constants/inventory"
	"github.com/Chronicle20/atlas/libs/atlas-constants/inventory/slot"
	itemconst "github.com/Chronicle20/atlas/libs/atlas-constants/item"
	_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"
	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
	packetmodel "github.com/Chronicle20/atlas/libs/atlas-packet/model"
)

// testConsumeSkillId is an arbitrary id NOT equal to NightLordShadowStarsId and
// with no mount/registry/dispatcher classification, so UseSkill exercises only
// the itemCon path (the Shadow Stars pre-flight and the mob/mount/dispatcher
// paths are all inert for it).
const testConsumeSkillId = uint32(99999999)

// magicRockItemId is Magic Rock (a real itemCon for several casts, e.g. Echo of
// Hero). classification 400 -> ETC compartment via inventoryconst.TypeFromItemId.
const magicRockItemId = uint32(4006000)

type consumeRecorder struct {
	called   bool
	itemId   itemconst.Id
	source   slot.Position
	quantity int16
}

// overrideConsumeSeams replaces the two itemCon seams with recorders and
// restores them via t.Cleanup. loadCasterWithInventoryFunc returns the given
// caster (or loadErr).
func overrideConsumeSeams(t *testing.T, c character.Model, loadErr error) *consumeRecorder {
	t.Helper()
	rec := &consumeRecorder{}

	prevLoad := loadCasterWithInventoryFunc
	loadCasterWithInventoryFunc = func(_ character.Processor, _ uint32) (character.Model, error) {
		return c, loadErr
	}
	prevConsume := requestItemConsumeFunc
	requestItemConsumeFunc = func(_ consumable.Processor, _ field.Model, _ charcon.Id, itemId itemconst.Id, source slot.Position, quantity int16, _ uint32) error {
		rec.called = true
		rec.itemId = itemId
		rec.source = source
		rec.quantity = quantity
		return nil
	}
	t.Cleanup(func() {
		loadCasterWithInventoryFunc = prevLoad
		requestItemConsumeFunc = prevConsume
	})
	return rec
}

func testField() field.Model {
	return field.NewBuilder(world.Id(0), channel.Id(1), _map.Id(100000000)).Build()
}

func consumeAsset(t *testing.T, slotIdx int16, templateId uint32, qty uint32) asset.Model {
	t.Helper()
	a, err := asset.NewBuilderWithId(uint32(slotIdx), uuid.New(), templateId).SetSlot(slotIdx).SetQuantity(qty).Build()
	if err != nil {
		t.Fatalf("asset build: %v", err)
	}
	return a
}

// casterWithCompartment builds a character whose compartment for the given
// inventory type holds the provided assets.
func casterWithCompartment(t *testing.T, it inventoryconst.Type, assets ...asset.Model) character.Model {
	t.Helper()
	cb := compartment.NewBuilder(uuid.New(), 1, it, 96)
	for _, a := range assets {
		cb.AddAsset(a)
	}
	cm, err := cb.Build()
	if err != nil {
		t.Fatalf("compartment build: %v", err)
	}
	inv := inventory.NewBuilder(1).SetCompartment(cm).MustBuild()
	return character.NewBuilder().SetId(1).SetInventory(inv).MustBuild()
}

func consumeEffect(t *testing.T, itemConsume uint32, amount uint32) effect.Model {
	t.Helper()
	e, err := effect.Extract(effect.RestModel{
		ItemConsume:       itemConsume,
		ItemConsumeAmount: amount,
	})
	if err != nil {
		t.Fatalf("effect extract: %v", err)
	}
	return e
}

func useSkillInfo() packetmodel.SkillUsageInfo {
	return packetmodel.NewSkillUsageInfoBuilder().SetSkillId(testConsumeSkillId).SetSkillLevel(1).Build()
}

func runUseSkill(t *testing.T, e effect.Model) {
	t.Helper()
	ctx, _ := newCtx(t)
	if err := UseSkill(logrus.New())(ctx)(nil, testField(), 1, useSkillInfo(), e); err != nil {
		t.Fatalf("UseSkill: %v", err)
	}
}

func TestUseSkill_ItemConsumeAmountPlumbed(t *testing.T) {
	it, _ := inventoryconst.TypeFromItemId(itemconst.Id(magicRockItemId))
	// Slot 1 is short (1 < 2); slot 2 qualifies -> quantity 2 from slot 2.
	c := casterWithCompartment(t, it, consumeAsset(t, 1, magicRockItemId, 1), consumeAsset(t, 2, magicRockItemId, 3))
	rec := overrideConsumeSeams(t, c, nil)

	runUseSkill(t, consumeEffect(t, magicRockItemId, 2))

	if !rec.called {
		t.Fatal("expected a consume request")
	}
	if rec.quantity != 2 {
		t.Errorf("quantity: got %d, want 2", rec.quantity)
	}
	if rec.source != slot.Position(2) {
		t.Errorf("source slot: got %d, want 2", rec.source)
	}
	if rec.itemId != itemconst.Id(magicRockItemId) {
		t.Errorf("itemId: got %d, want %d", rec.itemId, magicRockItemId)
	}
}

func TestUseSkill_ItemConsumeAmountZeroFloorsToOne(t *testing.T) {
	it, _ := inventoryconst.TypeFromItemId(itemconst.Id(magicRockItemId))
	c := casterWithCompartment(t, it, consumeAsset(t, 1, magicRockItemId, 5))
	rec := overrideConsumeSeams(t, c, nil)

	runUseSkill(t, consumeEffect(t, magicRockItemId, 0))

	if !rec.called || rec.quantity != 1 {
		t.Fatalf("got (called=%v, quantity=%d), want (true, 1)", rec.called, rec.quantity)
	}
}

func TestUseSkill_ItemConShortfallSkipsButCastProceeds(t *testing.T) {
	it, _ := inventoryconst.TypeFromItemId(itemconst.Id(magicRockItemId))
	// Two slots of 1 each; no single slot holds 2 -> skip, warn, no error (cast proceeds).
	c := casterWithCompartment(t, it, consumeAsset(t, 1, magicRockItemId, 1), consumeAsset(t, 2, magicRockItemId, 1))
	rec := overrideConsumeSeams(t, c, nil)

	runUseSkill(t, consumeEffect(t, magicRockItemId, 2)) // no t.Fatalf => UseSkill returned nil

	if rec.called {
		t.Error("expected NO consume request on shortfall")
	}
}

func TestUseSkill_ItemConCasterLoadFailureCastProceeds(t *testing.T) {
	rec := overrideConsumeSeams(t, character.Model{}, errors.New("boom"))

	runUseSkill(t, consumeEffect(t, magicRockItemId, 2)) // no t.Fatalf => UseSkill returned nil

	if rec.called {
		t.Error("expected NO consume request on load failure")
	}
}
