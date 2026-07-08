package conversation

import (
	sharedsaga "atlas-npc-conversations/saga"
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

// capturingSagaProcessor is a test double for sharedsaga.Processor that
// records the Saga passed to Create and returns a canned error (nil for success).
type capturingSagaProcessor struct {
	created []sharedsaga.Saga
	err     error
}

func (c *capturingSagaProcessor) Create(s sharedsaga.Saga) error {
	c.created = append(c.created, s)
	return c.err
}

// newRPSTestProcessor wires a miniredis-backed registry and a ProcessorImpl,
// mirroring the setup in processor_state_transition_test.go.
func newRPSTestProcessor(t *testing.T) (*ProcessorImpl, context.Context) {
	t.Helper()
	mr := miniredis.RunT(t)
	rc := goredis.NewClient(&goredis.Options{Addr: mr.Addr()})
	InitRegistry(rc)

	l, _ := test.NewNullLogger()
	l.SetLevel(logrus.DebugLevel)

	var tm tenant.Model
	tctx := tenant.WithContext(context.Background(), tm)

	p := &ProcessorImpl{
		l:   l,
		ctx: tctx,
		t:   tm,
	}
	return p, tctx
}

func buildRPSConversationContext(t *testing.T, stateId string, container StateContainer) ConversationContext {
	t.Helper()
	f := field.NewBuilder(world.Id(0), channel.Id(1), _map.Id(100000000)).Build()
	return NewConversationContextBuilder().
		SetField(f).
		SetCharacterId(42).
		SetNpcId(9000019).
		SetCurrentState(stateId).
		SetConversation(container).
		Build()
}

// TestProcessRPSActionState_BuildsEntrySagaAndParks verifies that
// processRPSActionState builds the two-step entry saga
// [AwardMesos(-entryCostMeso), StartRPSGame], sets pendingSagaId, stores the
// rpsAction_failureState context flag, and parks on the same state (design
// D3 — the entry saga).
func TestProcessRPSActionState_BuildsEntrySagaAndParks(t *testing.T) {
	p, tctx := newRPSTestProcessor(t)

	mock := &capturingSagaProcessor{}
	SetSagaProcessorFactory(func(l logrus.FieldLogger, ctx context.Context) sharedsaga.Processor {
		return mock
	})
	defer SetSagaProcessorFactory(nil)

	rpsAction, err := NewRPSActionBuilder().
		SetNpcId(9000019).
		SetEntryCostMeso(1000).
		SetFailureState("noMeso").
		Build()
	if err != nil {
		t.Fatalf("build rpsAction: %v", err)
	}

	state, err := NewStateBuilder().SetId("playRPS").SetRPSAction(rpsAction).Build()
	if err != nil {
		t.Fatalf("build state: %v", err)
	}

	container := testStateContainer{start: "playRPS", states: []StateModel{state}}
	ctx := buildRPSConversationContext(t, "playRPS", container)

	nextStateId, err := p.processRPSActionState(ctx, state)
	if err != nil {
		t.Fatalf("processRPSActionState returned error: %v", err)
	}

	// Parks on the same state, waiting for the saga-status consumer (Task 24).
	if nextStateId != "playRPS" {
		t.Errorf("nextStateId = %q, want %q (park)", nextStateId, "playRPS")
	}

	if len(mock.created) != 1 {
		t.Fatalf("saga Create called %d times, want 1", len(mock.created))
	}
	s := mock.created[0]

	if len(s.Steps) != 2 {
		t.Fatalf("saga has %d steps, want 2", len(s.Steps))
	}

	// Step 1: AwardMesos with a NEGATIVE amount.
	step1 := s.Steps[0]
	if step1.Action != sharedsaga.AwardMesos {
		t.Errorf("step 1 action = %q, want %q", step1.Action, sharedsaga.AwardMesos)
	}
	mesoPayload, ok := step1.Payload.(sharedsaga.AwardMesosPayload)
	if !ok {
		t.Fatalf("step 1 payload type = %T, want AwardMesosPayload", step1.Payload)
	}
	if mesoPayload.Amount != -int32(1000) {
		t.Errorf("AwardMesos.Amount = %d, want %d", mesoPayload.Amount, -int32(1000))
	}
	if mesoPayload.CharacterId != 42 {
		t.Errorf("AwardMesos.CharacterId = %d, want %d", mesoPayload.CharacterId, 42)
	}
	if mesoPayload.ActorType != "NPC" {
		t.Errorf("AwardMesos.ActorType = %q, want %q", mesoPayload.ActorType, "NPC")
	}
	if mesoPayload.ShowEffect != false {
		t.Errorf("AwardMesos.ShowEffect = %v, want false", mesoPayload.ShowEffect)
	}

	// Step 2: StartRPSGame carrying the npcId.
	step2 := s.Steps[1]
	if step2.Action != sharedsaga.StartRPSGame {
		t.Errorf("step 2 action = %q, want %q", step2.Action, sharedsaga.StartRPSGame)
	}
	startPayload, ok := step2.Payload.(sharedsaga.StartRPSGamePayload)
	if !ok {
		t.Fatalf("step 2 payload type = %T, want StartRPSGamePayload", step2.Payload)
	}
	if startPayload.NpcId != 9000019 {
		t.Errorf("StartRPSGame.NpcId = %d, want %d", startPayload.NpcId, 9000019)
	}
	if startPayload.CharacterId != 42 {
		t.Errorf("StartRPSGame.CharacterId = %d, want %d", startPayload.CharacterId, 42)
	}

	// pendingSagaId set + rpsAction_failureState stored in the registry context.
	stored, err := GetRegistry().GetPreviousContext(tctx, 42)
	if err != nil {
		t.Fatalf("GetPreviousContext: %v", err)
	}
	if stored.PendingSagaId() == nil {
		t.Fatalf("PendingSagaId() = nil, want set")
	}
	if got := stored.Context()["rpsAction_failureState"]; got != "noMeso" {
		t.Errorf("rpsAction_failureState = %q, want %q", got, "noMeso")
	}
}

// TestProcessRPSActionState_SagaCreateFailureRoutesToFailureState verifies
// that when saga creation fails outright (e.g. orchestrator unreachable),
// processRPSActionState routes directly to the state's failureState rather
// than parking (mirrors processGachaponActionState's error branch).
func TestProcessRPSActionState_SagaCreateFailureRoutesToFailureState(t *testing.T) {
	p, _ := newRPSTestProcessor(t)

	mock := &capturingSagaProcessor{err: errors.New("orchestrator unreachable")}
	SetSagaProcessorFactory(func(l logrus.FieldLogger, ctx context.Context) sharedsaga.Processor {
		return mock
	})
	defer SetSagaProcessorFactory(nil)

	rpsAction, err := NewRPSActionBuilder().
		SetNpcId(9000019).
		SetEntryCostMeso(1000).
		SetFailureState("noMeso").
		Build()
	if err != nil {
		t.Fatalf("build rpsAction: %v", err)
	}

	state, err := NewStateBuilder().SetId("playRPS").SetRPSAction(rpsAction).Build()
	if err != nil {
		t.Fatalf("build state: %v", err)
	}

	container := testStateContainer{start: "playRPS", states: []StateModel{state}}
	ctx := buildRPSConversationContext(t, "playRPS", container)

	nextStateId, err := p.processRPSActionState(ctx, state)
	if err != nil {
		t.Fatalf("processRPSActionState returned error: %v", err)
	}
	if nextStateId != "noMeso" {
		t.Errorf("nextStateId = %q, want %q (failureState)", nextStateId, "noMeso")
	}
}
