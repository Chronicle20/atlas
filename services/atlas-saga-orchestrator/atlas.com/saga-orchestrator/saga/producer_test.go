package saga

import (
	asset2 "atlas-saga-orchestrator/kafka/message/asset"
	"atlas-saga-orchestrator/kafka/message/broadcast"
	incubator2 "atlas-saga-orchestrator/kafka/message/incubator"
	"atlas-saga-orchestrator/kafka/message/megaphone"
	"atlas-saga-orchestrator/kafka/message/npc"
	npcshop "atlas-saga-orchestrator/kafka/message/npcshop"
	"atlas-saga-orchestrator/kafka/message/saga"
	"context"
	"encoding/json"
	"testing"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"github.com/sirupsen/logrus/hooks/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Chronicle20/atlas/libs/atlas-kafka/producer"

	"github.com/Chronicle20/atlas/libs/atlas-constants/channel"
	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
)

// buildTakeHomeSaga builds a completed WithdrawFromMts saga in its post-expansion
// shape: release_from_mts_holding (ReleaseFromMtsHolding) + accept_to_character
// (AcceptToCharacter). The high-level WithdrawFromMts step is replaced by these
// two during expansion, so this is what the saga looks like at COMPLETED.
func buildTakeHomeSaga(t *testing.T, characterId uint32, templateId uint32) Saga {
	t.Helper()
	txId := uuid.New()
	holdingId := uuid.New()
	s, err := NewBuilder().
		SetTransactionId(txId).
		SetSagaType(MtsOperation).
		SetInitiatedBy("take-home-test").
		AddStep("release_from_mts_holding", Completed, ReleaseFromMtsHolding, ReleaseFromMtsHoldingPayload{
			TransactionId: txId,
			HoldingId:     holdingId,
		}).
		AddStep("accept_to_character", Completed, AcceptToCharacter, AcceptToCharacterPayload{
			TransactionId: txId,
			CharacterId:   characterId,
			InventoryType: 1,
			TemplateId:    templateId,
			AssetData:     asset2.AssetData{Quantity: 1},
		}).
		Build()
	require.NoError(t, err)
	return s
}

// TestExtractMtsTakeHomeResults_PopulatesCharacterAndTemplate proves a completed
// take-home saga yields a Results map marked mts_take_home and carrying the
// originating characterId + templateId (so the channel can target the session).
func TestExtractMtsTakeHomeResults_PopulatesCharacterAndTemplate(t *testing.T) {
	const characterId = uint32(1001)
	const templateId = uint32(1402001)
	s := buildTakeHomeSaga(t, characterId, templateId)

	results := extractMtsTakeHomeResults(s)
	require.NotNil(t, results, "a completed WithdrawFromMts saga must yield take-home results")
	require.Equal(t, MtsTakeHomeResultKind, results["kind"])
	require.Equal(t, characterId, results["characterId"])
	require.Equal(t, templateId, results["templateId"])
}

// TestExtractMtsTakeHomeResults_NotTakeHome proves a non-take-home MtsOperation
// saga (e.g. a settle that moves a listing to a holding, no ReleaseFromMtsHolding)
// is NOT misclassified as take-home — the channel must not fire MoveItcPurchaseItemLtoSDone.
func TestExtractMtsTakeHomeResults_NotTakeHome(t *testing.T) {
	txId := uuid.New()
	s, err := NewBuilder().
		SetTransactionId(txId).
		SetSagaType(MtsOperation).
		SetInitiatedBy("settle-test").
		AddStep("mts_move_listing_to_holding", Completed, MtsMoveListingToHolding, MtsMoveListingToHoldingPayload{
			TransactionId: txId,
			ListingId:     uuid.New(),
			BuyerId:       5,
			WorldId:       0,
		}).
		Build()
	require.NoError(t, err)

	require.Nil(t, extractMtsTakeHomeResults(s), "a settle saga (no ReleaseFromMtsHolding) must not be classified take-home")
}

