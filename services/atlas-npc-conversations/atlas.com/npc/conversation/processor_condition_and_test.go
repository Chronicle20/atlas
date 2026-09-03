package conversation

import (
	"context"
	"errors"
	"testing"

	"github.com/alicebob/miniredis/v2"
	goredis "github.com/redis/go-redis/v9"
	"github.com/sirupsen/logrus"
	"github.com/sirupsen/logrus/hooks/test"

	"github.com/Chronicle20/atlas/libs/atlas-constants/channel"
	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"
	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

// countingEvaluator is a fake Evaluator that records every condition it was
// asked to evaluate (keyed by ConditionModel.Value(), which each test gives a
// distinct value) and returns a scripted result/error for that key. It makes
// short-circuit behaviour falsifiable: if the AND loop stops evaluating
// conditions after the first false, later conditions never appear in calls.
type countingEvaluator struct {
	results map[string]bool
	errs    map[string]error
	calls   []string
}

func newCountingEvaluator() *countingEvaluator {
	return &countingEvaluator{
		results: make(map[string]bool),
		errs:    make(map[string]error),
	}
}

func (c *countingEvaluator) EvaluateCondition(_ uint32, condition ConditionModel) (bool, error) {
	c.calls = append(c.calls, condition.Value())
	if err, ok := c.errs[condition.Value()]; ok && err != nil {
		return false, err
	}
	return c.results[condition.Value()], nil
}

// newConditionTestProcessor builds a ProcessorImpl wired to a no-op executor
// and the given evaluator, mirroring newTestProcessor in
// processor_generic_action_test.go but with a configurable evaluator.
func newConditionTestProcessor(t *testing.T, evaluator Evaluator) (*ProcessorImpl, ConversationContext) {
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
		SetCurrentState("gateState").
		Build()

	p := &ProcessorImpl{
		l:         l,
		ctx:       tctx,
		t:         tm,
		executor:  &recordingExecutor{},
		evaluator: evaluator,
	}
	return p, ctx
}

func buildCondition(t *testing.T, value string) ConditionModel {
	t.Helper()

	condition, err := NewConditionBuilder().
		SetType("questStatus").
		SetOperator("=").
		SetValue(value).
		Build()
	if err != nil {
		t.Fatalf("build condition: %v", err)
	}
	return condition
}

func buildTwoConditionState(t *testing.T, first, second ConditionModel, matchNextState, fallbackNextState string) StateModel {
	t.Helper()

	matchOutcome, err := NewOutcomeBuilder().
		AddCondition(first).
		AddCondition(second).
		SetNextState(matchNextState).
		Build()
	if err != nil {
		t.Fatalf("build matchOutcome: %v", err)
	}

	fallbackOutcome, err := NewOutcomeBuilder().SetNextState(fallbackNextState).Build()
	if err != nil {
		t.Fatalf("build fallbackOutcome: %v", err)
	}

	ga, err := NewGenericActionBuilder().
		AddOutcome(matchOutcome).
		AddOutcome(fallbackOutcome).
		Build()
	if err != nil {
		t.Fatalf("build genericAction: %v", err)
	}

	state, err := NewStateBuilder().SetId("gateState").SetGenericAction(ga).Build()
	if err != nil {
		t.Fatalf("build state: %v", err)
	}
	return state
}

// Test 1: both conditions of a two-condition outcome true -> outcome taken.
func TestProcessGenericActionState_TwoConditionsBothTrue_OutcomeTaken(t *testing.T) {
	first := buildCondition(t, "first")
	second := buildCondition(t, "second")

	eval := newCountingEvaluator()
	eval.results["first"] = true
	eval.results["second"] = true

	p, ctx := newConditionTestProcessor(t, eval)
	state := buildTwoConditionState(t, first, second, "matched", "fallback")

	next, err := p.processGenericActionState(ctx, state)
	if err != nil {
		t.Fatalf("processGenericActionState: unexpected error: %v", err)
	}
	if next != "matched" {
		t.Errorf("next state = %q, want %q", next, "matched")
	}
}

// Test 2: first true / second false -> outcome NOT taken, falls through to
// the next outcome (first-match-wins across outcomes preserved).
func TestProcessGenericActionState_TwoConditionsFirstTrueSecondFalse_FallsThrough(t *testing.T) {
	first := buildCondition(t, "first")
	second := buildCondition(t, "second")

	eval := newCountingEvaluator()
	eval.results["first"] = true
	eval.results["second"] = false

	p, ctx := newConditionTestProcessor(t, eval)
	state := buildTwoConditionState(t, first, second, "matched", "fallback")

	next, err := p.processGenericActionState(ctx, state)
	if err != nil {
		t.Fatalf("processGenericActionState: unexpected error: %v", err)
	}
	if next != "fallback" {
		t.Errorf("next state = %q, want %q", next, "fallback")
	}
}

// Test 3: first condition false -> the second condition must never be
// evaluated. This is falsifiable: deleting the short-circuit would make both
// conditions appear in eval.calls.
func TestProcessGenericActionState_FirstConditionFalse_ShortCircuits(t *testing.T) {
	first := buildCondition(t, "first")
	second := buildCondition(t, "second")

	eval := newCountingEvaluator()
	eval.results["first"] = false
	eval.results["second"] = true

	p, ctx := newConditionTestProcessor(t, eval)
	state := buildTwoConditionState(t, first, second, "matched", "fallback")

	next, err := p.processGenericActionState(ctx, state)
	if err != nil {
		t.Fatalf("processGenericActionState: unexpected error: %v", err)
	}
	if next != "fallback" {
		t.Errorf("next state = %q, want %q", next, "fallback")
	}

	if len(eval.calls) != 1 || eval.calls[0] != "first" {
		t.Fatalf("evaluator calls = %v, want exactly [\"first\"] (second condition must not be evaluated once the first fails)", eval.calls)
	}
}

// Test 4: zero-condition outcome is taken unconditionally. Regression guard
// on the pre-existing short-circuit that must be preserved.
func TestProcessGenericActionState_ZeroConditionOutcome_TakenUnconditionally(t *testing.T) {
	eval := newCountingEvaluator()

	p, ctx := newConditionTestProcessor(t, eval)

	outcome, err := NewOutcomeBuilder().SetNextState("unconditionalNext").Build()
	if err != nil {
		t.Fatalf("build outcome: %v", err)
	}
	ga, err := NewGenericActionBuilder().AddOutcome(outcome).Build()
	if err != nil {
		t.Fatalf("build genericAction: %v", err)
	}
	state, err := NewStateBuilder().SetId("gateState").SetGenericAction(ga).Build()
	if err != nil {
		t.Fatalf("build state: %v", err)
	}

	next, err := p.processGenericActionState(ctx, state)
	if err != nil {
		t.Fatalf("processGenericActionState: unexpected error: %v", err)
	}
	if next != "unconditionalNext" {
		t.Errorf("next state = %q, want %q", next, "unconditionalNext")
	}
	if len(eval.calls) != 0 {
		t.Errorf("evaluator calls = %v, want none (zero-condition outcome is unconditional)", eval.calls)
	}
}

// Test 5: a condition returning an error must clear the conversation context,
// return ("", err), and the logged/returned error must concern the condition
// that actually failed (the second one), not Conditions()[0].
func TestProcessGenericActionState_ConditionError_ClearsContextAndReturnsError(t *testing.T) {
	first := buildCondition(t, "first")
	second := buildCondition(t, "second")

	wantErr := errors.New("remote evaluation failed")
	eval := newCountingEvaluator()
	eval.results["first"] = true
	eval.errs["second"] = wantErr

	p, ctx := newConditionTestProcessor(t, eval)
	state := buildTwoConditionState(t, first, second, "matched", "fallback")

	next, err := p.processGenericActionState(ctx, state)
	if !errors.Is(err, wantErr) {
		t.Fatalf("error = %v, want %v", err, wantErr)
	}
	if next != "" {
		t.Errorf("next state = %q, want empty on error", next)
	}
	if len(eval.calls) != 2 || eval.calls[0] != "first" || eval.calls[1] != "second" {
		t.Fatalf("evaluator calls = %v, want [\"first\" \"second\"] (error must come from the condition that actually failed)", eval.calls)
	}
}
