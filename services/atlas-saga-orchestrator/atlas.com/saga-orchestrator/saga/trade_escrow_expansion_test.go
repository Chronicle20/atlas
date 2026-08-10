package saga

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

// TestExpandTransferToTradeReleasesThenAccepts pins design §5A.4: staging one
// item expands to exactly [release_from_character, accept_to_trade], in that
// order, with the accept carrying the snapshot read from the owner's
// compartment.
//
// The order is the load-bearing part. If the accept ran first, atlas-trades
// would hold an escrow row for an asset still sitting in the owner's
// inventory — and the client would never receive the INVENTORY_OPERATION that
// clears m_bExclRequestSent, which is the entire reason this slice exists
// (design §5A.1).
func TestExpandTransferToTradeReleasesThenAccepts(t *testing.T) {
	const assetId = uint32(42)
	const templateId = uint32(1302000)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/vnd.api+json")
		_, _ = w.Write([]byte(inventoryCompartmentDoc("42", templateId)))
	}))
	defer srv.Close()
	t.Setenv("INVENTORY_SERVICE_URL", srv.URL+"/")

	txId := uuid.New()
	escrowId := uuid.New()
	roomId := uuid.New()
	payload := TransferToTradePayload{
		TransactionId:       txId,
		EscrowId:            escrowId,
		RoomId:              roomId,
		CharacterId:         1001,
		TradeSlot:           3,
		SourceInventoryType: 1,
		SourceSlot:          7,
		AssetId:             assetId,
		Quantity:            1,
	}
	st := NewStep[any]("transfer_to_trade-1", Pending, TransferToTrade, payload)

	p := newTestExpansionProcessor(t)
	steps, err := p.expandTransferToTrade(st)
	require.NoError(t, err)
	require.Len(t, steps, 2)

	require.Equal(t, ReleaseFromCharacter, steps[0].Action())
	require.Equal(t, "release_from_character", steps[0].StepId())
	rel, ok := steps[0].Payload().(ReleaseFromCharacterPayload)
	require.True(t, ok)
	require.Equal(t, assetId, rel.AssetId)
	require.Equal(t, uint32(1001), rel.CharacterId)
	require.Equal(t, byte(1), rel.InventoryType)
	require.Equal(t, uint32(1), rel.Quantity)

	require.Equal(t, AcceptToTrade, steps[1].Action())
	require.Equal(t, "accept_to_trade", steps[1].StepId())
	acc, ok := steps[1].Payload().(AcceptToTradePayload)
	require.True(t, ok)

	// Escrow identity — the row atlas-trades will write, and the room slot it backs.
	require.Equal(t, escrowId, acc.EscrowId)
	require.Equal(t, roomId, acc.RoomId)
	require.Equal(t, uint32(1001), acc.OwnerId)
	require.Equal(t, byte(3), acc.TradeSlot)
	require.Equal(t, byte(1), acc.SourceInventoryType)

	// Item snapshot, read from the compartment at expansion time. Slot is the
	// compartment's, not the composite's: the compartment is authoritative about
	// where the asset actually sat.
	require.Equal(t, templateId, acc.Snapshot.TemplateId)
	require.Equal(t, int16(1), acc.Snapshot.Slot)
	require.Equal(t, uint32(1), acc.Snapshot.Quantity)
	require.Equal(t, uint16(5), acc.Snapshot.Strength)
	require.Equal(t, uint16(7), acc.Snapshot.WeaponAttack)
	require.Equal(t, uint16(3), acc.Snapshot.Slots)
	require.Equal(t, uint16(2), acc.Snapshot.Flag)
	require.Equal(t, "Chronicle", acc.Snapshot.Owner)
}

// TestExpandTransferToTradeCarriesCashAndPetState pins the fields the bespoke
// stat list this replaced omitted: expiry, cash serial, rechargeable count, the
// item-level triple and the pet id.
//
// This is not hypothetical fidelity. Cash items and pets ARE stageable —
// atlas-trades' checkRestrictions blocks equipped items, the untradeable flags
// and the WZ tradeBlock, and nothing about the cash inventory — so an escrowed
// pet whose id never reached the escrow row came back out of the trade as an
// empty shell, and a timed item came back permanent.
func TestExpandTransferToTradeCarriesCashAndPetState(t *testing.T) {
	const templateId = uint32(5000000) // a pet template: cash inventory

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/vnd.api+json")
		_, _ = w.Write([]byte(cashInventoryCompartmentDoc("42", templateId)))
	}))
	defer srv.Close()
	t.Setenv("INVENTORY_SERVICE_URL", srv.URL+"/")

	st := NewStep[any]("transfer_to_trade-1", Pending, TransferToTrade, TransferToTradePayload{
		TransactionId:       uuid.New(),
		EscrowId:            uuid.New(),
		RoomId:              uuid.New(),
		CharacterId:         1001,
		SourceInventoryType: 5,
		AssetId:             42,
		Quantity:            1,
	})

	p := newTestExpansionProcessor(t)
	steps, err := p.expandTransferToTrade(st)
	require.NoError(t, err)
	require.Len(t, steps, 2)

	acc, ok := steps[1].Payload().(AcceptToTradePayload)
	require.True(t, ok)
	require.Equal(t, int64(4815162342), acc.Snapshot.CashId)
	require.Equal(t, uint32(909), acc.Snapshot.PetId)
	require.Equal(t, uint64(200), acc.Snapshot.Rechargeable)
	require.Equal(t, byte(1), acc.Snapshot.LevelType)
	require.Equal(t, byte(4), acc.Snapshot.Level)
	require.Equal(t, uint32(1234), acc.Snapshot.Experience)
	require.Equal(t, uint32(2), acc.Snapshot.HammersApplied)
	require.False(t, acc.Snapshot.Expiration.IsZero(), "a timed item that loses its expiry becomes permanent")

	// The same snapshot is what a settlement or an unwind re-grants from, so the
	// AssetData it produces must still carry all of it.
	ad := assetDataFromSnapshot(acc.Snapshot)
	require.Equal(t, int64(4815162342), ad.CashId)
	require.Equal(t, uint32(909), ad.PetId)
	require.Equal(t, uint64(200), ad.Rechargeable)
	require.Equal(t, uint32(2), ad.HammersApplied)
	require.Equal(t, acc.Snapshot.Expiration, ad.Expiration)
}

