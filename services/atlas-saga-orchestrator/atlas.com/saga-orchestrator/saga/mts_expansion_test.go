package saga

import (
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	"github.com/Chronicle20/atlas/libs/atlas-constants/inventory"
	"github.com/Chronicle20/atlas/libs/atlas-constants/item"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus/hooks/test"
	"github.com/stretchr/testify/require"
)

// newTestExpansionProcessor builds a *ProcessorImpl wired only with a tenant
// context — the expansion functions under test only use the REST clients
// (compartment.RequestCompartment / mts.RequestHoldings), which resolve their
// base URL from the *_SERVICE_URL env vars, so no processor deps are needed.
func newTestExpansionProcessor(t *testing.T) *ProcessorImpl {
	t.Helper()
	_, tctx := setupContext()
	logger, _ := test.NewNullLogger()
	p, ok := NewProcessor(logger, tctx).(*ProcessorImpl)
	require.True(t, ok, "NewProcessor must return *ProcessorImpl")
	return p
}

// inventoryCompartmentDoc is a JSON:API compartment with one equipment asset,
// matching the orchestrator's compartment.CompartmentRestModel (assets is a
// toMany relationship materialized from the `included` block).
func inventoryCompartmentDoc(assetId string, templateId uint32) string {
	return `{
		"data": {
			"type": "compartments",
			"id": "comp-1",
			"attributes": {"type": 1, "capacity": 24},
			"relationships": {
				"assets": {"data": [{"type": "assets", "id": "` + assetId + `"}]}
			}
		},
		"included": [
			{
				"type": "assets",
				"id": "` + assetId + `",
				"attributes": {
					"slot": 1,
					"templateId": ` + itoa(templateId) + `,
					"quantity": 1,
					"strength": 5,
					"weaponAttack": 7,
					"slots": 3,
					"flag": 2,
					"owner": "Chronicle"
				}
			}
		]
	}`
}

func itoa(v uint32) string {
	return strconv.FormatUint(uint64(v), 10)
}

// TestExpandTransferToMts asserts TransferToMts expands to
// [release_from_character, accept_to_mts_listing] and the accept step carries
// the looked-up item snapshot plus the seller's sale params.
func TestExpandTransferToMts(t *testing.T) {
	const assetId = uint32(42)
	const templateId = uint32(1302000)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/vnd.api+json")
		_, _ = w.Write([]byte(inventoryCompartmentDoc("42", templateId)))
	}))
	defer srv.Close()
	t.Setenv("INVENTORY_SERVICE_URL", srv.URL+"/")

	listingId := uuid.New()
	txId := uuid.New()
	buyNow := uint32(5000)
	payload := TransferToMtsPayload{
		TransactionId:       txId,
		CharacterId:         1001,
		WorldId:             0,
		SourceInventoryType: 1,
		AssetId:             assetId,
		Quantity:            1,
		ListingId:           listingId,
		SellerName:          "Seller",
		SaleType:            "auction",
		ListValue:           1000,
		BuyNowPrice:         &buyNow,
		CommissionRate:      0.10,
		Category:            "weapon",
		SubCategory:         "onehand",
		MinIncrement:        100,
	}
	st := NewStep[any]("transfer_to_mts-1", Pending, TransferToMts, payload)

	p := newTestExpansionProcessor(t)
	steps, err := p.expandTransferToMts(st)
	require.NoError(t, err)
	require.Len(t, steps, 2)

	require.Equal(t, ReleaseFromCharacter, steps[0].Action())
	require.Equal(t, "release_from_character", steps[0].StepId())
	rel, ok := steps[0].Payload().(ReleaseFromCharacterPayload)
	require.True(t, ok)
	require.Equal(t, assetId, rel.AssetId)
	require.Equal(t, uint32(1001), rel.CharacterId)

	require.Equal(t, AcceptToMtsListing, steps[1].Action())
	require.Equal(t, "accept_to_mts_listing", steps[1].StepId())
	acc, ok := steps[1].Payload().(AcceptToMtsListingPayload)
	require.True(t, ok)
	// snapshot
	require.Equal(t, templateId, acc.TemplateId)
	require.Equal(t, uint16(5), acc.Strength)
	require.Equal(t, uint16(7), acc.WeaponAttack)
	require.Equal(t, uint16(3), acc.Slots)
	require.Equal(t, uint16(2), acc.Flags)
	require.Equal(t, "Chronicle", acc.Owner)
	// identity + sale params
	require.Equal(t, listingId, acc.ListingId)
	require.Equal(t, uint32(1001), acc.SellerId)
	require.Equal(t, "auction", acc.SaleType)
	require.Equal(t, uint32(1000), acc.ListValue)
	require.NotNil(t, acc.BuyNowPrice)
	require.Equal(t, uint32(5000), *acc.BuyNowPrice)
	require.Equal(t, 0.10, acc.CommissionRate)
	require.Equal(t, "weapon", acc.Category)
	require.Equal(t, uint32(100), acc.MinIncrement)
}

