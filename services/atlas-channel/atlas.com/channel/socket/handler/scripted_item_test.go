package handler

import (
	consumabledata "atlas-channel/data/consumable"
	"atlas-channel/saga"
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/sirupsen/logrus"
	testlog "github.com/sirupsen/logrus/hooks/test"

	"github.com/Chronicle20/atlas/libs/atlas-constants/inventory"
	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
)

// scriptedItemLogContains reports whether any entry the hook captured
// contains substr in its formatted message (precedent: testlog.NewNullLogger
// usage in npc_continue_conversation_test.go:39,69).
func scriptedItemLogContains(hook *testlog.Hook, substr string) bool {
	for _, e := range hook.AllEntries() {
		if strings.Contains(e.Message, substr) {
			return true
		}
	}
	return false
}

// installScriptedItemDataSeam swaps the atlas-data consumable read (same
// package-var injection precedent as installRemoteMerchantCashItemDataSeam).
func installScriptedItemDataSeam(t *testing.T, m consumabledata.Model, err error) func() {
	t.Helper()
	orig := scriptedItemDataFunc
	scriptedItemDataFunc = func(_ logrus.FieldLogger, _ context.Context, _ uint32) (consumabledata.Model, error) {
		return m, err
	}
	return func() {
		scriptedItemDataFunc = orig
	}
}

// installScriptedItemSagaSeam records created sagas instead of producing
// (precedent: installRemoteMerchantSagaSeam).
func installScriptedItemSagaSeam(t *testing.T) (*[]saga.Saga, func()) {
	t.Helper()
	var got []saga.Saga
	orig := scriptedItemSagaCreateFunc
	scriptedItemSagaCreateFunc = func(_ logrus.FieldLogger, _ context.Context, s saga.Saga) error {
		got = append(got, s)
		return nil
	}
	return &got, func() {
		scriptedItemSagaCreateFunc = orig
	}
}

// scriptedItemUsePayload encodes the wire layout of invsb.ScriptedItem:
// uint32 updateTime, int16 source, uint32 itemId (scripted_item.go's
// Decode order, mirrored from libs/atlas-packet/inventory/serverbound/scripted_item_test.go).
func scriptedItemUsePayload(updateTime uint32, source int16, itemId uint32) []byte {
	return []byte{
		byte(updateTime), byte(updateTime >> 8), byte(updateTime >> 16), byte(updateTime >> 24),
		byte(source), byte(source >> 8),
		byte(itemId), byte(itemId >> 8), byte(itemId >> 16), byte(itemId >> 24),
	}
}

// scriptedItemDataBuilder is a small builder for the consumable data fixture
// (Test Helper Pattern: builder-based setup, no *_testhelpers.go). Model has
// only unexported fields, so the builder routes through the same
// RestModel -> Extract path the real atlas-data processor uses.
type scriptedItemDataBuilder struct {
	rm consumabledata.RestModel
}

func newScriptedItemDataBuilder() *scriptedItemDataBuilder {
	return &scriptedItemDataBuilder{}
}

func (b *scriptedItemDataBuilder) SetNpc(npc uint32) *scriptedItemDataBuilder {
	b.rm.Npc = npc
	return b
}

func (b *scriptedItemDataBuilder) Build(t *testing.T) consumabledata.Model {
	t.Helper()
	m, err := consumabledata.Extract(b.rm)
	if err != nil {
		t.Fatalf("Extract: %v", err)
	}
	return m
}

// Every rejection path must unlock the client and consume nothing. The success
// path must NOT unlock: the destroy step's inventory delta is what clears the
// client's m_bExclRequestSent, and an explicit unlock as well would
// double-resolve the lock.

// itemId 2000000 is a 200-class consumable, not 243. Impossible from a
// legitimate client — the sender is gated on itemId/10000 == 243.
// Expect: no saga created, EnableActions called once.
func TestScriptedItem_RejectsNonScriptedClassification(t *testing.T) {
	const itemId = uint32(2000000)
	const srcSlot = int16(3)

	restoreSlot := installItemInSlotSeam(t, srcSlot, itemId)
	defer restoreSlot()
	sagas, restoreSaga := installScriptedItemSagaSeam(t)
	defer restoreSaga()

	s, ctx, cleanup := newCashItemUseTestSession(t, 555)
	defer cleanup()

	rec := &gaugeProducerRecorder{}
	req := request.Request(scriptedItemUsePayload(0, srcSlot, itemId))
	reader := request.NewRequestReader(&req, 0)
	ScriptedItemHandleFunc(logrus.New(), ctx, rec.producer())(s, &reader, map[string]interface{}{})

	if len(*sagas) != 0 {
		t.Errorf("sagas created = %d, want 0 for a non-scripted item", len(*sagas))
	}
	if rec.calls != 1 {
		t.Errorf("producer calls = %d, want exactly 1 (the enable-actions unlock)", rec.calls)
	}
}