// TestExtractMtsTakeHomeResults_WrongSagaType proves a non-MtsOperation saga is
// never classified as take-home even if it somehow contained a release step.
func TestExtractMtsTakeHomeResults_WrongSagaType(t *testing.T) {
	s, err := NewBuilder().SetSagaType(CharacterCreation).SetInitiatedBy("test").Build()
	require.NoError(t, err)
	require.Nil(t, extractMtsTakeHomeResults(s))
}

// TestCompletedStatusEventProvider_TakeHomeBodyRoundTrips proves the COMPLETED
// event the orchestrator emits for a take-home saga carries SagaType + the
// take-home Results, and survives a JSON round-trip (characterId becomes float64,
// which the channel's resultUint32 tolerates).
func TestCompletedStatusEventProvider_TakeHomeBodyRoundTrips(t *testing.T) {
	const characterId = uint32(1001)
	s := buildTakeHomeSaga(t, characterId, 1402001)

	msgs, err := CompletedStatusEventProvider(s)()
	require.NoError(t, err)
	require.Len(t, msgs, 1)

	var ev saga.StatusEvent[saga.StatusEventCompletedBody]
	require.NoError(t, json.Unmarshal(msgs[0].Value, &ev))
	require.Equal(t, saga.StatusEventTypeCompleted, ev.Type)
	require.Equal(t, string(MtsOperation), ev.Body.SagaType)
	require.Equal(t, MtsTakeHomeResultKind, ev.Body.Results["kind"])
	// After JSON round-trip numeric values are float64.
	cid, ok := ev.Body.Results["characterId"].(float64)
	require.True(t, ok, "characterId should round-trip as float64")
	require.Equal(t, characterId, uint32(cid))
}

// buildCraftSaga builds a completed-shape InventoryTransaction saga whose
// first step is RecordCraftManifest (task-285 Task 26a) -- the shape all
// four craft modes build, RecordCraftManifest first.
func buildCraftSaga(t *testing.T, characterId uint32, mode uint32) Saga {
	t.Helper()
	s, err := NewBuilder().
		SetSagaType(InventoryTransaction).
		SetInitiatedBy("MAKER_SKILL").
		AddStep("record_manifest", Completed, RecordCraftManifest, CraftManifestPayload{
			CharacterId:  characterId,
			Mode:         mode,
			TargetItemId: 1082002,
			ItemNum:      1,
			Materials:    []CraftManifestItem{{ItemId: 4011001, Count: 5}},
			MesoCost:     1200,
		}).
		AddStep("deduct_meso", Completed, AwardMesos, AwardMesosPayload{
			CharacterId: characterId, ActorType: "SYSTEM", Amount: -1200, ShowEffect: true,
		}).
		AddStep("destroy_0", Completed, DestroyAssetFromSlot, DestroyAssetFromSlotPayload{
			CharacterId: characterId, InventoryType: 4, Slot: 1, Quantity: 5, TemplateId: 4011001,
		}).
		AddStep("award_item", Completed, AwardCraftedAsset, AwardCraftedAssetPayload{
			CharacterId: characterId, TemplateId: 1082002, Quantity: 1, Slots: 7,
		}).
		Build()
	require.NoError(t, err)
	return s
}

// TestCompletedCraftSagaEchoesManifest proves the COMPLETED event for a
// craft saga carries kind=maker_craft, the top-level characterId scalar
// (for session resolution before decoding anything nested), and the full
// CraftManifestPayload as craftManifest -- decoded here the same way Task
// 26 must (re-marshal the nested map, then unmarshal into the typed
// struct), proving the contract Task 26 depends on round-trips.
func TestCompletedCraftSagaEchoesManifest(t *testing.T) {
	const characterId = uint32(1001)
	s := buildCraftSaga(t, characterId, 1)

	msgs, err := CompletedStatusEventProvider(s)()
	require.NoError(t, err)
	require.Len(t, msgs, 1)

	var ev saga.StatusEvent[saga.StatusEventCompletedBody]
	require.NoError(t, json.Unmarshal(msgs[0].Value, &ev))
	require.Equal(t, saga.StatusEventTypeCompleted, ev.Type)
	require.Equal(t, string(InventoryTransaction), ev.Body.SagaType)
	require.Equal(t, MakerCraftResultKind, ev.Body.Results["kind"])

	cid, ok := ev.Body.Results["characterId"].(float64)
	require.True(t, ok, "characterId should round-trip as float64")
	require.Equal(t, characterId, uint32(cid))

	raw, ok := ev.Body.Results["craftManifest"]
	require.True(t, ok, "craftManifest key must be present")
	bs, err := json.Marshal(raw)
	require.NoError(t, err)
	var manifest CraftManifestPayload
	require.NoError(t, json.Unmarshal(bs, &manifest))
	require.Equal(t, characterId, manifest.CharacterId)
	require.EqualValues(t, 1, manifest.Mode)
	require.EqualValues(t, 1082002, manifest.TargetItemId)
	require.Len(t, manifest.Materials, 1)
	require.EqualValues(t, 4011001, manifest.Materials[0].ItemId)
	require.EqualValues(t, 5, manifest.Materials[0].Count)
	require.EqualValues(t, 1200, manifest.MesoCost)
}

