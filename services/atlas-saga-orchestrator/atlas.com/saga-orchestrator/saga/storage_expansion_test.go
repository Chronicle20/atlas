package saga

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

// transferToStorageFixture deposits `quantity` of a 200-stack sitting in the
// character's USE compartment at slot 2.
func transferToStorageFixture(quantity uint32) TransferToStoragePayload {
	return TransferToStoragePayload{
		TransactionId:       uuid.MustParse("44444444-4444-4444-4444-444444444444"),
		CharacterId:         100,
		WorldId:             1,
		AccountId:           7,
		SourceSlot:          2,
		SourceInventoryType: 2,
		Quantity:            quantity,
	}
}

func storageDepositStack() []testAsset {
	return []testAsset{
		{CharacterId: 100, InventoryType: 2, Slot: 2, TemplateId: 2000005, Quantity: 200, Id: "2"},
	}
}

// TestExpandTransferToStorageAcceptsOnlyTheDepositedQuantity pins the same
// defect the trade settlement had: accept_to_storage RECREATES the asset from
// AssetData, so AssetData.Quantity is what lands in storage. The snapshot is of
// the whole SOURCE STACK, so depositing 1 of a 200 stack released 1 from the
// character and stored 200 — minting 199.
func TestExpandTransferToStorageAcceptsOnlyTheDepositedQuantity(t *testing.T) {
	p := testProcessorWithCompartments(t, storageDepositStack())

	steps, err := p.expandTransferToStorage(NewStep[any]("transfer_to_storage", Pending, TransferToStorage, transferToStorageFixture(1)))
	require.NoError(t, err)
	require.Len(t, steps, 2)

	release, ok := steps[0].Payload().(ReleaseFromCharacterPayload)
	require.True(t, ok, "first step payload must be ReleaseFromCharacterPayload, got %T", steps[0].Payload())
	require.Equal(t, uint32(1), release.Quantity, "release must take only the deposited quantity")

	accept, ok := steps[1].Payload().(AcceptToStoragePayload)
	require.True(t, ok, "second step payload must be AcceptToStoragePayload, got %T", steps[1].Payload())
	require.Equal(t, uint32(1), accept.AssetData.Quantity, "accept must store the DEPOSITED quantity, not the source stack")
}

// TestExpandTransferToStorageTreatsZeroAsTheWholeStack pins the "take all"
// convention expandWithdrawFromStorage already uses: a caller that names no
// quantity deposits the whole stack, so the snapshot's own quantity stands. The
// quantity override must not turn that into a zero-quantity asset.
func TestExpandTransferToStorageTreatsZeroAsTheWholeStack(t *testing.T) {
	p := testProcessorWithCompartments(t, storageDepositStack())

	steps, err := p.expandTransferToStorage(NewStep[any]("transfer_to_storage", Pending, TransferToStorage, transferToStorageFixture(0)))
	require.NoError(t, err)
	require.Len(t, steps, 2)

	accept, ok := steps[1].Payload().(AcceptToStoragePayload)
	require.True(t, ok, "second step payload must be AcceptToStoragePayload, got %T", steps[1].Payload())
	require.Equal(t, uint32(200), accept.AssetData.Quantity)
}

// TestExpandTransferToStorageKeepsTheSnapshotsNonQuantityFields pins that the
// quantity override leaves the rest of the snapshot alone — the deposit must
// still carry cash ownership and rolled stats into storage.
func TestExpandTransferToStorageKeepsTheSnapshotsNonQuantityFields(t *testing.T) {
	p := testProcessorWithCompartments(t, storageDepositStack())

	steps, err := p.expandTransferToStorage(NewStep[any]("transfer_to_storage", Pending, TransferToStorage, transferToStorageFixture(1)))
	require.NoError(t, err)

	accept, ok := steps[1].Payload().(AcceptToStoragePayload)
	require.True(t, ok, "second step payload must be AcceptToStoragePayload, got %T", steps[1].Payload())
	require.Equal(t, "Chronicle", accept.AssetData.Owner)
	require.Equal(t, uint32(2000005), accept.TemplateId)
}
