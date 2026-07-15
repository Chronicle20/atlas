package saga

import (
	asset2 "atlas-saga-orchestrator/kafka/message/asset"
	"atlas-saga-orchestrator/kafka/message/saga"
	"encoding/json"
	"testing"

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