// TestCompletedNonCraftSagaEmitsNoManifest proves an InventoryTransaction
// saga with no RecordCraftManifest step yields Results without a
// craftManifest key, and the existing MTS/note/character-creation arms stay
// untouched (they simply do not match this saga either).
func TestCompletedNonCraftSagaEmitsNoManifest(t *testing.T) {
	s, err := NewBuilder().
		SetSagaType(InventoryTransaction).
		SetInitiatedBy("some-other-inventory-flow").
		AddStep("destroy_0", Completed, DestroyAssetFromSlot, DestroyAssetFromSlotPayload{
			CharacterId: 1001, InventoryType: 4, Slot: 1, Quantity: 1, TemplateId: 2000000,
		}).
		Build()
	require.NoError(t, err)

	require.Nil(t, extractMakerCraftResults(s), "a non-craft InventoryTransaction saga must not be classified as a craft")

	msgs, err := CompletedStatusEventProvider(s)()
	require.NoError(t, err)
	require.Len(t, msgs, 1)
	var ev saga.StatusEvent[saga.StatusEventCompletedBody]
	require.NoError(t, json.Unmarshal(msgs[0].Value, &ev))
	require.Empty(t, ev.Body.Results, "no extractor matches this saga, so Results stays empty")
}

// TestCompensatedCraftSagaEmitsNoManifest proves a craft saga that fails and
// compensates never reaches CompletedStatusEventProvider: the FAILED path
// (FailedStatusEventProvider / StatusEventFailedBody) carries no Results
// field at all, so no consumption is ever reported for work that was
// reversed.
func TestCompensatedCraftSagaEmitsNoManifest(t *testing.T) {
	msgs, err := FailedStatusEventProvider(uuid.New(), 0, 1001, string(InventoryTransaction), "SOME_ERROR", "compensated", "award_item")()
	require.NoError(t, err)
	require.Len(t, msgs, 1)

	var ev saga.StatusEvent[saga.StatusEventFailedBody]
	require.NoError(t, json.Unmarshal(msgs[0].Value, &ev))
	require.Equal(t, saga.StatusEventTypeFailed, ev.Type)
	// StatusEventFailedBody has no Results/craftManifest field at all --
	// verified here by decoding into the concrete map and confirming no such
	// key ever reaches the wire, structurally, not just by omission.
	var raw map[string]any
	require.NoError(t, json.Unmarshal(msgs[0].Value, &raw))
	body, ok := raw["body"].(map[string]any)
	require.True(t, ok)
	_, hasManifest := body["craftManifest"]
	require.False(t, hasManifest)
}

// TestManifestSurvivesSagaRehydration proves the manifest is typed (not a
// map[string]any) after the jsonb round-trip a pod restart performs
// (saga/entity.go): marshal the saga to JSON and unmarshal it back through
// Saga's own UnmarshalJSON (which routes each step through
// Step[any].UnmarshalJSON) before calling the extractor.
func TestManifestSurvivesSagaRehydration(t *testing.T) {
	const characterId = uint32(1001)
	s := buildCraftSaga(t, characterId, 3)

	data, err := json.Marshal(s)
	require.NoError(t, err)

	var rehydrated Saga
	require.NoError(t, json.Unmarshal(data, &rehydrated))

	results := extractMakerCraftResults(rehydrated)
	require.NotNil(t, results)
	manifest, ok := results["craftManifest"].(CraftManifestPayload)
	require.True(t, ok, "manifest must be a typed CraftManifestPayload after rehydration, not a map[string]any")
	require.Equal(t, characterId, manifest.CharacterId)
	require.EqualValues(t, 3, manifest.Mode)
}

