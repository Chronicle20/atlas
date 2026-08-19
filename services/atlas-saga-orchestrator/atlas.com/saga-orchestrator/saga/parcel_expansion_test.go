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

// parcelDoc is a JSON:API single-resource doc matching the orchestrator's
// parcel.RestModel (no relationships, so no `included` block). itemId is the
// literal JSON fragment for the "itemId" attribute — "null" for a meso-only
// parcel, or a bare integer for an item parcel — since RestModel.ItemId is a
// *uint32 and api2go/jsonapi decodes JSON null into a nil pointer.
func parcelDoc(parcelId string, itemIdJSON string, templateId uint32, quantity uint16, strength uint16, owner string) string {
	return `{
		"data": {
			"type": "parcels",
			"id": "` + parcelId + `",
			"attributes": {
				"worldId": 0,
				"senderId": 1,
				"senderAccountId": 1,
				"senderName": "Sender",
				"recipientId": 100,
				"recipientAccountId": 10,
				"message": "hello",
				"mesoAmount": 500,
				"feePaid": 100,
				"itemId": ` + itemIdJSON + `,
				"itemType": 1,
				"quantity": ` + itoa(uint32(quantity)) + `,
				"itemSnapshot": {
					"quantity": ` + itoa(uint32(quantity)) + `,
					"strength": ` + itoa(uint32(strength)) + `,
					"owner": "` + owner + `"
				},
				"status": "pending",
				"quick": false,
				"returned": false
			}
		}
	}`
}

// TestExpandWithdrawFromParcel asserts WithdrawFromParcel expands to
// [release_from_parcel, accept_to_character] when the parcel row holds an
// item, to a single [release_from_parcel] step when it is meso-only (design
// §12 RISK-2 — nothing to grant into inventory, so accept_to_character would
// carry a meaningless zero-valued item), and errors without emitting steps
// when the parcel lookup fails.
func TestExpandWithdrawFromParcel(t *testing.T) {
	t.Run("withdraw with item", func(t *testing.T) {
		const templateId = uint32(1302000)
		parcelId := uuid.New()

		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/vnd.api+json")
			_, _ = w.Write([]byte(parcelDoc(parcelId.String(), itoa(templateId), templateId, 1, 9, "Bob")))
		}))
		defer srv.Close()
		t.Setenv("PARCEL_SERVICE_URL", srv.URL+"/")

		txId := uuid.New()
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
		// The real snapshot from the parcel row, not zeros (this is the defect
		// the fix round exists to close: an item withdrawn from a parcel must
		// not be handed to the character as template 0).
		require.Equal(t, templateId, acc.TemplateId)
		require.Equal(t, uint16(9), acc.AssetData.Strength)
		require.Equal(t, "Bob", acc.AssetData.Owner)
		require.Equal(t, uint32(1), acc.AssetData.Quantity)
	})

	t.Run("withdraw meso only", func(t *testing.T) {
		parcelId := uuid.New()

		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/vnd.api+json")
			_, _ = w.Write([]byte(parcelDoc(parcelId.String(), "null", 0, 0, 0, "")))
		}))
		defer srv.Close()
		t.Setenv("PARCEL_SERVICE_URL", srv.URL+"/")

		payload := WithdrawFromParcelPayload{
			TransactionId: uuid.New(),
			ParcelId:      parcelId,
			CharacterId:   100,
			WorldId:       0,
			InventoryType: 1,
		}
		st := NewStep[any]("withdraw_from_parcel-1", Pending, WithdrawFromParcel, payload)

		p := newTestExpansionProcessor(t)
		steps, err := p.expandWithdrawFromParcel(st)
		require.NoError(t, err)
		// Meso-only: only release_from_parcel is emitted. No accept_to_character
		// carrying a zero-valued item — mirrors expandTransferToParcel's
		// HasItem=false branch on the send side (RISK-2).
		require.Len(t, steps, 1)
		require.Equal(t, ReleaseFromParcel, steps[0].Action())
		rel, ok := steps[0].Payload().(ReleaseFromParcelPayload)
		require.True(t, ok)
		require.Equal(t, parcelId, rel.ParcelId)
		require.Equal(t, uint32(100), rel.RecipientId)
	})

	t.Run("withdraw parcel lookup fails", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusNotFound)
		}))
		defer srv.Close()
		t.Setenv("PARCEL_SERVICE_URL", srv.URL+"/")

		payload := WithdrawFromParcelPayload{
			TransactionId: uuid.New(),
			ParcelId:      uuid.New(),
			CharacterId:   100,
			WorldId:       0,
			InventoryType: 1,
		}
		st := NewStep[any]("withdraw_from_parcel-1", Pending, WithdrawFromParcel, payload)

		p := newTestExpansionProcessor(t)
		steps, err := p.expandWithdrawFromParcel(st)
		require.Error(t, err)
		require.Nil(t, steps)
	})
}
