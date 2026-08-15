package saga

import (
	"testing"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus/hooks/test"
	"github.com/stretchr/testify/require"
)

// EventKindPetNameChanged must complete a pending rename_pet step. Without the
// event_acceptance.go wiring, AcceptEvent returns ok=false and the saga stalls
// until its timer fires — the rename would apply while the tag was never
// consumed, and the timeout backstop would then revert the name.
func TestRenamePetStepCompletesOnMatchingTransaction(t *testing.T) {
	logger, _ := test.NewNullLogger()
	ctx := testTenantContext()
	tx := uuid.New()

	s, err := NewBuilder().
		SetTransactionId(tx).
		SetSagaType(PetNameTagUse).
		SetInitiatedBy("pet-name-tag-accept-test").
		AddStep("rename_pet", Pending, RenamePet, RenamePetPayload{
			CharacterId: 77002, PetId: 4242, Name: "Renamed", PreviousName: "Original",
		}).
		AddStep("consume_pet_name_tag", Pending, DestroyAsset, DestroyAssetPayload{
			CharacterId: 77002, TemplateId: 5170000, Quantity: 1, RemoveAll: false,
		}).
		Build()
	require.NoError(t, err)
	require.NoError(t, GetCache().Put(ctx, s))

	p := NewProcessor(logger, ctx)

	_, ok := p.AcceptEvent(tx, EventKindPetNameChanged)
	require.True(t, ok, "EventKindPetNameChanged must be accepted for a pending rename_pet step")
}

// An event carrying a transaction id this orchestrator knows nothing about must
// complete nothing.
func TestRenamePetStepDoesNotCompleteOnMismatchedTransaction(t *testing.T) {
	logger, _ := test.NewNullLogger()
	ctx := testTenantContext()

	p := NewProcessor(logger, ctx)

	_, ok := p.AcceptEvent(uuid.New(), EventKindPetNameChanged)
	require.False(t, ok, "an unknown transaction id must not be accepted")
}