// TestFailedCraftSagaCarriesCharacterId proves a compensated craft saga
// emits FAILED with the originating characterId, not 0 (task-285 Task 26a
// addendum). EmitSagaFailed is called identically by both entry points --
// the compensator (compensator.go) and the timeout backstop (timer.go) --
// so exercising EmitSagaFailed directly covers both.
func TestFailedCraftSagaCarriesCharacterId(t *testing.T) {
	logger, _ := test.NewNullLogger()
	const characterId = uint32(1001)
	s := buildCraftSaga(t, characterId, 1)

	type emitted struct {
		SagaType    string
		CharacterId uint32
	}
	var got []emitted
	prev := emitSagaFailedByIdsFn
	emitSagaFailedByIdsFn = func(_ logrus.FieldLogger, _ context.Context, _ uuid.UUID, sagaType string, _ uint32, characterId uint32, _ string, _ string, _ string) error {
		got = append(got, emitted{sagaType, characterId})
		return nil
	}
	t.Cleanup(func() { emitSagaFailedByIdsFn = prev })

	require.NoError(t, EmitSagaFailed(logger, context.Background(), s, "SOME_ERROR", "compensated", "award_item"))

	require.Len(t, got, 1)
	assert.Equal(t, string(InventoryTransaction), got[0].SagaType)
	assert.Equal(t, characterId, got[0].CharacterId, "characterId must be the manifest's, never 0")
}

// TestFailedNonCraftInventoryTransactionUnchanged is the regression guard:
// InventoryTransaction is not maker-exclusive, so a saga of that type with
// no RecordCraftManifest step must still route through
// ExtractCharacterCreationIds exactly as before -- characterId 0 here since
// there is no CreateCharacter step either.
func TestFailedNonCraftInventoryTransactionUnchanged(t *testing.T) {
	logger, _ := test.NewNullLogger()
	s, err := NewBuilder().
		SetSagaType(InventoryTransaction).
		SetInitiatedBy("some-other-inventory-flow").
		AddStep("destroy_0", Completed, DestroyAssetFromSlot, DestroyAssetFromSlotPayload{
			CharacterId: 1001, InventoryType: 4, Slot: 1, Quantity: 1, TemplateId: 2000000,
		}).
		Build()
	require.NoError(t, err)

	type emitted struct {
		SagaType    string
		CharacterId uint32
	}
	var got []emitted
	prev := emitSagaFailedByIdsFn
	emitSagaFailedByIdsFn = func(_ logrus.FieldLogger, _ context.Context, _ uuid.UUID, sagaType string, _ uint32, characterId uint32, _ string, _ string, _ string) error {
		got = append(got, emitted{sagaType, characterId})
		return nil
	}
	t.Cleanup(func() { emitSagaFailedByIdsFn = prev })

	require.NoError(t, EmitSagaFailed(logger, context.Background(), s, "SOME_ERROR", "failed", "destroy_0"))

	require.Len(t, got, 1)
	assert.Equal(t, string(InventoryTransaction), got[0].SagaType)
	assert.EqualValues(t, 0, got[0].CharacterId, "a non-craft InventoryTransaction saga with no CreateCharacter step must keep characterId 0")
}