// cashInventoryCompartmentDoc is a compartment document for one CASH asset — a
// pet with a cash serial and an expiry. Separate from inventoryCompartmentDoc
// because those extra attributes are exactly what this file needs to observe and
// the MTS fixture has no use for them.
func cashInventoryCompartmentDoc(assetId string, templateId uint32) string {
	return `{
		"data": {
			"type": "compartments",
			"id": "comp-1",
			"attributes": {"type": 5, "capacity": 24},
			"relationships": {
				"assets": {"data": [{"type": "assets", "id": "` + assetId + `"}]}
			}
		},
		"included": [
			{
				"type": "assets",
				"id": "` + assetId + `",
				"attributes": {
					"slot": 2,
					"templateId": ` + itoa(templateId) + `,
					"quantity": 1,
					"expiration": "2031-04-05T06:07:08Z",
					"cashId": "4815162342",
					"rechargeable": 200,
					"levelType": 1,
					"level": 4,
					"experience": 1234,
					"hammersApplied": 2,
					"petId": 909,
					"owner": "Chronicle"
				}
			}
		]
	}`
}

// TestExpandTransferToTradeStagesTheRequestedQuantity pins the partial-stack
// case: the accept records the STAGED quantity, not the whole stack's. A
// snapshot that copied the compartment's quantity would escrow — and then
// deliver — the entire stack when the player staged one of forty.
func TestExpandTransferToTradeStagesTheRequestedQuantity(t *testing.T) {
	const templateId = uint32(2000000)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/vnd.api+json")
		_, _ = w.Write([]byte(inventoryCompartmentDoc("42", templateId)))
	}))
	defer srv.Close()
	t.Setenv("INVENTORY_SERVICE_URL", srv.URL+"/")

	st := NewStep[any]("transfer_to_trade-1", Pending, TransferToTrade, TransferToTradePayload{
		TransactionId:       uuid.New(),
		EscrowId:            uuid.New(),
		RoomId:              uuid.New(),
		CharacterId:         1001,
		SourceInventoryType: 2,
		AssetId:             42,
		Quantity:            5,
	})

	p := newTestExpansionProcessor(t)
	steps, err := p.expandTransferToTrade(st)
	require.NoError(t, err)
	require.Len(t, steps, 2)

	rel, ok := steps[0].Payload().(ReleaseFromCharacterPayload)
	require.True(t, ok)
	require.Equal(t, uint32(5), rel.Quantity)

	acc, ok := steps[1].Payload().(AcceptToTradePayload)
	require.True(t, ok)
	require.Equal(t, uint32(5), acc.Snapshot.Quantity)
}

// TestExpandTransferToTradeFailsWhenTheAssetIsGone pins the race that
// escrow-at-staging makes possible: the player dropped or used the item
// between the client's PUT_ITEM and expansion. Expansion must fail loudly so
// the stage is refused (and the client unlocked, design §5A.6) rather than
// producing an escrow row for an asset that no longer exists.
func TestExpandTransferToTradeFailsWhenTheAssetIsGone(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/vnd.api+json")
		_, _ = w.Write([]byte(inventoryCompartmentDoc("42", 1302000)))
	}))
	defer srv.Close()
	t.Setenv("INVENTORY_SERVICE_URL", srv.URL+"/")

	st := NewStep[any]("transfer_to_trade-1", Pending, TransferToTrade, TransferToTradePayload{
		TransactionId:       uuid.New(),
		EscrowId:            uuid.New(),
		RoomId:              uuid.New(),
		CharacterId:         1001,
		SourceInventoryType: 1,
		AssetId:             999, // not in the compartment
		Quantity:            1,
	})

	p := newTestExpansionProcessor(t)
	_, err := p.expandTransferToTrade(st)
	require.Error(t, err)
}
