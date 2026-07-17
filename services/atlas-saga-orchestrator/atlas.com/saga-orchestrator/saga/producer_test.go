package saga

import (
	asset2 "atlas-saga-orchestrator/kafka/message/asset"
	"atlas-saga-orchestrator/kafka/message/broadcast"
	"atlas-saga-orchestrator/kafka/message/megaphone"
	"atlas-saga-orchestrator/kafka/message/saga"
	"encoding/json"
	"testing"

	"github.com/Chronicle20/atlas/libs/atlas-kafka/producer"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
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
