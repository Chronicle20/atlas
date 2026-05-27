package saga

import (
	"context"
	"testing"

	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"github.com/sirupsen/logrus/hooks/test"
	"github.com/stretchr/testify/assert"
)

// TestAwaitInventoryCreated_BlocksAwardAssetUntilEvent verifies the race fix:
// AwardAsset must not dispatch until INVENTORY_STATUS.CREATED arrives.
//
// The pre-fix behaviour: orchestrator advanced create_character → award_item_0
// on CHARACTER_STATUS.CREATED. With the fix, the orchestrator advances
// create_character → await_inventory_created on CHARACTER_STATUS.CREATED,
// then advances await_inventory_created → award_item_0 only on
// INVENTORY_STATUS.CREATED.
func TestAwaitInventoryCreated_BlocksAwardAssetUntilEvent(t *testing.T) {
	l, _ := test.NewNullLogger()
	tnt, _ := tenant.Create(uuid.New(), "GMS", 83, 1)
	ctx := tenant.WithContext(context.Background(), tnt)

	txId := uuid.New()
	sg, err := NewBuilder().
		SetTransactionId(txId).
		SetSagaType(CharacterCreation).
		SetInitiatedBy("test").
		AddStep("create_character", Pending, CreateCharacter, CharacterCreatePayload{AccountId: 1, Name: "Test"}).
		AddStep("await_inventory_created", Pending, AwaitInventoryCreated, AwaitInventoryCreatedPayload{CharacterId: 0}).
		AddStep("award_item_0", Pending, AwardAsset, AwardItemActionPayload{CharacterId: 0, Item: ItemPayload{TemplateId: 2010000, Quantity: 1}}).
		Build()
	assert.NoError(t, err)

	p := NewProcessor(logrus.FieldLogger(l), ctx)
	assert.NoError(t, GetCache().Put(ctx, sg))

	// Mark create_character completed with a real characterId, exactly as
	// handleCharacterCreatedEvent would.
	_ = p.StepCompletedWithResult(txId, true, map[string]any{"characterId": uint32(42)})

	got, err := p.GetById(txId)
	assert.NoError(t, err)

	// Assertion 1: create_character is completed, await_inventory_created is
	// pending, award_item_0 is still pending (NOT completed, NOT advanced
	// past await).
	assertAwaitStepStatus(t, got, "create_character", Completed)
	assertAwaitStepStatus(t, got, "await_inventory_created", Pending)
	assertAwaitStepStatus(t, got, "award_item_0", Pending)

	// Assertion 2: result-forwarding substituted CharacterId=0 with 42 in
	// the await step's payload AND the award step's payload.
	awaitPl := findAwaitStep(t, got, "await_inventory_created").Payload().(AwaitInventoryCreatedPayload)
	assert.Equal(t, uint32(42), awaitPl.CharacterId)
	awardPl := findAwaitStep(t, got, "award_item_0").Payload().(AwardItemActionPayload)
	assert.Equal(t, uint32(42), awardPl.CharacterId)

	// Simulate INVENTORY_STATUS.CREATED arriving.
	if _, ok := p.AcceptEvent(txId, EventKindInventoryCreated); !ok {
		t.Fatalf("AcceptEvent(EventKindInventoryCreated) returned false; expected true for pending AwaitInventoryCreated step")
	}
	_ = p.StepCompleted(txId, true)

	// Assertion 3: now await is completed and award_item_0 has been dispatched
	// (still Pending — it transitions to Completed on EventKindAssetCreated,
	// which is out of this test's scope; what matters is the dispatcher was
	// able to advance past the await step).
	got, _ = p.GetById(txId)
	assertAwaitStepStatus(t, got, "await_inventory_created", Completed)
}

func TestAwaitInventoryCreated_FailEventCompensates(t *testing.T) {
	l, _ := test.NewNullLogger()
	tnt, _ := tenant.Create(uuid.New(), "GMS", 83, 1)
	ctx := tenant.WithContext(context.Background(), tnt)

	txId := uuid.New()
	sg, err := NewBuilder().
		SetTransactionId(txId).
		SetSagaType(CharacterCreation).
		SetInitiatedBy("test").
		AddStep("create_character", Completed, CreateCharacter, CharacterCreatePayload{AccountId: 1, Name: "Test"}).
		AddStep("await_inventory_created", Pending, AwaitInventoryCreated, AwaitInventoryCreatedPayload{CharacterId: 42}).
		Build()
	assert.NoError(t, err)

	p := NewProcessor(logrus.FieldLogger(l), ctx)
	assert.NoError(t, GetCache().Put(ctx, sg))

	// AcceptEvent gates the event: must return true for a pending AwaitInventoryCreated step.
	_, ok := p.AcceptEvent(txId, EventKindInventoryCreationFailed)
	assert.True(t, ok, "AcceptEvent(EventKindInventoryCreationFailed) should return true for pending AwaitInventoryCreated step")

	// Mark the step explicitly failed (as the consumer does via StepCompleted(false))
	// and verify the saga enters a Failing state before compensation runs.
	assert.NoError(t, p.MarkEarliestPendingStep(txId, Failed))

	got, err := p.GetById(txId)
	assert.NoError(t, err)
	assertAwaitStepStatus(t, got, "await_inventory_created", Failed)
	assert.True(t, got.Failing(), "saga should be in Failing state after inventory.creation_failed")
}

func assertAwaitStepStatus(t *testing.T, sg Saga, stepId string, want Status) {
	t.Helper()
	for _, st := range sg.Steps() {
		if st.StepId() == stepId {
			if st.Status() != want {
				t.Errorf("step %q status: got %q, want %q", stepId, st.Status(), want)
			}
			return
		}
	}
	t.Fatalf("step %q not found in saga", stepId)
}

func findAwaitStep(t *testing.T, sg Saga, stepId string) Step[any] {
	t.Helper()
	for _, st := range sg.Steps() {
		if st.StepId() == stepId {
			return st
		}
	}
	t.Fatalf("step %q not found", stepId)
	return Step[any]{}
}