// TestIncubatorResultEventProvider_EggIdSurvives proves the sacrificed Pigmy
// Egg id set on IncubatorResultPayload survives into the emitted
// EVENT_TOPIC_INCUBATOR_RESULT message body untouched. A struct-tag typo or
// field swap on ResultEvent (or a dropped assignment in the provider) must
// make this test fail — it decodes the actual produced kafka message bytes,
// not just checks err == nil.
func TestIncubatorResultEventProvider_EggIdSurvives(t *testing.T) {
	const characterId = uint32(12345)
	const itemId = uint32(5000000)
	const count = uint32(1)
	const eggId = uint32(4170003)

	payload := IncubatorResultPayload{
		CharacterId: characterId,
		WorldId:     world.Id(0),
		ChannelId:   channel.Id(1),
		ItemId:      itemId,
		Count:       count,
		EggId:       eggId,
	}

	msgs, err := IncubatorResultEventProvider(payload)()
	require.NoError(t, err)
	require.Len(t, msgs, 1)

	var ev incubator2.ResultEvent
	require.NoError(t, json.Unmarshal(msgs[0].Value, &ev))
	require.Equal(t, characterId, ev.CharacterId)
	require.Equal(t, byte(0), ev.WorldId)
	require.Equal(t, byte(1), ev.ChannelId)
	require.Equal(t, itemId, ev.ItemId)
	require.Equal(t, count, ev.Count)
	require.Equal(t, eggId, ev.EggId, "EggId must survive from payload into the emitted event body")
}

// TestMegaphoneBroadcastEventProvider_MessageShape proves
// MegaphoneBroadcastEventProvider builds exactly one message, keyed by
// WorldId (D1: single-partition ordering per world), whose JSON body
// round-trips into megaphone.BroadcastEvent with every field carried
// through from EmitMegaphonePayload. This is the happy-path coverage for
// handleEmitMegaphone's message-building logic; the handler itself is not
// exercised here because it calls the real atlas-kafka producer.ProviderImpl
// (see TestHandleEmitMegaphone_InvalidPayload for why).
func TestMegaphoneBroadcastEventProvider_MessageShape(t *testing.T) {
	payload := EmitMegaphonePayload{
		Tier:        "SUPER",
		Scope:       "WORLD",
		WorldId:     3,
		ChannelId:   1,
		CharacterId: 555,
		SenderName:  "Bob",
		SenderMedal: "<Adventurer>",
		Messages:    []string{"hello", "world"},
		WhispersOn:  true,
		Item: &AssetSnapshot{
			Slot:       -1,
			TemplateId: 5062000,
			Quantity:   1,
		},
	}

	msgs, err := MegaphoneBroadcastEventProvider(payload)()
	require.NoError(t, err)
	require.Len(t, msgs, 1)
	require.Equal(t, producer.CreateKey(int(payload.WorldId)), msgs[0].Key)

	var ev megaphone.BroadcastEvent
	require.NoError(t, json.Unmarshal(msgs[0].Value, &ev))
	require.Equal(t, payload.Tier, ev.Tier)
	require.Equal(t, payload.Scope, ev.Scope)
	require.Equal(t, byte(payload.WorldId), ev.WorldId)
	require.Equal(t, byte(payload.ChannelId), ev.ChannelId)
	require.Equal(t, payload.CharacterId, ev.CharacterId)
	require.Equal(t, payload.SenderName, ev.SenderName)
	require.Equal(t, payload.SenderMedal, ev.SenderMedal)
	require.Equal(t, payload.Messages, ev.Messages)
	require.Equal(t, payload.WhispersOn, ev.WhispersOn)
	require.NotNil(t, ev.Item)
	require.Equal(t, payload.Item.TemplateId, ev.Item.TemplateId)
}

// TestMegaphoneBroadcastEventProvider_NilItem proves the ITEM tier's Item
// field round-trips as absent (json:",omitempty") for the non-ITEM tiers,
// rather than a spurious non-nil zero value.
func TestMegaphoneBroadcastEventProvider_NilItem(t *testing.T) {
	payload := EmitMegaphonePayload{
		Tier:        "MEGAPHONE",
		Scope:       "CHANNEL",
		WorldId:     0,
		ChannelId:   0,
		CharacterId: 1,
		SenderName:  "Alice",
		Messages:    []string{"hi"},
	}

	msgs, err := MegaphoneBroadcastEventProvider(payload)()
	require.NoError(t, err)
	require.Len(t, msgs, 1)

	var ev megaphone.BroadcastEvent
	require.NoError(t, json.Unmarshal(msgs[0].Value, &ev))
	require.Nil(t, ev.Item)
}

