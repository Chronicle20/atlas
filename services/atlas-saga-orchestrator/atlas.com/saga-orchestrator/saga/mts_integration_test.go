package saga

// TestMtsSettlePurchaseCompensation is the MTS dupe-safety integration test
// (task-102 §4.1 / §14). It drives an MtsSettlePurchase saga, forces a mid-saga
// failure at the FINAL custody step (mts_move_listing_to_holding), and asserts
// the reverse-walk compensation runs correctly:
//
//   - the buyer is re-credited (+markedUpPrice) — the inverse of the step-1 debit,
//   - the seller is debited (-listValue) — the inverse of the step-2 credit,
//   - the net currency change across the whole saga is ZERO (commission sink aside,
//     the two reversals exactly cancel the two forward awards),
//   - NO custody "un-move" is dispatched: the move step is one atomic local DB tx
//     in atlas-mts, so a failure there commits nothing — the listing stays `active`
//     and no buyer holding was created. The single-custody invariant holds: exactly
//     one custody copy of the item exists (the still-active listing), never zero or
//     two.
//
// Saga shape mirrors expandMtsSettlePurchase:
//
//	step 0: award_currency_buyer  (prepaid, -markedUp) ← Completed
//	step 1: award_currency_seller (points,  +listVal)  ← Completed
//	step 2: mts_move_listing_to_holding                ← Failed (forced failure here)
//
// Expected reverse-walk dispatches from DispatchMtsOperationRollbacks:
//   - AwardCurrency(seller, points,  -listValue)   (inverse of step 1) — debit
//   - AwardCurrency(buyer,  prepaid, +markedUpPrice) (inverse of step 0) — re-credit
//   - NO MtsMoveListingToHolding inverse — step 2 was Failed (committed nothing).
//
// The test exercises DispatchMtsOperationRollbacks directly to avoid the
// EmitSagaFailed Kafka path (no broker in the test environment), mirroring
// preset_integration_test.go's TestPresetCompensation.

