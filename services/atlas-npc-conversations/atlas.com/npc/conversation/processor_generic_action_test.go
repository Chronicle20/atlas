package conversation

import (
	"context"
	"errors"
	"testing"

	"github.com/Chronicle20/atlas/libs/atlas-constants/channel"
	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"
	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
	"github.com/alicebob/miniredis/v2"
	goredis "github.com/redis/go-redis/v9"
	"github.com/sirupsen/logrus"
	"github.com/sirupsen/logrus/hooks/test"
)

// recordingExecutor captures how processGenericActionState invokes the executor.
// ExecuteOperations should be called exactly once per state with the full slice;
// ExecuteOperation (singular) should never be called from that path, because
// per-operation sagas defeat the batching / cross-reference logic in
// createSagaForOperations (e.g. attaching award_asset rewards to complete_quest).
type recordingExecutor struct {
	singularCalls [][]OperationModel
	batchCalls    [][]OperationModel
	batchErr      error
}

func (r *recordingExecutor) ExecuteOperation(_ field.Model, _ uint32, op OperationModel) error {
	r.singularCalls = append(r.singularCalls, []OperationModel{op})
	return nil
}

func (r *recordingExecutor) ExecuteOperations(_ field.Model, _ uint32, ops []OperationModel) error {
	captured := make([]OperationModel, len(ops))
	copy(captured, ops)
	r.batchCalls = append(r.batchCalls, captured)
	return r.batchErr
}

func buildRewardsState(t *testing.T) StateModel {
	t.Helper()

	awardExp, err := NewOperationBuilder().
		SetType("award_exp").
		AddParamValue("amount", "10").
		Build()
	if err != nil {
		t.Fatalf("build award_exp: %v", err)
	}
	awardItemA, err := NewOperationBuilder().
		SetType("award_item").
		AddParamValue("itemId", "2010000").
		AddParamValue("quantity", "3").
		Build()
	if err != nil {
		t.Fatalf("build award_item A: %v", err)
	}
	awardItemB, err := NewOperationBuilder().
		SetType("award_item").
		AddParamValue("itemId", "2010009").
		AddParamValue("quantity", "3").
		Build()
	if err != nil {
		t.Fatalf("build award_item B: %v", err)
	}
	completeQuest, err := NewOperationBuilder().
		SetType("complete_quest").
		Build()
	if err != nil {
		t.Fatalf("build complete_quest: %v", err)
	}

	terminalOutcome, err := NewOutcomeBuilder().SetNextState("").Build()
	if err != nil {
		t.Fatalf("build outcome: %v", err)
	}

	ga, err := NewGenericActionBuilder().
		AddOperation(awardExp).
		AddOperation(awardItemA).
		AddOperation(awardItemB).
		AddOperation(completeQuest).
		AddOutcome(terminalOutcome).
		Build()
	if err != nil {
		t.Fatalf("build genericAction: %v", err)
	}

	state, err := NewStateBuilder().
		SetId("giveRewards").
		SetGenericAction(ga).
		Build()
	if err != nil {
		t.Fatalf("build state: %v", err)
	}
	return state
}

func newTestProcessor(t *testing.T, executor OperationExecutor) (*ProcessorImpl, ConversationContext) {
	t.Helper()

	mr := miniredis.RunT(t)
	rc := goredis.NewClient(&goredis.Options{Addr: mr.Addr()})
	InitRegistry(rc)

	l, _ := test.NewNullLogger()
	l.SetLevel(logrus.DebugLevel)

	var tm tenant.Model
	tctx := tenant.WithContext(context.Background(), tm)

	f := field.NewBuilder(world.Id(0), channel.Id(1), _map.Id(20000)).Build()

	ctx := NewConversationContextBuilder().
		SetField(f).
		SetCharacterId(7).
		SetNpcId(2000).
		SetCurrentState("giveRewards").
		Build()

	p := &ProcessorImpl{
		l:        l,
		ctx:      tctx,
		t:        tm,
		executor: executor,
	}
	return p, ctx
}

// A single genericAction state with N operations must batch into exactly one
// ExecuteOperations call. Quest 1021's giveRewards state is the canonical
// example: award_exp, award_item, award_item, complete_quest must share a saga
// so PR #229's reward-forwarding cross-reference fires.
func TestProcessGenericActionState_BatchesOperationsIntoSingleExecutorCall(t *testing.T) {
	exec := &recordingExecutor{}
	p, ctx := newTestProcessor(t, exec)
	state := buildRewardsState(t)

	next, err := p.processGenericActionState(ctx, state)
	if err != nil {
		t.Fatalf("processGenericActionState: unexpected error: %v", err)
	}
	if next != "" {
		t.Errorf("next state = %q, want terminal (empty)", next)
	}

	if got := len(exec.singularCalls); got != 0 {
		t.Errorf("ExecuteOperation (singular) called %d times; per-operation sagas break batching and must not be used", got)
	}

	if got := len(exec.batchCalls); got != 1 {
		t.Fatalf("ExecuteOperations called %d times, want 1", got)
	}
	ops := exec.batchCalls[0]
	if got := len(ops); got != 4 {
		t.Fatalf("batch size = %d, want 4", got)
	}
	wantTypes := []string{"award_exp", "award_item", "award_item", "complete_quest"}
	for i, want := range wantTypes {
		if got := ops[i].Type(); got != want {
			t.Errorf("ops[%d].Type = %q, want %q", i, got, want)
		}
	}
}

// Empty operations list must not reach the executor at all.
func TestProcessGenericActionState_SkipsExecutorWhenNoOperations(t *testing.T) {
	exec := &recordingExecutor{}
	p, ctx := newTestProcessor(t, exec)

	outcome, err := NewOutcomeBuilder().SetNextState("someNextState").Build()
	if err != nil {
		t.Fatalf("build outcome: %v", err)
	}
	ga, err := NewGenericActionBuilder().AddOutcome(outcome).Build()
	if err != nil {
		t.Fatalf("build genericAction: %v", err)
	}
	state, err := NewStateBuilder().SetId("noops").SetGenericAction(ga).Build()
	if err != nil {
		t.Fatalf("build state: %v", err)
	}

	next, err := p.processGenericActionState(ctx, state)
	if err != nil {
		t.Fatalf("processGenericActionState: unexpected error: %v", err)
	}
	if next != "someNextState" {
		t.Errorf("next state = %q, want %q", next, "someNextState")
	}
	if len(exec.singularCalls)+len(exec.batchCalls) != 0 {
		t.Errorf("executor called %d times for empty ops state; want 0",
			len(exec.singularCalls)+len(exec.batchCalls))
	}
}

// Executor failure must propagate and clean up conversation state.
func TestProcessGenericActionState_PropagatesExecutorError(t *testing.T) {
	wantErr := errors.New("saga create failed")
	exec := &recordingExecutor{batchErr: wantErr}
	p, ctx := newTestProcessor(t, exec)
	state := buildRewardsState(t)

	_, err := p.processGenericActionState(ctx, state)
	if !errors.Is(err, wantErr) {
		t.Fatalf("error = %v, want %v", err, wantErr)
	}
	if len(exec.batchCalls) != 1 {
		t.Errorf("ExecuteOperations called %d times, want 1", len(exec.batchCalls))
	}
}
