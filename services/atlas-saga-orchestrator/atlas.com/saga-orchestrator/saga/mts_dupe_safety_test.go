package saga

// mts_dupe_safety_test.go — the orchestrator arm of the MTS dupe-safety suite
// (task-102 §5.2, NFR 8.1: "no trade can duplicate an item or desync currency
// under crash, replay, or race"). Each test asserts the SINGLE-CUSTODY invariant
// (exactly one copy of the item; currency net-zero across forward+compensation).
//
// Scenarios covered here:
//   1. crash-mid-list      — TestDupeSafety_CrashMidList_RegrantsToExactlyOnePlace
//   2. grant-before-debit  — TestDupeSafety_GrantBeforeDebit_DebitFirstNoEarlyGrant
//                          — TestDupeSafety_GrantBeforeDebit_DebitFailGrantsNothing
//
// (Scenarios 3 double-grant replay, 4 cancel-racing-purchase, and 5 take-home
// replay are atlas-mts custody/listing tests; see the dupe_safety_test.go files
// under services/atlas-mts/.../custody and .../listing.)
//
// The compensation tests exercise DispatchMtsOperationRollbacks directly to avoid
// the EmitSagaFailed Kafka path (no broker in the test environment), mirroring
// preset_integration_test.go's TestPresetCompensation and mts_integration_test.go.