// v95 alone whitelists 3994225 (Evolving Ring Upgrade Potion, an Install/Setup
// item). Supporting it needs setup/reader.go spec parsing plus a second
// inventory type on the destroy step, so it is a documented gap — but the
// rejection must NAME it, so a play-test report is self-explaining.
func TestScriptedItem_RejectsItem3994225ByName(t *testing.T) {
	const itemId = uint32(3994225)
	const srcSlot = int16(4)

	restoreSlot := installItemInSlotSeam(t, srcSlot, itemId)
	defer restoreSlot()
	sagas, restoreSaga := installScriptedItemSagaSeam(t)
	defer restoreSaga()

	s, ctx, cleanup := newCashItemUseTestSession(t, 556)
	defer cleanup()

	rec := &gaugeProducerRecorder{}

	logger, hook := testlog.NewNullLogger()
	req := request.Request(scriptedItemUsePayload(0, srcSlot, itemId))
	reader := request.NewRequestReader(&req, 0)
	ScriptedItemHandleFunc(logger, ctx, rec.producer())(s, &reader, map[string]interface{}{})

	if len(*sagas) != 0 {
		t.Errorf("sagas created = %d, want 0 for item 3994225", len(*sagas))
	}
	if rec.calls != 1 {
		t.Errorf("producer calls = %d, want exactly 1 (the enable-actions unlock)", rec.calls)
	}
	if !scriptedItemLogContains(hook, "3994225") {
		t.Errorf("expected a log line naming item 3994225 explicitly, got entries: %+v", hook.AllEntries())
	}
}

// GetItemInSlot returns a different template than the packet claims.
// Expect: no saga, EnableActions called once.
func TestScriptedItem_RejectsSlotTemplateMismatch(t *testing.T) {
	const itemId = uint32(2430008)
	const srcSlot = int16(3)

	// Slot 3 holds a different template id than the packet claims.
	restoreSlot := installItemInSlotSeam(t, srcSlot, itemId+1)
	defer restoreSlot()
	sagas, restoreSaga := installScriptedItemSagaSeam(t)
	defer restoreSaga()

	s, ctx, cleanup := newCashItemUseTestSession(t, 557)
	defer cleanup()

	rec := &gaugeProducerRecorder{}
	req := request.Request(scriptedItemUsePayload(0, srcSlot, itemId))
	reader := request.NewRequestReader(&req, 0)
	ScriptedItemHandleFunc(logrus.New(), ctx, rec.producer())(s, &reader, map[string]interface{}{})

	if len(*sagas) != 0 {
		t.Errorf("sagas created = %d, want 0 on slot/template mismatch", len(*sagas))
	}
	if rec.calls != 1 {
		t.Errorf("producer calls = %d, want exactly 1 (the enable-actions unlock)", rec.calls)
	}
}

// Until the atlas-data re-ingest lands, every tenant is in exactly this state.
// The log line must say so rather than presenting as a mysterious content gap.
func TestScriptedItem_RejectsNpcZeroAndNamesReingest(t *testing.T) {
	const itemId = uint32(2430008)
	const srcSlot = int16(3)

	restoreSlot := installItemInSlotSeam(t, srcSlot, itemId)
	defer restoreSlot()
	restoreData := installScriptedItemDataSeam(t, newScriptedItemDataBuilder().SetNpc(0).Build(t), nil)
	defer restoreData()
	sagas, restoreSaga := installScriptedItemSagaSeam(t)
	defer restoreSaga()

	s, ctx, cleanup := newCashItemUseTestSession(t, 558)
	defer cleanup()

	rec := &gaugeProducerRecorder{}
	logger, hook := testlog.NewNullLogger()
	req := request.Request(scriptedItemUsePayload(0, srcSlot, itemId))
	reader := request.NewRequestReader(&req, 0)
	ScriptedItemHandleFunc(logger, ctx, rec.producer())(s, &reader, map[string]interface{}{})

	if len(*sagas) != 0 {
		t.Errorf("sagas created = %d, want 0 when npc == 0", len(*sagas))
	}
	if rec.calls != 1 {
		t.Errorf("producer calls = %d, want exactly 1 (the enable-actions unlock)", rec.calls)
	}
	if !scriptedItemLogContains(hook, "re-ingest") {
		t.Errorf("expected the npc==0 warn to name re-ingest as the likely cause, got entries: %+v", hook.AllEntries())
	}
}

