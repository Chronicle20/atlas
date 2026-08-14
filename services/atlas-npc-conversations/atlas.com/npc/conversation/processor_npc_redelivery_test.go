package conversation

import (
	"errors"
	"testing"

	"github.com/google/uuid"

	"github.com/Chronicle20/atlas/libs/atlas-constants/channel"
	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"
	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
)

// testNpcConversationProvider serves one conversation for any npc id.
type testNpcConversationProvider struct {
	conv NpcConversation
}

func (p testNpcConversationProvider) ByNpcIdProvider(npcId uint32) func() (NpcConversation, error) {
	return func() (NpcConversation, error) { return p.conv, nil }
}

// newNpcTestProcessor mirrors newItemTestProcessor but wires the NPC
// conversation provider that Start requires.
func newNpcTestProcessor(t *testing.T, npcId uint32, sm StateContainer) *ProcessorImpl {
	t.Helper()
	tsc, ok := sm.(testStateContainer)
	if !ok {
		t.Fatalf("expected testStateContainer, got %T", sm)
	}
	p, _ := newItemTestProcessor(t)
	p.npcConversationProvider = testNpcConversationProvider{
		conv: testNpcConversation{npcId: npcId, start: tsc.start, states: tsc.states},
	}
	return p
}

// A redelivered start_npc_conversation command carrying the SAME transaction id
// as the live context must be treated as already-succeeded, not as a conflict.
// Kafka is at-least-once; emitting START_ERROR here would fail a saga step that
// already succeeded and drive a compensation that force-ends a conversation the
// player is legitimately in. This mirrors the StartItem guard.
func TestStartRedeliveryOfSameTransactionIsNotAConflict(t *testing.T) {
	var npcId uint32 = 9010000
	sm := parkingItemStateMachine(t, "wait")
	p := newNpcTestProcessor(t, npcId, sm)

	f := field.NewBuilder(world.Id(0), channel.Id(1), _map.Id(100000000)).Build()
	var characterId uint32 = 42
	var accountId uint32 = 1
	txn := uuid.New()

	if err := p.Start(f, npcId, characterId, accountId, txn); err != nil {
		t.Fatalf("first start: %v", err)
	}
	err := p.Start(f, npcId, characterId, accountId, txn)
	if !errors.Is(err, ErrAlreadyStartedByThisTransaction) {
		t.Fatalf("redelivery: got %v, want ErrAlreadyStartedByThisTransaction", err)
	}
}

// A DIFFERENT transaction against a live context is a genuine conflict.
func TestStartDifferentTransactionIsAConflict(t *testing.T) {
	var npcId uint32 = 9010000
	sm := parkingItemStateMachine(t, "wait")
	p := newNpcTestProcessor(t, npcId, sm)

	f := field.NewBuilder(world.Id(0), channel.Id(1), _map.Id(100000000)).Build()
	var characterId uint32 = 42
	var accountId uint32 = 1

	if err := p.Start(f, npcId, characterId, accountId, uuid.New()); err != nil {
		t.Fatalf("first start: %v", err)
	}
	err := p.Start(f, npcId, characterId, accountId, uuid.New())
	if !errors.Is(err, ErrConversationInProgress) {
		t.Fatalf("conflict: got %v, want ErrConversationInProgress", err)
	}
}

// The ordinary NPC-talk path (uuid.Nil, no saga awaiting) must keep its previous
// behaviour: a second start against a live context is a conflict, and uuid.Nil
// must never match a uuid.Nil-stamped context as a "redelivery".
func TestStartNilTransactionNeverMatchesAsRedelivery(t *testing.T) {
	var npcId uint32 = 9010000
	sm := parkingItemStateMachine(t, "wait")
	p := newNpcTestProcessor(t, npcId, sm)

	f := field.NewBuilder(world.Id(0), channel.Id(1), _map.Id(100000000)).Build()
	var characterId uint32 = 42
	var accountId uint32 = 1

	if err := p.Start(f, npcId, characterId, accountId, uuid.Nil); err != nil {
		t.Fatalf("first start: %v", err)
	}
	err := p.Start(f, npcId, characterId, accountId, uuid.Nil)
	if !errors.Is(err, ErrConversationInProgress) {
		t.Fatalf("nil-transaction second start: got %v, want ErrConversationInProgress", err)
	}
}