// TestWorldBroadcastEnqueueCommandProvider_MessageShape proves
// WorldBroadcastEnqueueCommandProvider builds exactly one message, keyed by
// WorldId (D1: single-partition ordering per world so atlas-world's queue
// consumer sees enqueue commands for a world in order), whose JSON body
// round-trips into broadcast.EnqueueCommand with every field carried
// through from EnqueueWorldBroadcastPayload — including TvMessageType as
// the semantic string key, never a client wire byte (A1 delta, DOM-25(c)).
func TestWorldBroadcastEnqueueCommandProvider_MessageShape(t *testing.T) {
	payload := EnqueueWorldBroadcastPayload{
		Family:          "TV",
		WorldId:         2,
		ChannelId:       4,
		CharacterId:     777,
		SenderName:      "Carol",
		SenderMedal:     "<GM>",
		Messages:        []string{"a", "b", "c", "d", "e"},
		WhispersOn:      false,
		ItemId:          0,
		TvMessageType:   "STAR",
		DurationSeconds: 30,
		SenderLook: AvatarSnapshot{
			Gender:    0,
			SkinColor: 0,
			Face:      20000,
			Hair:      30000,
		},
		ReceiverName: "Dave",
		ReceiverLook: &AvatarSnapshot{
			Gender: 1,
			Face:   20001,
			Hair:   30001,
		},
	}

	msgs, err := WorldBroadcastEnqueueCommandProvider(payload)()
	require.NoError(t, err)
	require.Len(t, msgs, 1)
	require.Equal(t, producer.CreateKey(int(payload.WorldId)), msgs[0].Key)

	var cmd broadcast.EnqueueCommand
	require.NoError(t, json.Unmarshal(msgs[0].Value, &cmd))
	require.Equal(t, payload.Family, cmd.Family)
	require.Equal(t, byte(payload.WorldId), cmd.WorldId)
	require.Equal(t, byte(payload.ChannelId), cmd.ChannelId)
	require.Equal(t, payload.CharacterId, cmd.CharacterId)
	require.Equal(t, payload.SenderName, cmd.SenderName)
	require.Equal(t, payload.SenderMedal, cmd.SenderMedal)
	require.Equal(t, payload.Messages, cmd.Messages)
	require.Equal(t, payload.WhispersOn, cmd.WhispersOn)
	require.Equal(t, payload.ItemId, cmd.ItemId)
	require.Equal(t, "STAR", cmd.TvMessageType)
	require.Equal(t, payload.DurationSeconds, cmd.DurationSeconds)
	require.Equal(t, payload.SenderLook.Face, cmd.SenderLook.Face)
	require.Equal(t, payload.ReceiverName, cmd.ReceiverName)
	require.NotNil(t, cmd.ReceiverLook)
	require.Equal(t, payload.ReceiverLook.Face, cmd.ReceiverLook.Face)
}

// TestNpcShopEnterCommandProvider asserts the ENTER command carries the saga's
// transaction id — the orchestrator's only correlation key when the ENTERED
// event comes back (task-221 design delta D2).
func TestNpcShopEnterCommandProvider(t *testing.T) {
	txn := uuid.New()
	msgs, err := NpcShopEnterCommandProvider(txn, OpenNpcShopPayload{
		CharacterId:   1234,
		WorldId:       world.Id(0),
		ChannelId:     channel.Id(1),
		NpcTemplateId: 9090000,
	})()
	if err != nil {
		t.Fatalf("provider: %v", err)
	}
	if len(msgs) != 1 {
		t.Fatalf("len(msgs) = %d, want 1", len(msgs))
	}
	var cmd npcshop.Command[npcshop.CommandShopEnterBody]
	if err := json.Unmarshal(msgs[0].Value, &cmd); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if cmd.TransactionId != txn {
		t.Errorf("TransactionId = %s, want %s", cmd.TransactionId, txn)
	}
	if cmd.Type != npcshop.CommandShopEnter {
		t.Errorf("Type = %q, want %q", cmd.Type, npcshop.CommandShopEnter)
	}
	if cmd.CharacterId != 1234 || cmd.Body.NpcTemplateId != 9090000 {
		t.Errorf("unexpected command: %+v", cmd)
	}
}

