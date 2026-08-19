package saga

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

// parcelCompartmentDoc is a JSON:API compartment with one equipment asset,
// matching the orchestrator's compartment.CompartmentRestModel (assets is a
// toMany relationship materialized from the `included` block). Mirrors
// inventoryCompartmentDoc in mts_expansion_test.go but with an owner field
// distinct enough to pin the accept step's Owner passthrough.
func parcelCompartmentDoc(assetId string, templateId uint32, owner string) string {
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
					"owner": "` + owner + `"
				}
			}
		]
	}`
}

// TestExpandTransferToParcel asserts TransferToParcel expands to
// [release_from_character, accept_to_parcel] for an item parcel, to a single
// [accept_to_parcel] step (HasItem=false) for a meso-only parcel, and errors
// without emitting steps when the asset or the payload type doesn't match.
func TestExpandTransferToParcel(t *testing.T) {
	t.Run("item parcel", func(t *testing.T) {
		const assetId = uint32(42)
		const templateId = uint32(1302000)

		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/vnd.api+json")
			_, _ = w.Write([]byte(parcelCompartmentDoc("42", templateId, "Alice")))
		}))
		defer srv.Close()
		t.Setenv("INVENTORY_SERVICE_URL", srv.URL+"/")

		txId := uuid.New()
		parcelId := uuid.New()
		payload := TransferToParcelPayload{
			TransactionId:       txId,
			ParcelId:            parcelId,
			CharacterId:         1001,
			WorldId:             0,
			SourceInventoryType: 1,
			AssetId:             assetId,
			Quantity:            1,
			SenderAccountId:     10,
			SenderName:          "Sender",
			RecipientId:         2002,
			RecipientAccountId:  20,
			MesoAmount:          500,
			FeePaid:             100,
			Quick:               true,
			Message:             "hello",
		}
		st := NewStep[any]("transfer_to_parcel-1", Pending, TransferToParcel, payload)

		p := newTestExpansionProcessor(t)
		steps, err := p.expandTransferToParcel(st)
		require.NoError(t, err)
		require.Len(t, steps, 2)

		require.Equal(t, ReleaseFromCharacter, steps[0].Action())
		require.Equal(t, "release_from_character", steps[0].StepId())
		rel, ok := steps[0].Payload().(ReleaseFromCharacterPayload)
		require.True(t, ok)
		require.Equal(t, assetId, rel.AssetId)
		require.Equal(t, uint32(1001), rel.CharacterId)

		require.Equal(t, AcceptToParcel, steps[1].Action())
		require.Equal(t, "accept_to_parcel", steps[1].StepId())
		acc, ok := steps[1].Payload().(AcceptToParcelPayload)
		require.True(t, ok)
		require.True(t, acc.HasItem)
		require.Equal(t, templateId, acc.TemplateId)
		require.Equal(t, uint16(5), acc.Strength)
		require.Equal(t, "Alice", acc.Owner)
		require.Equal(t, parcelId, acc.ParcelId)
		require.Equal(t, uint32(2002), acc.RecipientId)
		require.Equal(t, uint32(500), acc.MesoAmount)
	})

	t.Run("meso only", func(t *testing.T) {
		txId := uuid.New()
		parcelId := uuid.New()
		payload := TransferToParcelPayload{
			TransactionId:       txId,
			ParcelId:            parcelId,
			CharacterId:         1001,
			WorldId:             0,
			SourceInventoryType: 1,
			AssetId:             0,
			Quantity:            0,
			MesoAmount:          500,
			RecipientId:         2002,
		}
		st := NewStep[any]("transfer_to_parcel-1", Pending, TransferToParcel, payload)

		p := newTestExpansionProcessor(t)
		steps, err := p.expandTransferToParcel(st)
		require.NoError(t, err)
		require.Len(t, steps, 1)

		require.Equal(t, AcceptToParcel, steps[0].Action())
		require.Equal(t, "accept_to_parcel", steps[0].StepId())
		acc, ok := steps[0].Payload().(AcceptToParcelPayload)
		require.True(t, ok)
		require.False(t, acc.HasItem)
		require.Zero(t, acc.TemplateId)
		require.Zero(t, acc.Strength)
		require.Equal(t, uint32(500), acc.MesoAmount)
	})

	t.Run("asset missing", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/vnd.api+json")
			_, _ = w.Write([]byte(parcelCompartmentDoc("42", 1302000, "Alice")))
		}))
		defer srv.Close()
		t.Setenv("INVENTORY_SERVICE_URL", srv.URL+"/")

		payload := TransferToParcelPayload{
			TransactionId:       uuid.New(),
			ParcelId:            uuid.New(),
			CharacterId:         1001,
			SourceInventoryType: 1,
			AssetId:             99,
		}
		st := NewStep[any]("transfer_to_parcel-1", Pending, TransferToParcel, payload)

		p := newTestExpansionProcessor(t)
		steps, err := p.expandTransferToParcel(st)
		require.Error(t, err)
		require.Nil(t, steps)
	})

	t.Run("wrong payload type", func(t *testing.T) {
		st := NewStep[any]("transfer_to_parcel-1", Pending, TransferToParcel, TransferToMtsPayload{})

		p := newTestExpansionProcessor(t)
		steps, err := p.expandTransferToParcel(st)
		require.Error(t, err)
		require.EqualError(t, err, "invalid payload type for TransferToParcel")
		require.Nil(t, steps)
	})
}

// TestExpandWithdrawFromParcel asserts WithdrawFromParcel expands to
// [release_from_parcel, accept_to_character].
func TestExpandWithdrawFromParcel(t *testing.T) {
	txId := uuid.New()
	parcelId := uuid.New()
	payload := WithdrawFromParcelPayload{
		TransactionId: txId,
		ParcelId:      parcelId,
		CharacterId:   100,
		WorldId:       0,
		InventoryType: 1,
	}
	st := NewStep[any]("withdraw_from_parcel-1", Pending, WithdrawFromParcel, payload)

	p := newTestExpansionProcessor(t)
	steps, err := p.expandWithdrawFromParcel(st)
	require.NoError(t, err)
	require.Len(t, steps, 2)

	require.Equal(t, ReleaseFromParcel, steps[0].Action())
	require.Equal(t, "release_from_parcel", steps[0].StepId())
	rel, ok := steps[0].Payload().(ReleaseFromParcelPayload)
	require.True(t, ok)
	require.Equal(t, parcelId, rel.ParcelId)
	require.Equal(t, uint32(100), rel.RecipientId)

	require.Equal(t, AcceptToCharacter, steps[1].Action())
	require.Equal(t, "accept_to_character", steps[1].StepId())
	acc, ok := steps[1].Payload().(AcceptToCharacterPayload)
	require.True(t, ok)
	require.Equal(t, uint32(100), acc.CharacterId)
	require.Equal(t, byte(1), acc.InventoryType)
}