// The happy path builds exactly two steps in this order. Conversation FIRST:
// an item with no authored conversation gets START_ERROR, the destroy never
// runs, and the player keeps the item — no rollback required.
func TestScriptedItem_CreatesConversationThenDestroySaga(t *testing.T) {
	const itemId = uint32(2430008)
	const srcSlot = int16(3)
	const npcTemplateId = uint32(9010000)
	const charId = uint32(559)

	restoreSlot := installItemInSlotSeam(t, srcSlot, itemId)
	defer restoreSlot()
	restoreData := installScriptedItemDataSeam(t, newScriptedItemDataBuilder().SetNpc(npcTemplateId).Build(t), nil)
	defer restoreData()
	sagas, restoreSaga := installScriptedItemSagaSeam(t)
	defer restoreSaga()

	s, ctx, cleanup := newCashItemUseTestSession(t, charId)
	defer cleanup()

	rec := &gaugeProducerRecorder{}
	req := request.Request(scriptedItemUsePayload(0, srcSlot, itemId))
	reader := request.NewRequestReader(&req, 0)
	ScriptedItemHandleFunc(logrus.New(), ctx, rec.producer())(s, &reader, map[string]interface{}{})

	if len(*sagas) != 1 {
		t.Fatalf("sagas created = %d, want 1", len(*sagas))
	}
	sg := (*sagas)[0]
	if sg.SagaType != saga.ScriptedItemUse {
		t.Errorf("SagaType = %q, want %q", sg.SagaType, saga.ScriptedItemUse)
	}
	if len(sg.Steps) != 2 {
		t.Fatalf("steps = %d, want 2", len(sg.Steps))
	}

	if sg.Steps[0].Action != saga.StartItemConversation {
		t.Errorf("step 0 action = %q, want %q", sg.Steps[0].Action, saga.StartItemConversation)
	}
	cp, ok := sg.Steps[0].Payload.(saga.StartItemConversationPayload)
	if !ok {
		t.Fatalf("step 0 payload type = %T", sg.Steps[0].Payload)
	}
	if cp.ItemId != itemId || cp.NpcTemplateId != npcTemplateId || cp.Slot != srcSlot || cp.CharacterId != charId {
		t.Errorf("step 0 payload = %+v", cp)
	}

	if sg.Steps[1].Action != saga.DestroyAssetFromSlot {
		t.Errorf("step 1 action = %q, want %q", sg.Steps[1].Action, saga.DestroyAssetFromSlot)
	}
	dp, ok := sg.Steps[1].Payload.(saga.DestroyAssetFromSlotPayload)
	if !ok {
		t.Fatalf("step 1 payload type = %T", sg.Steps[1].Payload)
	}
	if dp.InventoryType != byte(inventory.TypeValueUse) || dp.Quantity != 1 || dp.TemplateId != itemId || dp.Slot != srcSlot {
		t.Errorf("step 1 payload = %+v", dp)
	}

	if rec.calls != 0 {
		t.Errorf("producer calls = %d, want 0 — the success path must not unlock (destroy step clears m_bExclRequestSent)", rec.calls)
	}
}

// scriptedItemSagaCreateFunc returns an error.
// Expect: EnableActions called once.
func TestScriptedItem_UnlocksWhenSagaCreationFails(t *testing.T) {
	const itemId = uint32(2430008)
	const srcSlot = int16(3)

	restoreSlot := installItemInSlotSeam(t, srcSlot, itemId)
	defer restoreSlot()
	restoreData := installScriptedItemDataSeam(t, newScriptedItemDataBuilder().SetNpc(9010000).Build(t), nil)
	defer restoreData()

	orig := scriptedItemSagaCreateFunc
	scriptedItemSagaCreateFunc = func(_ logrus.FieldLogger, _ context.Context, _ saga.Saga) error {
		return errors.New("boom")
	}
	defer func() { scriptedItemSagaCreateFunc = orig }()

	s, ctx, cleanup := newCashItemUseTestSession(t, 560)
	defer cleanup()

	rec := &gaugeProducerRecorder{}
	req := request.Request(scriptedItemUsePayload(0, srcSlot, itemId))
	reader := request.NewRequestReader(&req, 0)
	ScriptedItemHandleFunc(logrus.New(), ctx, rec.producer())(s, &reader, map[string]interface{}{})

	if rec.calls != 1 {
		t.Errorf("producer calls = %d, want exactly 1 (the enable-actions unlock)", rec.calls)
	}
}