// TestNpcConversationStartItemCommandProvider asserts the START_ITEM_CONVERSATION
// command carries the saga's transaction id, the item id, slot, and account id
// (task-230) — the orchestrator's only correlation key when STARTED/START_ERROR
// comes back.
func TestNpcConversationStartItemCommandProvider(t *testing.T) {
	txn := uuid.New()
	msgs, err := NpcConversationStartItemCommandProvider(txn, StartItemConversationPayload{
		CharacterId:   1234,
		AccountId:     77,
		ItemId:        2430008,
		NpcTemplateId: 2084002,
		Slot:          5,
		ChannelId:     1,
		MapId:         100000000,
	})()
	if err != nil {
		t.Fatalf("provider: %v", err)
	}
	if len(msgs) != 1 {
		t.Fatalf("messages: got %d, want 1", len(msgs))
	}

	var c npc.Command[npc.CommandItemConversationStartBody]
	if err := json.Unmarshal(msgs[0].Value, &c); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if c.Type != npc.CommandTypeStartItemConversation {
		t.Errorf("type: got %q", c.Type)
	}
	if c.TransactionId != txn {
		t.Errorf("transactionId: got %s want %s", c.TransactionId, txn)
	}
	if c.NpcId != 2084002 {
		t.Errorf("npcId (the avatar): got %d want 2084002", c.NpcId)
	}
	if c.Body.ItemId != 2430008 || c.Body.Slot != 5 || c.Body.AccountId != 77 {
		t.Errorf("body: %+v", c.Body)
	}
}

// TestNpcConversationStartNpcCommandProvider asserts the START_CONVERSATION
// command reuses the ordinary conversation command type while still carrying
// the saga's transaction id (task-230) — the transaction id is what makes it
// saga-driven, distinct from the ordinary NPC-talk path's uuid.Nil.
func TestNpcConversationStartNpcCommandProvider(t *testing.T) {
	txn := uuid.New()
	msgs, err := NpcConversationStartNpcCommandProvider(txn, StartNpcConversationPayload{
		CharacterId:   4243,
		AccountId:     77,
		NpcTemplateId: 9090002,
		ChannelId:     1,
		MapId:         100000000,
	})()
	if err != nil {
		t.Fatalf("provider: %v", err)
	}
	if len(msgs) != 1 {
		t.Fatalf("messages: got %d, want 1", len(msgs))
	}

	var c npc.Command[npc.CommandConversationStartBody]
	if err := json.Unmarshal(msgs[0].Value, &c); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if c.Type != npc.CommandTypeStartConversation {
		t.Errorf("type: got %q", c.Type)
	}
	if c.TransactionId != txn {
		t.Errorf("transactionId: got %s want %s", c.TransactionId, txn)
	}
	if c.NpcId != 9090002 {
		t.Errorf("npcId: got %d want 9090002", c.NpcId)
	}
	if c.Body.AccountId != 77 {
		t.Errorf("body: %+v", c.Body)
	}
}

// TestNpcConversationEndCommandProvider asserts the END_CONVERSATION
// compensation command carries the saga's transaction id, character id, and
// npc template id (task-230).
func TestNpcConversationEndCommandProvider(t *testing.T) {
	txn := uuid.New()
	msgs, err := NpcConversationEndCommandProvider(txn, 1234, 2084002)()
	if err != nil {
		t.Fatalf("provider: %v", err)
	}
	if len(msgs) != 1 {
		t.Fatalf("messages: got %d, want 1", len(msgs))
	}

	var c npc.Command[npc.CommandConversationEndBody]
	if err := json.Unmarshal(msgs[0].Value, &c); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if c.Type != npc.CommandTypeEndConversation {
		t.Errorf("type: got %q", c.Type)
	}
	if c.TransactionId != txn {
		t.Errorf("transactionId: got %s want %s", c.TransactionId, txn)
	}
	if c.CharacterId != 1234 || c.NpcId != 2084002 {
		t.Errorf("unexpected command: %+v", c)
	}
}