// TestExpandWithdrawFromMts asserts WithdrawFromMts expands to
// [release_from_mts_holding, accept_to_character] and the accept step carries
// the holding's item snapshot.
func TestExpandWithdrawFromMts(t *testing.T) {
	holdingId := uuid.New()
	const templateId = uint32(1402001)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/vnd.api+json")
		_, _ = w.Write([]byte(`{
			"data": [
				{
					"type": "holdings",
					"id": "` + holdingId.String() + `",
					"attributes": {
						"worldId": 0,
						"ownerId": 1001,
						"origin": "purchased",
						"templateId": ` + itoa(templateId) + `,
						"quantity": 1,
						"strength": 9,
						"weaponAttack": 11,
						"slots": 7,
						"flags": 4
					}
				}
			]
		}`))
	}))
	defer srv.Close()
	t.Setenv("MTS_SERVICE_URL", srv.URL+"/")

	txId := uuid.New()
	payload := WithdrawFromMtsPayload{
		TransactionId: txId,
		CharacterId:   1001,
		WorldId:       0,
		HoldingId:     holdingId,
		// The channel passes 0 (advisory placeholder); 0 matches no compartment, so
		// the expansion MUST derive the real type from the holding's template (a
		// weapon, 1402001 -> equip type 1), NOT pass this 0 through.
		InventoryType: 0,
	}
	st := NewStep[any]("withdraw_from_mts-1", Pending, WithdrawFromMts, payload)

	p := newTestExpansionProcessor(t)
	steps, err := p.expandWithdrawFromMts(st)
	require.NoError(t, err)
	require.Len(t, steps, 2)

	require.Equal(t, ReleaseFromMtsHolding, steps[0].Action())
	require.Equal(t, "release_from_mts_holding", steps[0].StepId())
	rel, ok := steps[0].Payload().(ReleaseFromMtsHoldingPayload)
	require.True(t, ok)
	require.Equal(t, holdingId, rel.HoldingId)

	require.Equal(t, AcceptToCharacter, steps[1].Action())
	require.Equal(t, "accept_to_character", steps[1].StepId())
	acc, ok := steps[1].Payload().(AcceptToCharacterPayload)
	require.True(t, ok)
	require.Equal(t, uint32(1001), acc.CharacterId)
	require.Equal(t, templateId, acc.TemplateId)
	// The accept step's inventory type is derived from the template, not the
	// passed-through advisory 0. 1402001 is a one-handed sword -> equip (type 1).
	expectedType, ok := inventory.TypeFromItemId(item.Id(templateId))
	require.True(t, ok)
	require.Equal(t, byte(expectedType), acc.InventoryType)
	require.NotZero(t, acc.InventoryType, "inventory type must be the template-derived non-zero type, not the advisory 0")
	require.Equal(t, uint16(9), acc.AssetData.Strength)
	require.Equal(t, uint16(11), acc.AssetData.WeaponAttack)
	require.Equal(t, uint16(7), acc.AssetData.Slots)
	require.Equal(t, uint16(4), acc.AssetData.Flag)
}

// TestExpandMtsSettlePurchase asserts the three ordered settlement steps:
// debit buyer prepaid (−markedUp), credit seller points (+listValue), then move
// listing custody to the buyer holding — IN THAT ORDER (debit-first).
func TestExpandMtsSettlePurchase(t *testing.T) {
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
	st := NewStep[any]("mts_settle_purchase-1", Pending, MtsSettlePurchase, payload)

	p := newTestExpansionProcessor(t)
	steps, err := p.expandMtsSettlePurchase(st)
	require.NoError(t, err)
	require.Len(t, steps, 3)

	// 1. Debit buyer prepaid FIRST.
	require.Equal(t, AwardCurrency, steps[0].Action())
	buyer, ok := steps[0].Payload().(AwardCurrencyPayload)
	require.True(t, ok)
	require.Equal(t, uint32(100), buyer.CharacterId)
	require.Equal(t, uint32(10), buyer.AccountId)
	require.Equal(t, uint32(3), buyer.CurrencyType) // prepaid
	require.Equal(t, int32(-1100), buyer.Amount)    // negative markedUpPrice

	// 2. Credit seller points.
	require.Equal(t, AwardCurrency, steps[1].Action())
	seller, ok := steps[1].Payload().(AwardCurrencyPayload)
	require.True(t, ok)
	require.Equal(t, uint32(200), seller.CharacterId)
	require.Equal(t, uint32(20), seller.AccountId)
	require.Equal(t, uint32(2), seller.CurrencyType) // points
	require.Equal(t, int32(1000), seller.Amount)     // positive listValue

	// 3. Move listing custody to buyer holding LAST.
	require.Equal(t, MtsMoveListingToHolding, steps[2].Action())
	move, ok := steps[2].Payload().(MtsMoveListingToHoldingPayload)
	require.True(t, ok)
	require.Equal(t, listingId, move.ListingId)
	require.Equal(t, uint32(100), move.BuyerId)
}

// TestIsExpandableActionCoversExpansionSwitch pins the composite-expansion GATE
// (isExpandableAction, used by Step()) to the set of actions that
// expandAndProcessStep actually expands. The MTS list/settle flow regressed
// because the expansion function + switch case existed but this gate didn't
// list the MTS composites, so a transfer_to_mts step fell through to GetHandler
// and failed at runtime with "unknown action type" — while the expansion unit
// tests (which call expand* directly) stayed green. This guards that gap.
func TestIsExpandableActionCoversExpansionSwitch(t *testing.T) {
	composites := []Action{
		TransferToStorage, WithdrawFromStorage,
		TransferToCashShop, WithdrawFromCashShop,
		TransferToMts, WithdrawFromMts, MtsSettlePurchase,
	}
	for _, a := range composites {
		require.Truef(t, isExpandableAction(a), "composite action %q must be routed to expansion by the Step() gate", a)
	}
	// Atomic actions (dispatched via GetHandler, not expanded) must NOT be gated
	// into expansion — e.g. the MTS custody steps the composites expand into.
	for _, a := range []Action{AcceptToMtsListing, ReleaseFromMtsHolding, MtsMoveListingToHolding} {
		require.Falsef(t, isExpandableAction(a), "atomic action %q must not be routed to expansion", a)
	}
}