import (
	compartmentmock "atlas-saga-orchestrator/compartment/mock"
	asset2 "atlas-saga-orchestrator/kafka/message/asset"
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	logtest "github.com/sirupsen/logrus/hooks/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

// dupeRegrantCall captures a RequestAcceptAsset (re-grant) dispatch.
type dupeRegrantCall struct {
	CharacterId   uint32
	InventoryType byte
	TemplateId    uint32
	AssetData     asset2.AssetData
}

// ---------------------------------------------------------------------------
// Scenario 1: crash-mid-list
//
// TransferToMts expands to [release_from_character, accept_to_mts_listing]. If the
// saga fails at/after accept_to_mts_listing, the reverse-walk compensation must
// re-grant the released item to the character (AcceptToCharacter via
// RequestAcceptAsset) so the item ends in EXACTLY ONE place (character inventory)
// and NO listing row exists.
//
// Why no listing-undo: AcceptToMtsListing is its own atomic local tx in atlas-mts;
// a failure there commits nothing, so there is no listing row to destroy — the
// ReleaseFromCharacter inverse alone restores single custody.
// ---------------------------------------------------------------------------

func TestDupeSafety_CrashMidList_RegrantsToExactlyOnePlace(t *testing.T) {
	logger, _ := logtest.NewNullLogger()
	logger.SetLevel(logrus.DebugLevel)

	ctx := context.Background()
	te, _ := tenant.Create(uuid.New(), "GMS", 83, 1)
	tctx := tenant.WithContext(ctx, te)

	const (
		characterId   = uint32(9001)
		inventoryType = byte(1)
		templateId    = uint32(1302000)
	)
	transactionId := uuid.New()
	listingId := uuid.New()

	var acceptCalls []dupeRegrantCall
	var destroyCalls int
	compMock := &compartmentmock.ProcessorMock{
		RequestAcceptAssetFunc: func(_ uuid.UUID, characterId uint32, inventoryType byte, templateId uint32, assetData asset2.AssetData) error {
			acceptCalls = append(acceptCalls, dupeRegrantCall{
				CharacterId:   characterId,
				InventoryType: inventoryType,
				TemplateId:    templateId,
				AssetData:     assetData,
			})
			return nil
		},
		RequestDestroyItemFunc: func(_ uuid.UUID, _ uint32, _ uint32, _ uint32, _ bool) error {
			destroyCalls++
			return nil
		},
	}

	// Saga shape mirrors expandTransferToMts:
	//   step 0: release_from_character    ← Completed (item left inventory)
	//   step 1: accept_to_mts_listing     ← Failed (the listing create tx rolled back)
	s, err := NewBuilder().
		SetTransactionId(transactionId).
		SetSagaType(MtsOperation).
		SetInitiatedBy("mts-crash-mid-list-test").
		AddStep("release_from_character", Completed, ReleaseFromCharacter, ReleaseFromCharacterPayload{
			TransactionId: transactionId,
			CharacterId:   characterId,
			InventoryType: inventoryType,
			AssetId:       42,
			Quantity:      1,
		}).
		AddStep("accept_to_mts_listing", Failed, AcceptToMtsListing, AcceptToMtsListingPayload{
			TransactionId: transactionId,
			ListingId:     listingId,
			WorldId:       0,
			SellerId:      characterId,
			SellerName:    "Seller",
			SaleType:      "buy_now",
			TemplateId:    templateId,
			Quantity:      1,
			WeaponAttack:  17,
			Slots:         7,
			Level:         1,
		}).
		Build()
	require.NoError(t, err, "saga build should not fail")
	require.NoError(t, GetCache().Put(tctx, s))

	require.True(t, GetCache().TryTransition(tctx, transactionId, SagaLifecyclePending, SagaLifecycleCompensating),
		"lifecycle should transition Pending → Compensating")

	compensator := NewCompensator(logger, tctx).WithCompartmentProcessor(compMock)
	compensator.DispatchMtsOperationRollbacks(s)

	require.True(t, GetCache().TryTransition(tctx, transactionId, SagaLifecycleCompensating, SagaLifecycleFailed),
		"lifecycle should transition Compensating → Failed")
	GetCache().Remove(tctx, transactionId)

	// INVARIANT: the item is re-granted to EXACTLY ONE place — the character's
	// inventory — once, carrying the listing snapshot stats.
	require.Equal(t, 1, len(acceptCalls),
		"crash-mid-list must re-grant the item to exactly one place (1 AcceptToCharacter), got %d", len(acceptCalls))
	regrant := acceptCalls[0]
	assert.Equal(t, characterId, regrant.CharacterId, "re-grant must target the seller character")
	assert.Equal(t, inventoryType, regrant.InventoryType, "re-grant must use the source inventory type")
	assert.Equal(t, templateId, regrant.TemplateId, "re-grant must restore the listed item")
	assert.Equal(t, uint16(17), regrant.AssetData.WeaponAttack, "re-grant must carry the snapshot stats")
	assert.Equal(t, uint16(7), regrant.AssetData.Slots, "re-grant must carry the snapshot slots")

	// INVARIANT: no destroy — the failed AcceptToMtsListing committed no listing
	// row, so there is nothing to un-list; a destroy would lose the item entirely.
	assert.Equal(t, 0, destroyCalls,
		"crash-mid-list must NOT destroy the item (no listing row was committed)")
}

// ---------------------------------------------------------------------------
// Scenario 2: grant-before-debit (debit-first)
//
// MtsSettlePurchase must expand debit-first: the buyer's prepaid debit is step 1,
// BEFORE the seller credit (step 2) and the custody move (step 3). So a buyer-debit
// failure grants nothing — no item moves and no currency is credited before the
// buyer has paid.
// ---------------------------------------------------------------------------

// TestDupeSafety_GrantBeforeDebit_DebitFirstNoEarlyGrant asserts NO grant (seller
// credit) and NO custody move precede the buyer debit in the expanded settlement.
// This is the dupe-safety ordering invariant: the buyer debit is strictly first.
func TestDupeSafety_GrantBeforeDebit_DebitFirstNoEarlyGrant(t *testing.T) {
	logger, _ := logtest.NewNullLogger()
	p, ok := NewProcessor(logger, mtsDupeTenantCtx(t)).(*ProcessorImpl)
	require.True(t, ok, "NewProcessor must return *ProcessorImpl")

	listingId := uuid.New()
	txId := uuid.New()
	payload := MtsSettlePurchasePayload{
		TransactionId:   txId,
		ListingId:       listingId,
		WorldId:         0,
		BuyerId:         100,
		BuyerAccountId:  10,
		SellerId:        200,
		SellerAccountId: 20,
		MarkedUpPrice:   1100,
		ListValue:       1000,
	}
	st := NewStep[any]("mts_settle_purchase-dupe", Pending, MtsSettlePurchase, payload)

	steps, err := p.expandMtsSettlePurchase(st)
	require.NoError(t, err)
	require.Len(t, steps, 3)

	// INVARIANT: the buyer debit is step 0 (strictly first). Scan every step BEFORE
	// the debit and assert none is a grant (seller credit) or a custody move.
	buyerDebitIdx := -1
	for i, step := range steps {
		if step.Action() == AwardCurrency {
			if pay, ok := step.Payload().(AwardCurrencyPayload); ok && pay.CharacterId == payload.BuyerId {
				buyerDebitIdx = i
				assert.Equal(t, int32(-1100), pay.Amount, "buyer debit must be the negative markedUpPrice")
				break
			}
		}
		// Any step encountered before the buyer debit must NOT be a grant/move.
		assert.NotEqual(t, MtsMoveListingToHolding, step.Action(),
			"custody move must NOT precede the buyer debit (no grant before debit)")
		if pay, ok := step.Payload().(AwardCurrencyPayload); ok {
			assert.NotEqual(t, payload.SellerId, pay.CharacterId,
				"seller credit must NOT precede the buyer debit (no grant before debit)")
		}
	}
	require.Equal(t, 0, buyerDebitIdx, "buyer prepaid debit must be the FIRST settlement step")

	// And the move (the actual item grant) is the LAST step.
	assert.Equal(t, MtsMoveListingToHolding, steps[2].Action(),
		"custody move (item grant) must be the last settlement step, after both currency moves")
}

// TestDupeSafety_GrantBeforeDebit_DebitFailGrantsNothing asserts the compensation
// invariant: if the buyer debit fails (step 0 Failed, nothing after it ran), the
// reverse-walk dispatches NOTHING — no currency reversal (nothing was credited),
// no custody un-move (nothing was moved). Net currency change is ZERO and no
// holding was granted. A buyer-debit failure grants nothing.
func TestDupeSafety_GrantBeforeDebit_DebitFailGrantsNothing(t *testing.T) {
	logger, _ := logtest.NewNullLogger()
	logger.SetLevel(logrus.DebugLevel)

	ctx := context.Background()
	te, _ := tenant.Create(uuid.New(), "GMS", 83, 1)
	tctx := tenant.WithContext(ctx, te)

	const (
		buyerId         = uint32(7001)
		buyerAccountId  = uint32(8001)
		sellerId        = uint32(7002)
		sellerAccountId = uint32(8002)
		markedUpPrice   = int32(1100)
		listValue       = int32(1000)
		currencyPoints  = uint32(2)
		currencyPrepaid = uint32(3)
	)
	transactionId := uuid.New()
	listingId := uuid.New()

	cashshopMock := &mtsTestCashshopMock{}
	mtsMockP := &mtsTestMtsMock{}

	// Settlement where the very FIRST step (buyer debit) Failed; the seller credit
	// and the move never ran (Pending).
	s, err := NewBuilder().
		SetTransactionId(transactionId).
		SetSagaType(MtsOperation).
		SetInitiatedBy("mts-debit-fail-test").
		AddStep("award_currency_buyer", Failed, AwardCurrency, AwardCurrencyPayload{
			CharacterId:  buyerId,
			AccountId:    buyerAccountId,
			CurrencyType: currencyPrepaid,
			Amount:       -markedUpPrice,
		}).
		AddStep("award_currency_seller", Pending, AwardCurrency, AwardCurrencyPayload{
			CharacterId:  sellerId,
			AccountId:    sellerAccountId,
			CurrencyType: currencyPoints,
			Amount:       listValue,
		}).
		AddStep("mts_move_listing_to_holding", Pending, MtsMoveListingToHolding, MtsMoveListingToHoldingPayload{
			TransactionId: transactionId,
			ListingId:     listingId,
			BuyerId:       buyerId,
			WorldId:       0,
		}).
		Build()
	require.NoError(t, err, "saga build should not fail")
	require.NoError(t, GetCache().Put(tctx, s))

	require.True(t, GetCache().TryTransition(tctx, transactionId, SagaLifecyclePending, SagaLifecycleCompensating))

	compensator := NewCompensator(logger, tctx).
		WithCashshopProcessor(cashshopMock).
		WithMtsProcessor(mtsMockP)
	compensator.DispatchMtsOperationRollbacks(s)

	require.True(t, GetCache().TryTransition(tctx, transactionId, SagaLifecycleCompensating, SagaLifecycleFailed))
	GetCache().Remove(tctx, transactionId)

	// INVARIANT: nothing was granted before the debit, so nothing needs reversing.
	assert.Equal(t, 0, len(cashshopMock.awardCalls),
		"a failed buyer debit must produce NO currency reversals (nothing was credited) — net currency zero")
	assert.Equal(t, 0, mtsMockP.moveCalls, "no custody move was dispatched (item never granted)")
	assert.Equal(t, 0, mtsMockP.restoreCalls, "no custody restore needed (no holding was created)")
	assert.Equal(t, 0, mtsMockP.acceptCalls, "no listing accept during a settle compensation")
	assert.Equal(t, 0, mtsMockP.releaseCalls, "no holding release during a settle compensation")
}

// mtsDupeTenantCtx builds a tenant context for the expansion-ordering test.
func mtsDupeTenantCtx(t *testing.T) context.Context {
	t.Helper()
	te, err := tenant.Create(uuid.New(), "GMS", 83, 1)
	require.NoError(t, err)
	return tenant.WithContext(context.Background(), te)
}
