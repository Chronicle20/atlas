package handler

import (
	"atlas-channel/data/cash"
	"atlas-channel/remotemerchant"
	"atlas-channel/saga"
	"context"
	"errors"
	"testing"

	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-constants/inventory"
	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

// installNpcShopProbeSeam swaps npcShopProbeFunc for the test and returns a
// restore func (same package-var injection precedent as itemInSlotFunc /
// scriptedItemSagaCreateFunc). err == nil means the probed npcTemplateId has
// a shop.
func installNpcShopProbeSeam(t *testing.T, err error) func() {
	t.Helper()
	orig := npcShopProbeFunc
	npcShopProbeFunc = func(_ logrus.FieldLogger, _ context.Context, _ uint32) error {
		return err
	}
	return func() {
		npcShopProbeFunc = orig
	}
}

// installNpcItemUseSagaSeam records created sagas instead of producing
// (precedent: installScriptedItemSagaSeam / installRemoteMerchantSagaSeam).
func installNpcItemUseSagaSeam(t *testing.T) (*[]saga.Saga, func()) {
	t.Helper()
	var got []saga.Saga
	orig := npcItemUseSagaCreateFunc
	npcItemUseSagaCreateFunc = func(_ logrus.FieldLogger, _ context.Context, s saga.Saga) error {
		got = append(got, s)
		return nil
	}
	return &got, func() {
		npcItemUseSagaCreateFunc = orig
	}
}

// npcItemUsePayload encodes the wire layout of invsb.NpcItemUse: int16
// source, uint32 itemId — NO leading updateTime (contrast
// scriptedItemUsePayload in scripted_item_test.go, whose sibling opcode has
// one).
func npcItemUsePayload(source int16, itemId uint32) []byte {
	return []byte{
		byte(source), byte(source >> 8),
		byte(itemId), byte(itemId >> 8), byte(itemId >> 16), byte(itemId >> 24),
	}
}

// 239 with a shop: open_npc_shop then destroy, from the USE inventory.
func TestNpcItemUse_RemoteNpcWithShopOpensShop(t *testing.T) {
	const itemId = uint32(2390000)
	const srcSlot = int16(3)
	const npcTemplateId = uint32(9090000)
	const charId = uint32(601)

	restoreSlot := installItemInSlotSeam(t, srcSlot, itemId)
	defer restoreSlot()
	restoreData := installScriptedItemDataSeam(t, newScriptedItemDataBuilder().SetNpc(npcTemplateId).Build(t), nil)
	defer restoreData()
	restoreProbe := installNpcShopProbeSeam(t, nil)
	defer restoreProbe()
	sagas, restoreSaga := installNpcItemUseSagaSeam(t)
	defer restoreSaga()

	s, ctx, cleanup := newCashItemUseTestSession(t, charId)
	defer cleanup()

	rec := &gaugeProducerRecorder{}
	req := request.Request(npcItemUsePayload(srcSlot, itemId))
	reader := request.NewRequestReader(&req, 0)
	NpcItemUseHandleFunc(logrus.New(), ctx, rec.producer())(s, &reader, map[string]interface{}{})

	if len(*sagas) != 1 {
		t.Fatalf("sagas created = %d, want 1", len(*sagas))
	}
	sg := (*sagas)[0]
	if sg.SagaType != saga.RemoteNpcUse {
		t.Errorf("SagaType = %q, want %q", sg.SagaType, saga.RemoteNpcUse)
	}
	if len(sg.Steps) != 2 {
		t.Fatalf("steps = %d, want 2", len(sg.Steps))
	}
	if sg.Steps[0].Action != saga.OpenNpcShop {
		t.Errorf("step 0 action = %q, want open_npc_shop", sg.Steps[0].Action)
	}
	op, ok := sg.Steps[0].Payload.(saga.OpenNpcShopPayload)
	if !ok || op.NpcTemplateId != npcTemplateId || op.CharacterId != charId {
		t.Errorf("step 0 payload = %+v", sg.Steps[0].Payload)
	}
	if sg.Steps[1].Action != saga.DestroyAssetFromSlot {
		t.Errorf("step 1 action = %q, want destroy_asset_from_slot", sg.Steps[1].Action)
	}
	dp, ok := sg.Steps[1].Payload.(saga.DestroyAssetFromSlotPayload)
	if !ok || dp.InventoryType != byte(inventory.TypeValueUse) || dp.Slot != srcSlot || dp.Quantity != 1 || dp.TemplateId != itemId {
		t.Errorf("step 1 payload = %+v", sg.Steps[1].Payload)
	}
	if rec.calls != 0 {
		t.Errorf("producer calls = %d, want 0 — the success path must not unlock", rec.calls)
	}
}

// 239 without a shop: start_npc_conversation then destroy.
func TestNpcItemUse_RemoteNpcWithoutShopStartsConversation(t *testing.T) {
	const itemId = uint32(2390000)
	const srcSlot = int16(3)
	const npcTemplateId = uint32(9010000)
	const charId = uint32(602)

	restoreSlot := installItemInSlotSeam(t, srcSlot, itemId)
	defer restoreSlot()
	restoreData := installScriptedItemDataSeam(t, newScriptedItemDataBuilder().SetNpc(npcTemplateId).Build(t), nil)
	defer restoreData()
	restoreProbe := installNpcShopProbeSeam(t, errors.New("not found"))
	defer restoreProbe()
	sagas, restoreSaga := installNpcItemUseSagaSeam(t)
	defer restoreSaga()

	s, ctx, cleanup := newCashItemUseTestSession(t, charId)
	defer cleanup()

	rec := &gaugeProducerRecorder{}
	req := request.Request(npcItemUsePayload(srcSlot, itemId))
	reader := request.NewRequestReader(&req, 0)
	NpcItemUseHandleFunc(logrus.New(), ctx, rec.producer())(s, &reader, map[string]interface{}{})

	if len(*sagas) != 1 {
		t.Fatalf("sagas created = %d, want 1", len(*sagas))
	}
	sg := (*sagas)[0]
	if sg.SagaType != saga.RemoteNpcUse {
		t.Errorf("SagaType = %q, want %q", sg.SagaType, saga.RemoteNpcUse)
	}
	if len(sg.Steps) != 2 {
		t.Fatalf("steps = %d, want 2", len(sg.Steps))
	}
	if sg.Steps[0].Action != saga.StartNpcConversation {
		t.Errorf("step 0 action = %q, want start_npc_conversation", sg.Steps[0].Action)
	}
	cp, ok := sg.Steps[0].Payload.(saga.StartNpcConversationPayload)
	if !ok || cp.NpcTemplateId != npcTemplateId || cp.CharacterId != charId {
		t.Errorf("step 0 payload = %+v", sg.Steps[0].Payload)
	}
	if sg.Steps[1].Action != saga.DestroyAssetFromSlot {
		t.Errorf("step 1 action = %q, want destroy_asset_from_slot", sg.Steps[1].Action)
	}
	dp, ok := sg.Steps[1].Payload.(saga.DestroyAssetFromSlotPayload)
	if !ok || dp.InventoryType != byte(inventory.TypeValueUse) || dp.Slot != srcSlot || dp.Quantity != 1 || dp.TemplateId != itemId {
		t.Errorf("step 1 payload = %+v", sg.Steps[1].Payload)
	}
	if rec.calls != 0 {
		t.Errorf("producer calls = %d, want 0 — the success path must not unlock", rec.calls)
	}
}

// 545 always takes the shop path, from the CASH inventory.
func TestNpcItemUse_RemoteMerchantOpensShopFromCashInventory(t *testing.T) {
	const itemId = uint32(5450000)
	const srcSlot = int16(4)
	const npcTemplateId = uint32(9090000)
	const charId = uint32(603)

	restoreSlot := installCashItemInSlotSeam(t, srcSlot, itemId)
	defer restoreSlot()
	restoreData := installRemoteMerchantCashItemDataSeam(t, cash.RestModel{Id: itemId, Npc: npcTemplateId}, nil)
	defer restoreData()
	sagas, restoreSaga := installNpcItemUseSagaSeam(t)
	defer restoreSaga()

	s, ctx, cleanup := newCashItemUseTestSession(t, charId)
	defer cleanup()

	rec := &gaugeProducerRecorder{}
	req := request.Request(npcItemUsePayload(srcSlot, itemId))
	reader := request.NewRequestReader(&req, 0)
	NpcItemUseHandleFunc(logrus.New(), ctx, rec.producer())(s, &reader, map[string]interface{}{})

	if len(*sagas) != 1 {
		t.Fatalf("sagas created = %d, want 1", len(*sagas))
	}
	sg := (*sagas)[0]
	if sg.SagaType != saga.RemoteMerchant {
		t.Errorf("SagaType = %q, want %q", sg.SagaType, saga.RemoteMerchant)
	}
	if len(sg.Steps) != 2 {
		t.Fatalf("steps = %d, want 2", len(sg.Steps))
	}
	if sg.Steps[0].Action != saga.OpenNpcShop {
		t.Errorf("step 0 action = %q, want open_npc_shop", sg.Steps[0].Action)
	}
	op, ok := sg.Steps[0].Payload.(saga.OpenNpcShopPayload)
	if !ok || op.NpcTemplateId != npcTemplateId || op.CharacterId != charId {
		t.Errorf("step 0 payload = %+v", sg.Steps[0].Payload)
	}
	if sg.Steps[1].Action != saga.DestroyAssetFromSlot {
		t.Errorf("step 1 action = %q, want destroy_asset_from_slot", sg.Steps[1].Action)
	}
	dp, ok := sg.Steps[1].Payload.(saga.DestroyAssetFromSlotPayload)
	if !ok || dp.InventoryType != 5 || dp.Slot != srcSlot || dp.Quantity != 1 || dp.TemplateId != itemId {
		t.Errorf("step 1 payload = %+v", sg.Steps[1].Payload)
	}

	ten := tenant.MustFromContext(ctx)
	if _, ok := remotemerchant.GetRegistry().Take(ten, charId); !ok {
		t.Error("no pending unlock registered")
	}
	if rec.calls != 0 {
		t.Errorf("producer calls = %d, want 0 — the success path must not unlock", rec.calls)
	}
}

// An unhandled classification is impossible from a legitimate client — the
// sender gates on itemId/10000 in {239, 545}.
func TestNpcItemUse_RejectsOtherClassifications(t *testing.T) {
	const itemId = uint32(2000000)
	const srcSlot = int16(3)

	restoreSlot := installItemInSlotSeam(t, srcSlot, itemId)
	defer restoreSlot()
	sagas, restoreSaga := installNpcItemUseSagaSeam(t)
	defer restoreSaga()

	s, ctx, cleanup := newCashItemUseTestSession(t, 604)
	defer cleanup()

	rec := &gaugeProducerRecorder{}
	req := request.Request(npcItemUsePayload(srcSlot, itemId))
	reader := request.NewRequestReader(&req, 0)
	NpcItemUseHandleFunc(logrus.New(), ctx, rec.producer())(s, &reader, map[string]interface{}{})

	if len(*sagas) != 0 {
		t.Errorf("sagas created = %d, want 0 for an unhandled classification", len(*sagas))
	}
	if rec.calls != 1 {
		t.Errorf("producer calls = %d, want exactly 1 (the enable-actions unlock)", rec.calls)
	}
}

func TestNpcItemUse_RejectsSlotTemplateMismatch(t *testing.T) {
	const itemId = uint32(2390000)
	const srcSlot = int16(3)

	// Slot 3 holds a different template than the packet claims.
	restoreSlot := installItemInSlotSeam(t, srcSlot, itemId+1)
	defer restoreSlot()
	sagas, restoreSaga := installNpcItemUseSagaSeam(t)
	defer restoreSaga()

	s, ctx, cleanup := newCashItemUseTestSession(t, 605)
	defer cleanup()

	rec := &gaugeProducerRecorder{}
	req := request.Request(npcItemUsePayload(srcSlot, itemId))
	reader := request.NewRequestReader(&req, 0)
	NpcItemUseHandleFunc(logrus.New(), ctx, rec.producer())(s, &reader, map[string]interface{}{})

	if len(*sagas) != 0 {
		t.Errorf("sagas created = %d, want 0 on slot/template mismatch", len(*sagas))
	}
	if rec.calls != 1 {
		t.Errorf("producer calls = %d, want exactly 1 (the enable-actions unlock)", rec.calls)
	}
}

func TestNpcItemUse_RejectsNpcZero(t *testing.T) {
	const itemId = uint32(2390000)
	const srcSlot = int16(3)

	restoreSlot := installItemInSlotSeam(t, srcSlot, itemId)
	defer restoreSlot()
	restoreData := installScriptedItemDataSeam(t, newScriptedItemDataBuilder().SetNpc(0).Build(t), nil)
	defer restoreData()
	sagas, restoreSaga := installNpcItemUseSagaSeam(t)
	defer restoreSaga()

	s, ctx, cleanup := newCashItemUseTestSession(t, 606)
	defer cleanup()

	rec := &gaugeProducerRecorder{}
	req := request.Request(npcItemUsePayload(srcSlot, itemId))
	reader := request.NewRequestReader(&req, 0)
	NpcItemUseHandleFunc(logrus.New(), ctx, rec.producer())(s, &reader, map[string]interface{}{})

	if len(*sagas) != 0 {
		t.Errorf("sagas created = %d, want 0 when npc == 0", len(*sagas))
	}
	if rec.calls != 1 {
		t.Errorf("producer calls = %d, want exactly 1 (the enable-actions unlock)", rec.calls)
	}
}

func TestNpcItemUse_UnlocksWhenSagaCreationFails(t *testing.T) {
	const itemId = uint32(2390000)
	const srcSlot = int16(3)

	restoreSlot := installItemInSlotSeam(t, srcSlot, itemId)
	defer restoreSlot()
	restoreData := installScriptedItemDataSeam(t, newScriptedItemDataBuilder().SetNpc(9010000).Build(t), nil)
	defer restoreData()
	restoreProbe := installNpcShopProbeSeam(t, errors.New("not found"))
	defer restoreProbe()

	orig := npcItemUseSagaCreateFunc
	npcItemUseSagaCreateFunc = func(_ logrus.FieldLogger, _ context.Context, _ saga.Saga) error {
		return errors.New("boom")
	}
	defer func() { npcItemUseSagaCreateFunc = orig }()

	s, ctx, cleanup := newCashItemUseTestSession(t, 607)
	defer cleanup()

	rec := &gaugeProducerRecorder{}
	req := request.Request(npcItemUsePayload(srcSlot, itemId))
	reader := request.NewRequestReader(&req, 0)
	NpcItemUseHandleFunc(logrus.New(), ctx, rec.producer())(s, &reader, map[string]interface{}{})

	if rec.calls != 1 {
		t.Errorf("producer calls = %d, want exactly 1 (the enable-actions unlock)", rec.calls)
	}
}

// Both routes must stay live on v72-v95: CDraggableItem::OnDoubleClicked
// decides whether a 545 item goes out as CASH_ITEM_USE or
// NPC_ITEM_USE_REQUEST, so neither handler may assume it is the only path.
//
// A v61 tenant — for which remoteMerchantEnabled() is deliberately false,
// because 545 sits in that version's CASH_ITEM_USE dispatcher default arm —
// must still open the shop through THIS route, since NPC_ITEM_USE_REQUEST is
// v61's only path for a 545 item.
func TestNpcItemUse_RemoteMerchantDoesNotDependOnRemoteMerchantEnabled(t *testing.T) {
	const itemId = uint32(5450000)
	const srcSlot = int16(3)
	const npcTemplateId = uint32(9090000)
	const charId = uint32(608)

	// Sanity: remoteMerchantEnabled is false for this version, so a pass here
	// cannot be explained by that predicate leaking into this route.
	if remoteMerchantEnabled(mustTenant(t, "GMS", 61, 1)) {
		t.Fatal("test setup invalid: remoteMerchantEnabled(GMS 61) must be false")
	}

	restoreSlot := installCashItemInSlotSeam(t, srcSlot, itemId)
	defer restoreSlot()
	restoreData := installRemoteMerchantCashItemDataSeam(t, cash.RestModel{Id: itemId, Npc: npcTemplateId}, nil)
	defer restoreData()
	sagas, restoreSaga := installNpcItemUseSagaSeam(t)
	defer restoreSaga()

	s, ctx, cleanup := newCashItemUseTestSessionForVersion(t, charId, "GMS", 61)
	defer cleanup()

	rec := &gaugeProducerRecorder{}
	req := request.Request(npcItemUsePayload(srcSlot, itemId))
	reader := request.NewRequestReader(&req, 0)
	NpcItemUseHandleFunc(logrus.New(), ctx, rec.producer())(s, &reader, map[string]interface{}{})

	if len(*sagas) != 1 {
		t.Fatalf("sagas created = %d, want 1 on v61 — this route does not gate on remoteMerchantEnabled", len(*sagas))
	}
	sg := (*sagas)[0]
	if sg.SagaType != saga.RemoteMerchant {
		t.Errorf("SagaType = %q, want %q", sg.SagaType, saga.RemoteMerchant)
	}
	if sg.Steps[0].Action != saga.OpenNpcShop {
		t.Errorf("step 0 action = %q, want open_npc_shop", sg.Steps[0].Action)
	}
}
