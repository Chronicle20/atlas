package conversation

import (
	"context"
	"errors"
	"testing"

	"github.com/alicebob/miniredis/v2"
	"github.com/google/uuid"
	goredis "github.com/redis/go-redis/v9"
	"github.com/sirupsen/logrus"
	"github.com/sirupsen/logrus/hooks/test"

	"github.com/Chronicle20/atlas/libs/atlas-constants/channel"
	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"
	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

// newItemTestProcessor wires a miniredis-backed registry and a ProcessorImpl,
// mirroring the setup in processor_state_transition_test.go and
// processor_rps_test.go.
func newItemTestProcessor(t *testing.T) (*ProcessorImpl, context.Context) {
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

// parkingItemStateMachine builds a single-state StateContainer whose
// genericAction outcome loops back to the same state (no operations, no
// conditions). ProcessState parks on a self-transition without clearing the
// registry, which is what lets the redelivery guard observe the live context
// on the second StartItem call.
func parkingItemStateMachine(t *testing.T, stateId string) StateContainer {
	t.Helper()
	outcome, err := NewOutcomeBuilder().SetNextState(stateId).Build()
	if err != nil {
		t.Fatalf("build outcome: %v", err)
	}
	ga, err := NewGenericActionBuilder().AddOutcome(outcome).Build()
	if err != nil {
		t.Fatalf("build genericAction: %v", err)
	}
	state, err := NewStateBuilder().SetId(stateId).SetGenericAction(ga).Build()
	if err != nil {
		t.Fatalf("build state: %v", err)
	}
	return testStateContainer{start: stateId, states: []StateModel{state}}
}

// A redelivered start command carrying the SAME transaction id as the live
// context must be treated as already-succeeded, not as a conflict. Kafka is
// at-least-once, and emitting START_ERROR here would fail a saga that had
// already opened its dialogue.
func TestStartItemRedeliveryOfSameTransactionIsNotAConflict(t *testing.T) {
	p, _ := newItemTestProcessor(t)

	f := field.NewBuilder(world.Id(0), channel.Id(1), _map.Id(100000000)).Build()
	sm := parkingItemStateMachine(t, "wait")

	var characterId uint32 = 42
	var accountId uint32 = 1
	txn := uuid.New()

	if err := p.StartItem(f, 2430013, 9010000, characterId, accountId, "item_2430013", txn, sm); err != nil {
		t.Fatalf("first start: %v", err)
	}
	err := p.StartItem(f, 2430013, 9010000, characterId, accountId, "item_2430013", txn, sm)
	if !errors.Is(err, ErrAlreadyStartedByThisTransaction) {
		t.Fatalf("redelivery: got %v, want ErrAlreadyStartedByThisTransaction", err)
	}
}

// A DIFFERENT transaction against a live context is a genuine conflict.
func TestStartItemDifferentTransactionIsAConflict(t *testing.T) {
	p, _ := newItemTestProcessor(t)

	f := field.NewBuilder(world.Id(0), channel.Id(1), _map.Id(100000000)).Build()
	sm := parkingItemStateMachine(t, "wait")
	sm2 := parkingItemStateMachine(t, "wait2")

	var characterId uint32 = 42
	var accountId uint32 = 1

	if err := p.StartItem(f, 2430013, 9010000, characterId, accountId, "item_2430013", uuid.New(), sm); err != nil {
		t.Fatalf("first start: %v", err)
	}
	err := p.StartItem(f, 2430008, 2084002, characterId, accountId, "compassUse", uuid.New(), sm2)
	if !errors.Is(err, ErrConversationInProgress) {
		t.Fatalf("conflict: got %v, want ErrConversationInProgress", err)
	}
}
