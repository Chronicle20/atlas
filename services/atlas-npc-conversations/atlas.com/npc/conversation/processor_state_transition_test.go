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

// testStateContainer is a minimal StateContainer implementation for tests
// that need FindState and States() (the latter for the MarshalJSON path).
type testStateContainer struct {
	start  string
	states []StateModel
}

func (t testStateContainer) StartState() string     { return t.start }
func (t testStateContainer) States() []StateModel   { return t.states }
func (t testStateContainer) FindState(id string) (StateModel, error) {
	for _, s := range t.states {
		if s.Id() == id {
			return s, nil
		}
	}
	return StateModel{}, errors.New("state not found")
}

// contextWritingExecutor simulates a local operation that mutates the
// conversation context in the registry (the pattern used by
// local:fetch_map_player_counts, local:generate_face_colors_for_onetime_lens,
// local:save_location, etc., all of which call setContextValue).
type contextWritingExecutor struct {
	tctx   context.Context
	writes map[string]string
}

func (c *contextWritingExecutor) ExecuteOperation(_ field.Model, characterId uint32, _ OperationModel) error {
	return c.mutate(characterId)
}

func (c *contextWritingExecutor) ExecuteOperations(_ field.Model, characterId uint32, _ []OperationModel) error {
	return c.mutate(characterId)
}

func (c *contextWritingExecutor) mutate(characterId uint32) error {
	ctx, err := GetRegistry().GetPreviousContext(c.tctx, characterId)
	if err != nil {
		return err
	}
	m := ctx.Context()
	for k, v := range c.writes {
		m[k] = v
	}
	GetRegistry().UpdateContext(c.tctx, characterId, ctx)
	return nil
}

// Regression for NPC 1052114 bug: the list selection for the Maple Island
// training centers rendered literal "{context.playerCount_910310000}" text
// because ProcessState was rebuilding the next state's context from its
// stale in-memory ctx.Context() and writing that back to Redis, wiping the
// playerCount_* keys the local:fetch_map_player_counts operation had just
// stored.
//
// This test: run a genericAction state whose operation mutates a context
// key via the registry (same path setContextValue uses). After the
// transition, the registry must still hold the mutation.
func TestProcessState_PreservesOperationContextMutationsAcrossTransition(t *testing.T) {
	mr := miniredis.RunT(t)
	rc := goredis.NewClient(&goredis.Options{Addr: mr.Addr()})
	InitRegistry(rc)

	l, _ := test.NewNullLogger()
	l.SetLevel(logrus.DebugLevel)

	var tm tenant.Model
	tctx := tenant.WithContext(context.Background(), tm)

	op, err := NewOperationBuilder().SetType("local:writeKey").Build()
	if err != nil {
		t.Fatalf("build op: %v", err)
	}
	outcome, err := NewOutcomeBuilder().SetNextState("B").Build()
	if err != nil {
		t.Fatalf("build outcome: %v", err)
	}
	ga, err := NewGenericActionBuilder().AddOperation(op).AddOutcome(outcome).Build()
	if err != nil {
		t.Fatalf("build genericAction: %v", err)
	}
	stateA, err := NewStateBuilder().SetId("A").SetGenericAction(ga).Build()
	if err != nil {
		t.Fatalf("build state A: %v", err)
	}

	container := testStateContainer{
		start:  "A",
		states: []StateModel{stateA},
	}

	exec := &contextWritingExecutor{
		tctx:   tctx,
		writes: map[string]string{"playerCount_910310000": "3"},
	}

	f := field.NewBuilder(world.Id(0), channel.Id(1), _map.Id(20000)).Build()

	ctx := NewConversationContextBuilder().
		SetField(f).
		SetCharacterId(7).
		SetNpcId(1052114).
		SetCurrentState("A").
		SetConversation(container).
		AddContextValue("preExisting", "value").
		Build()

	GetRegistry().SetContext(tctx, ctx.CharacterId(), ctx)

	p := &ProcessorImpl{
		l:        l,
		ctx:      tctx,
		t:        tm,
		executor: exec,
	}

	if _, err := p.ProcessState(ctx); err != nil {
		t.Fatalf("ProcessState: %v", err)
	}

	got, err := GetRegistry().GetPreviousContext(tctx, ctx.CharacterId())
	if err != nil {
		t.Fatalf("GetPreviousContext after transition: %v", err)
	}

	if got.CurrentState() != "B" {
		t.Errorf("CurrentState = %q, want %q", got.CurrentState(), "B")
	}
	if v, ok := got.Context()["playerCount_910310000"]; !ok || v != "3" {
		t.Errorf("playerCount_910310000 = %q (present=%v); want %q — operation-produced context mutation was overwritten by stale rebuild in ProcessState",
			v, ok, "3")
	}
	if v := got.Context()["preExisting"]; v != "value" {
		t.Errorf("preExisting = %q; want %q — pre-existing context dropped on transition", v, "value")
	}
}