import (
	"atlas-saga-orchestrator/cashshop"
	"atlas-saga-orchestrator/kafka/message"
	mtsmock "atlas-saga-orchestrator/mts"
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"github.com/sirupsen/logrus/hooks/test"
	"github.com/stretchr/testify/assert"

	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

// mtsTestCashshopMock captures AwardCurrencyAndEmit calls so the test can assert
// the reversal awards (re-credit buyer / debit seller). It implements only the
// methods the compensator reverse-walk touches; the rest satisfy the interface.
type mtsTestCashshopMock struct {
	awardCalls []mtsAwardCurrencyCall
}

type mtsAwardCurrencyCall struct {
	AccountId    uint32
	CurrencyType uint32
	Amount       int32
}

func (m *mtsTestCashshopMock) AwardCurrencyAndEmit(_ uuid.UUID, accountId uint32, currencyType uint32, amount int32) error {
	m.awardCalls = append(m.awardCalls, mtsAwardCurrencyCall{AccountId: accountId, CurrencyType: currencyType, Amount: amount})
	return nil
}

func (m *mtsTestCashshopMock) AwardCurrency(_ *message.Buffer) func(uuid.UUID, uint32, uint32, int32) error {
	return func(_ uuid.UUID, _ uint32, _ uint32, _ int32) error { return nil }
}

func (m *mtsTestCashshopMock) AcceptAndEmit(_ uuid.UUID, _ uint32, _ uint32, _ uuid.UUID, _ byte, _ int64, _ uint32, _ uint32, _ uint32, _ uint32, _ uint16) error {
	return nil
}

func (m *mtsTestCashshopMock) Accept(_ *message.Buffer) func(uuid.UUID, uint32, uint32, uuid.UUID, byte, int64, uint32, uint32, uint32, uint32, uint16) error {
	return func(_ uuid.UUID, _ uint32, _ uint32, _ uuid.UUID, _ byte, _ int64, _ uint32, _ uint32, _ uint32, _ uint32, _ uint16) error {
		return nil
	}
}

func (m *mtsTestCashshopMock) ReleaseAndEmit(_ uuid.UUID, _ uint32, _ uint32, _ uuid.UUID, _ byte, _ uint32, _ int64, _ uint32) error {
	return nil
}

func (m *mtsTestCashshopMock) Release(_ *message.Buffer) func(uuid.UUID, uint32, uint32, uuid.UUID, byte, uint32, int64, uint32) error {
	return func(_ uuid.UUID, _ uint32, _ uint32, _ uuid.UUID, _ byte, _ uint32, _ int64, _ uint32) error {
		return nil
	}
}

// Ensure the mock satisfies cashshop.Processor at compile time.
var _ cashshop.Processor = (*mtsTestCashshopMock)(nil)

// mtsTestMtsMock captures custody dispatches so the test can assert NO un-move /
// restore was dispatched for the (Failed) settlement move.
type mtsTestMtsMock struct {
	restoreCalls          int
	restoreHoldingId      uuid.UUID
	moveCalls             int
	acceptCalls           int
	releaseCalls          int
	removeListingCalls    int
	removeListingId       uuid.UUID
	restoreListingCalls   int
	restoreListingId      uuid.UUID
	restoreListingBuyerId uint32
}

func (m *mtsTestMtsMock) RestoreMtsHoldingAndEmit(_ uuid.UUID, holdingId uuid.UUID) error {
	m.restoreCalls++
	m.restoreHoldingId = holdingId
	return nil
}

func (m *mtsTestMtsMock) MoveListingToHoldingAndEmit(_ uuid.UUID, _ uuid.UUID, _ uint32, _ byte, _ string, _ uint32) error {
	m.moveCalls++
	return nil
}

func (m *mtsTestMtsMock) AcceptToMtsListingAndEmit(_ uuid.UUID, _ mtsmock.AcceptToMtsListingParams) error {
	m.acceptCalls++
	return nil
}

func (m *mtsTestMtsMock) ReleaseFromMtsHoldingAndEmit(_ uuid.UUID, _ uuid.UUID) error {
	m.releaseCalls++
	return nil
}

func (m *mtsTestMtsMock) AcceptToMtsListing(_ *message.Buffer) func(uuid.UUID, mtsmock.AcceptToMtsListingParams) error {
	return func(_ uuid.UUID, _ mtsmock.AcceptToMtsListingParams) error { return nil }
}

func (m *mtsTestMtsMock) ReleaseFromMtsHolding(_ *message.Buffer) func(uuid.UUID, uuid.UUID) error {
	return func(_ uuid.UUID, _ uuid.UUID) error { return nil }
}

func (m *mtsTestMtsMock) RestoreMtsHolding(_ *message.Buffer) func(uuid.UUID, uuid.UUID) error {
	return func(_ uuid.UUID, _ uuid.UUID) error { return nil }
}

func (m *mtsTestMtsMock) MoveListingToHolding(_ *message.Buffer) func(uuid.UUID, uuid.UUID, uint32, byte, string, uint32) error {
	return func(_ uuid.UUID, _ uuid.UUID, _ uint32, _ byte, _ string, _ uint32) error { return nil }
}

func (m *mtsTestMtsMock) RemoveMtsListingAndEmit(_ uuid.UUID, listingId uuid.UUID) error {
	m.removeListingCalls++
	m.removeListingId = listingId
	return nil
}

func (m *mtsTestMtsMock) RemoveMtsListing(_ *message.Buffer) func(uuid.UUID, uuid.UUID) error {
	return func(_ uuid.UUID, _ uuid.UUID) error { return nil }
}

func (m *mtsTestMtsMock) RestoreListingFromHoldingAndEmit(_ uuid.UUID, listingId uuid.UUID, buyerId uint32) error {
	m.restoreListingCalls++
	m.restoreListingId = listingId
	m.restoreListingBuyerId = buyerId
	return nil
}

func (m *mtsTestMtsMock) RestoreListingFromHolding(_ *message.Buffer) func(uuid.UUID, uuid.UUID, uint32) error {
	return func(_ uuid.UUID, _ uuid.UUID, _ uint32) error { return nil }
}

// Ensure the mock satisfies mts.Processor at compile time.
var _ mtsmock.Processor = (*mtsTestMtsMock)(nil)

func TestMtsSettlePurchaseCompensation(t *testing.T) {
	// ------------------------------------------------------------------ setup
	logger, _ := test.NewNullLogger()
	logger.SetLevel(logrus.DebugLevel)

	ctx := context.Background()
	te, _ := tenant.Create(uuid.New(), "GMS", 83, 1)
	tctx := tenant.WithContext(ctx, te)

	const (
		buyerId         = uint32(7001)
		buyerAccountId  = uint32(8001)
		sellerId        = uint32(7002)
		sellerAccountId = uint32(8002)
		markedUpPrice   = int32(1200) // buyer pays (debited -markedUp at step 0)
		listValue       = int32(1000) // seller receives (credited +listVal at step 1); commission = 200 = sink
		currencyPoints  = uint32(2)
		currencyPrepaid = uint32(3)
	)

	transactionId := uuid.New()
	listingId := uuid.New()

	cashshopMock := &mtsTestCashshopMock{}
	mtsMockP := &mtsTestMtsMock{}

	// ------------------------------------------------------------------ build saga
	//
	// Steps 0–1 Completed (currency moved), step 2 Failed (the atomic custody move
	// failed → committed nothing). SagaType MtsOperation routes CompensateFailedStep
	// to the MTS reverse-walk.
	s, err := NewBuilder().
		SetTransactionId(transactionId).
		SetSagaType(MtsOperation).
		SetInitiatedBy("mts-settle-compensation-test").
		AddStep("award_currency_buyer", Completed, AwardCurrency, AwardCurrencyPayload{
			CharacterId:  buyerId,
			AccountId:    buyerAccountId,
			CurrencyType: currencyPrepaid,
			Amount:       -markedUpPrice,
		}).
		AddStep("award_currency_seller", Completed, AwardCurrency, AwardCurrencyPayload{
			CharacterId:  sellerId,
			AccountId:    sellerAccountId,
			CurrencyType: currencyPoints,
			Amount:       listValue,
		}).
		AddStep("mts_move_listing_to_holding", Failed, MtsMoveListingToHolding, MtsMoveListingToHoldingPayload{
			TransactionId: transactionId,
			ListingId:     listingId,
			BuyerId:       buyerId,
			WorldId:       0,
		}).
		Build()
	assert.NoError(t, err, "saga build should not fail")

	assert.NoError(t, GetCache().Put(tctx, s))

	// ------------------------------------------------------------------ lifecycle: Pending → Compensating
	ok := GetCache().TryTransition(tctx, transactionId, SagaLifecyclePending, SagaLifecycleCompensating)
	assert.True(t, ok, "lifecycle should transition Pending → Compensating")

	// ------------------------------------------------------------------ dispatch rollbacks
	compensator := NewCompensator(logger, tctx).
		WithCashshopProcessor(cashshopMock).
		WithMtsProcessor(mtsMockP)

	compensator.DispatchMtsOperationRollbacks(s)

	// ------------------------------------------------------------------ lifecycle: Compensating → Failed
	finalizedOk := GetCache().TryTransition(tctx, transactionId, SagaLifecycleCompensating, SagaLifecycleFailed)
	assert.True(t, finalizedOk, "lifecycle should transition Compensating → Failed after compensation")
	GetCache().Remove(tctx, transactionId)

	// ------------------------------------------------------------------ assertions

	// 1. Exactly two currency reversals were dispatched (one per completed award).
	assert.Equal(t, 2, len(cashshopMock.awardCalls),
		"expected 2 AwardCurrency reversals (re-credit buyer + debit seller)")

	// 2. Buyer is re-credited +markedUpPrice on the prepaid wallet.
	var buyerReCredit, sellerDebit *mtsAwardCurrencyCall
	for i := range cashshopMock.awardCalls {
		switch cashshopMock.awardCalls[i].AccountId {
		case buyerAccountId:
			buyerReCredit = &cashshopMock.awardCalls[i]
		case sellerAccountId:
			sellerDebit = &cashshopMock.awardCalls[i]
		}
	}
	assert.NotNil(t, buyerReCredit, "buyer reversal must be dispatched")
	if buyerReCredit != nil {
		assert.Equal(t, currencyPrepaid, buyerReCredit.CurrencyType, "buyer reversal hits the prepaid wallet")
		assert.Equal(t, markedUpPrice, buyerReCredit.Amount,
			"buyer re-credit must be +markedUpPrice (inverse of the -markedUp debit)")
	}

	// 3. Seller is debited -listValue on the points wallet.
	assert.NotNil(t, sellerDebit, "seller reversal must be dispatched")
	if sellerDebit != nil {
		assert.Equal(t, currencyPoints, sellerDebit.CurrencyType, "seller reversal hits the points wallet")
		assert.Equal(t, -listValue, sellerDebit.Amount,
			"seller debit must be -listValue (inverse of the +listValue credit)")
	}

	// 4. Net currency change across forward + compensation is ZERO for each party.
	//    Buyer:  -markedUp (forward) + markedUp (reversal) = 0.
	//    Seller: +listVal  (forward) - listVal  (reversal) = 0.
	if buyerReCredit != nil {
		assert.Equal(t, int32(0), -markedUpPrice+buyerReCredit.Amount, "buyer nets to zero")
	}
	if sellerDebit != nil {
		assert.Equal(t, int32(0), listValue+sellerDebit.Amount, "seller nets to zero")
	}

	// 5. Single-custody invariant: the move step Failed (atomic tx committed
	//    nothing → listing still `active`, no holding created), so NO custody
	//    inverse is dispatched. Exactly one custody copy of the item exists.
	assert.Equal(t, 0, mtsMockP.restoreCalls, "no RestoreMtsHolding for a settle-purchase compensation")
	assert.Equal(t, 0, mtsMockP.moveCalls, "no MtsMoveListingToHolding re-dispatch during compensation")
	assert.Equal(t, 0, mtsMockP.acceptCalls, "no AcceptToMtsListing during a settle compensation")
	assert.Equal(t, 0, mtsMockP.releaseCalls, "no ReleaseFromMtsHolding during a settle compensation")

	// 6. Cache entry should be gone after eviction.
	_, lifecycleOk := GetCache().GetLifecycle(tctx, transactionId)
	assert.False(t, lifecycleOk, "saga should be evicted from cache after compensation")
}
