package saga

import (
	"testing"

	sagaMsg "atlas-saga-orchestrator/kafka/message/saga"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

// TestExtractMtsFailureTarget_Buy proves a settle/buy saga (whose move step carries
// the buyer) reports the buyer as the character to notify and the buy kind, so the
// channel writes BuyItemFailed to unhang the buy dialog on a saga failure.
func TestExtractMtsFailureTarget_Buy(t *testing.T) {
	s, err := NewBuilder().
		SetSagaType(MtsOperation).
		SetInitiatedBy("test").
		AddStep("award_currency_buyer", Completed, AwardCurrency, AwardCurrencyPayload{CharacterId: 100, AccountId: 10, CurrencyType: 3, Amount: -1100}).
		AddStep("award_currency_seller", Pending, AwardCurrency, AwardCurrencyPayload{CharacterId: 200, AccountId: 20, CurrencyType: 2, Amount: 1000}).
		AddStep("mts_move_listing_to_holding", Pending, MtsMoveListingToHolding, MtsMoveListingToHoldingPayload{ListingId: uuid.New(), BuyerId: 100}).
		Build()
	require.NoError(t, err)

	characterId, kind := extractMtsFailureTarget(s)
	require.Equal(t, uint32(100), characterId)
	require.Equal(t, sagaMsg.MtsFailureKindBuy, kind)
}

// TestExtractMtsFailureTarget_List proves a list (TransferToMts) saga reports the
// seller and the list kind, so the channel writes RegisterSaleEntryFailed.
func TestExtractMtsFailureTarget_List(t *testing.T) {
	s, err := NewBuilder().
		SetSagaType(MtsOperation).
		SetInitiatedBy("test").
		AddStep("release_from_character", Completed, ReleaseFromCharacter, ReleaseFromCharacterPayload{CharacterId: 300}).
		AddStep("accept_to_mts_listing", Pending, AcceptToMtsListing, AcceptToMtsListingPayload{SellerId: 300}).
		Build()
	require.NoError(t, err)

	characterId, kind := extractMtsFailureTarget(s)
	require.Equal(t, uint32(300), characterId)
	require.Equal(t, sagaMsg.MtsFailureKindList, kind)
}

// TestExtractMtsFailureTarget_TakeHome proves a take-home (WithdrawFromMts) saga
// (unique in carrying a ReleaseFromMtsHolding step) reports the AcceptToCharacter
// recipient and the take-home kind, so the channel writes
// MoveItcPurchaseItemLtoSFailed.
func TestExtractMtsFailureTarget_TakeHome(t *testing.T) {
	s, err := NewBuilder().
		SetSagaType(MtsOperation).
		SetInitiatedBy("test").
		AddStep("release_from_mts_holding", Completed, ReleaseFromMtsHolding, ReleaseFromMtsHoldingPayload{HoldingId: uuid.New()}).
		AddStep("accept_to_character", Pending, AcceptToCharacter, AcceptToCharacterPayload{CharacterId: 400, TemplateId: 1402001}).
		Build()
	require.NoError(t, err)

	characterId, kind := extractMtsFailureTarget(s)
	require.Equal(t, uint32(400), characterId)
	require.Equal(t, sagaMsg.MtsFailureKindTakeHome, kind)
}

// TestExtractMtsFailureTarget_Unknown proves a saga matching none of the MTS
// discriminators reports (0, "") so the channel skips notification rather than
// guessing an arm.
func TestExtractMtsFailureTarget_Unknown(t *testing.T) {
	s, err := NewBuilder().
		SetSagaType(MtsOperation).
		SetInitiatedBy("test").
		AddStep("award_currency_only", Pending, AwardCurrency, AwardCurrencyPayload{CharacterId: 1, AccountId: 1, CurrencyType: 2, Amount: 5}).
		Build()
	require.NoError(t, err)

	characterId, kind := extractMtsFailureTarget(s)
	require.Equal(t, uint32(0), characterId)
	require.Equal(t, "", kind)
}

// TestWalletErrorOutcomeIsFailure pins the fast-fail wiring: a wallet ERROR ack is
// classified as a failure and is accepted on the currency-moving steps, so a failed
// transactional adjust fails the saga step instead of waiting out the timeout.
func TestWalletErrorOutcomeIsFailure(t *testing.T) {
	require.Equal(t, OutcomeFailure, outcomeTable[EventKindCashShopWalletError])

	accepts := func(action Action) bool {
		for _, k := range acceptanceTable[action] {
			if k == EventKindCashShopWalletError {
				return true
			}
		}
		return false
	}
	require.True(t, accepts(AwardCurrency), "AwardCurrency must accept the wallet error ack")
	require.True(t, accepts(MtsBidEscrow), "MtsBidEscrow must accept the wallet error ack")
}
