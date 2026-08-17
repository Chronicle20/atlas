package saga

// world_transfer_compensation_test.go — the FR-4.8 proof (task-227 Task 14).
//
// The invariant under test is WHOLENESS: a world transfer that fails partway
// must leave the character entirely in the SOURCE world, with every recoverable
// severance undone and the backing pending-change record resolved to
// REJECTED/saga_failed exactly once. Never in two worlds, never in none, never
// stripped of a guild they never actually left.
//
// The failure is real, not stubbed: the saga is walked forward through the
// production Processor writes (MarkEarliestPendingStepCompleted /
// MarkEarliestPendingStep(Failed)), each of which runs
// ValidateStateConsistency, and the compensation is then entered through the
// production dispatch point CompensateFailedStep — the same call
// Processor.Step makes when it observes s.Failing(). Only the downstream
// processors are mocked, which is what makes the dispatched arguments
// assertable at all.

import (
	buddymock "atlas-saga-orchestrator/buddylist/mock"
	guildmock "atlas-saga-orchestrator/guild/mock"
	pcmock "atlas-saga-orchestrator/pending_change/mock"
	"context"
	"fmt"
	"strings"
	"sync"
	"testing"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	logtest "github.com/sirupsen/logrus/hooks/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

// worldTransferFixture is the shape of one transfer under test: a character
// with a guild rank worth preserving, a party, and two buddies.
type worldTransferFixture struct {
	CharacterId        uint32
	SourceWorldId      world.Id
	DestinationWorldId world.Id
	GuildId            uint32
	GuildTitle         byte
	PartyId            uint32
	BuddyIds           []uint32
	PendingChangeId    uuid.UUID
}

func defaultWorldTransferFixture() worldTransferFixture {
	return worldTransferFixture{
		CharacterId:        1,
		SourceWorldId:      world.Id(0),
		DestinationWorldId: world.Id(1),
		GuildId:            5,
		GuildTitle:         3,
		PartyId:            9,
		BuddyIds:           []uint32{2, 3},
		PendingChangeId:    uuid.MustParse("cccccccc-0000-0000-0000-00000000002a"),
	}
}

// The five step ids, in the fixed order design §3.11 mandates.
const (
	wtStepValidate = "validate_world_transfer"
	wtStepGuild    = "leave_guild_for_transfer"
	wtStepParty    = "leave_party_for_transfer"
	wtStepBuddies  = "sever_buddies_for_transfer"
	wtStepWorld    = "change_character_world"
)

// worldTransferTestEnv records every compensating dispatch, in order, so the
// reverse-order requirement is assertable rather than merely asserted about.
type worldTransferTestEnv struct {
	t           *testing.T
	ctx         context.Context
	logger      *logrus.Logger
	hook        *logtest.Hook
	compensator Compensator

	mu    sync.Mutex
	calls []string
}

func newWorldTransferTestEnv(t *testing.T) *worldTransferTestEnv {
	t.Helper()
	logger, hook := logtest.NewNullLogger()
	logger.SetLevel(logrus.DebugLevel)

	te, err := tenant.Create(uuid.New(), "GMS", 83, 1)
	require.NoError(t, err)
	ctx := tenant.WithContext(context.Background(), te)

	env := &worldTransferTestEnv{t: t, ctx: ctx, logger: logger, hook: hook}

	guildP := &guildmock.ProcessorMock{
		RequestRejoinFunc: func(_ uuid.UUID, characterId uint32, guildId uint32, title byte) error {
			env.record(fmt.Sprintf("guild-rejoin:%d:%d:%d", characterId, guildId, title))
			return nil
		},
		RequestLeaveFunc: func(_ uuid.UUID, characterId uint32, guildId uint32, force bool) error {
			env.record(fmt.Sprintf("guild-leave:%d:%d:%t", characterId, guildId, force))
			return nil
		},
	}
	buddyP := &buddymock.ProcessorMock{
		RestoreAndEmitFunc: func(_ uuid.UUID, characterId uint32, worldId world.Id, targetId uint32) error {
			env.record(fmt.Sprintf("buddy-restore:%d:%d", characterId, targetId))
			return nil
		},
		RequestDeleteAndEmitFunc: func(_ uuid.UUID, characterId uint32, _ world.Id, targetId uint32) error {
			env.record(fmt.Sprintf("buddy-delete:%d:%d", characterId, targetId))
			return nil
		},
	}
	pcP := &pcmock.ProcessorMock{
		ChangeWorldFunc: func(_ uuid.UUID, characterId uint32, newWorldId world.Id) error {
			env.record(fmt.Sprintf("world-change:%d:%d", characterId, newWorldId))
			return nil
		},
		ResolveFunc: func(characterId uint32, id uuid.UUID, status string, reason string) error {
			env.record(fmt.Sprintf("resolve:%d:%s:%s:%s", characterId, id, status, reason))
			return nil
		},
	}

	env.compensator = NewCompensator(logger, ctx).
		WithGuildProcessor(guildP).
		WithBuddyListProcessor(buddyP).
		WithPendingChangeProcessor(pcP)

	return env
}

func (e *worldTransferTestEnv) record(s string) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.calls = append(e.calls, s)
}

func (e *worldTransferTestEnv) recorded() []string {
	e.mu.Lock()
	defer e.mu.Unlock()
	out := make([]string, len(e.calls))
	copy(out, e.calls)
	return out
}

// countPrefix reports how many recorded dispatches start with prefix.
func (e *worldTransferTestEnv) countPrefix(prefix string) int {
	n := 0
	for _, c := range e.recorded() {
		if strings.HasPrefix(c, prefix) {
			n++
		}
	}
	return n
}

// indexOf returns the position of an exact dispatch, or -1.
func (e *worldTransferTestEnv) indexOf(call string) int {
	for i, c := range e.recorded() {
		if c == call {
			return i
		}
	}
	return -1
}

// buildWorldTransferSaga assembles the five steps in their fixed order, all
// Pending. Statuses are then driven forward through the production Processor
// writes so no test ever hand-forges a Completed/Failed saga.
func buildWorldTransferSaga(t *testing.T, tx uuid.UUID, fx worldTransferFixture) Saga {
	t.Helper()
	s, err := NewBuilder().
		SetTransactionId(tx).
		SetSagaType(WorldTransfer).
		SetInitiatedBy("world-transfer-compensation-test").
		AddStep(wtStepValidate, Pending, ValidateWorldTransfer, ValidateWorldTransferPayload{
			CharacterId:        fx.CharacterId,
			SourceWorldId:      fx.SourceWorldId,
			DestinationWorldId: fx.DestinationWorldId,
			PendingChangeId:    fx.PendingChangeId,
		}).
		AddStep(wtStepGuild, Pending, LeaveGuildForTransfer, LeaveGuildForTransferPayload{
			CharacterId: fx.CharacterId,
			WorldId:     fx.SourceWorldId,
			GuildId:     fx.GuildId,
			Title:       fx.GuildTitle,
		}).
		AddStep(wtStepParty, Pending, LeavePartyForTransfer, LeavePartyForTransferPayload{
			CharacterId: fx.CharacterId,
			WorldId:     fx.SourceWorldId,
			PartyId:     fx.PartyId,
		}).
		AddStep(wtStepBuddies, Pending, SeverBuddiesForTransfer, SeverBuddiesForTransferPayload{
			CharacterId: fx.CharacterId,
			WorldId:     fx.SourceWorldId,
			BuddyIds:    fx.BuddyIds,
		}).
		AddStep(wtStepWorld, Pending, ChangeCharacterWorld, ChangeCharacterWorldPayload{
			CharacterId:        fx.CharacterId,
			SourceWorldId:      fx.SourceWorldId,
			DestinationWorldId: fx.DestinationWorldId,
			PendingChangeId:    fx.PendingChangeId,
		}).
		Build()
	require.NoError(t, err)
	return s
}

// runToFailureAt walks the saga forward through the REAL processor writes,
// completing every step ahead of failStepId and then failing that one. Each
// write runs Saga.ValidateStateConsistency, so an ordering the orchestrator
// would reject is rejected here too — the failure is produced, not asserted
// into existence. It returns the saga as the compensator will read it.
func (e *worldTransferTestEnv) runToFailureAt(fx worldTransferFixture, failStepId string) Saga {
	e.t.Helper()
	tx := uuid.New()
	s := buildWorldTransferSaga(e.t, tx, fx)
	require.NoError(e.t, GetCache().Put(e.ctx, s))
	e.t.Cleanup(func() { GetCache().Remove(e.ctx, tx) })

	p := NewProcessor(e.logger, e.ctx)
	for {
		cur, err := p.GetById(tx)
		require.NoError(e.t, err)
		st, ok := cur.GetCurrentStep()
		require.True(e.t, ok, "ran out of pending steps before reaching [%s]", failStepId)
		if st.StepId() == failStepId {
			require.NoError(e.t, p.MarkEarliestPendingStep(tx, Failed))
			break
		}
		require.NoError(e.t, p.MarkEarliestPendingStepCompleted(tx))
	}

	failed, err := p.GetById(tx)
	require.NoError(e.t, err)
	require.True(e.t, failed.Failing(), "the saga must genuinely be in a failing state")

	// The lifecycle transition Processor.Step's sync-error path takes before
	// handing off to the compensator. Without it compensateWorldTransfer's
	// Compensating -> Failed guard reads as "already terminal" and skips the
	// emission half.
	require.True(e.t, GetCache().TryTransition(e.ctx, tx, SagaLifecyclePending, SagaLifecycleCompensating))
	return failed
}

// TestChangeCharacterWorldFailureCompensatesEverySeverance is the FR-4.8 / PRD
// acceptance proof. A failure at the LAST step must leave the character wholly
// in the source world — which, because change_character_world is last and its
// own step failed, means no world move ever committed — with every recoverable
// severance undone and the record REJECTED exactly once so the coupon refunds.
func TestChangeCharacterWorldFailureCompensatesEverySeverance(t *testing.T) {
	env := newWorldTransferTestEnv(t)
	fx := defaultWorldTransferFixture()

	s := env.runToFailureAt(fx, wtStepWorld)

	failedStep, ok := s.StepAt(s.FindFailedStepIndex())
	require.True(t, ok)
	require.Equal(t, wtStepWorld, failedStep.StepId())

	// The production dispatch point, not the arm directly.
	require.NoError(t, env.compensator.CompensateFailedStep(s))

	calls := env.recorded()

	// The character never moved: change_character_world FAILED, so it is not
	// Completed and has no inverse to run. A world-change dispatch here would
	// mean the compensator moved a character the saga never moved.
	assert.Equal(t, 0, env.countPrefix("world-change:"),
		"no world move committed, so none may be reversed; got %v", calls)

	// Both buddies, both directions — 2N restores for N buddies, mirroring the
	// 2N severances the forward step emitted.
	for _, b := range fx.BuddyIds {
		assert.NotEqual(t, -1, env.indexOf(fmt.Sprintf("buddy-restore:%d:%d", fx.CharacterId, b)),
			"buddy %d not restored on the character's own list", b)
		assert.NotEqual(t, -1, env.indexOf(fmt.Sprintf("buddy-restore:%d:%d", b, fx.CharacterId)),
			"buddy %d not restored in the reverse direction", b)
	}
	assert.Equal(t, 2*len(fx.BuddyIds), env.countPrefix("buddy-restore:"))

	// The guild membership comes back at the EXACT prior title. A rejoin at
	// the default rookie rank silently demotes an officer.
	guildCall := fmt.Sprintf("guild-rejoin:%d:%d:%d", fx.CharacterId, fx.GuildId, fx.GuildTitle)
	guildIdx := env.indexOf(guildCall)
	assert.NotEqual(t, -1, guildIdx, "guild membership was not restored at the prior title; got %v", calls)
	assert.Equal(t, 1, env.countPrefix("guild-rejoin:"))

	// Party membership is deliberately NOT restorable (design §3.11).
	assert.Equal(t, 0, env.countPrefix("party-"), "party must have no compensating dispatch")

	// REVERSE step order: the buddy step sits after the guild step in the
	// saga, so its inverse must run BEFORE the guild's.
	buddyIdx := env.indexOf(fmt.Sprintf("buddy-restore:%d:%d", fx.CharacterId, fx.BuddyIds[0]))
	require.NotEqual(t, -1, buddyIdx)
	assert.Less(t, buddyIdx, guildIdx, "compensations must run in reverse step order; got %v", calls)

	// The record resolves to REJECTED/saga_failed exactly once — that
	// transition is what drives the refund, and a second one would either
	// double-refund or be rejected as already-terminal.
	resolveCall := fmt.Sprintf("resolve:%d:%s:%s:%s", fx.CharacterId, fx.PendingChangeId, "REJECTED", "saga_failed")
	assert.Equal(t, 1, env.countPrefix("resolve:"), "exactly one record resolution; got %v", calls)
	assert.NotEqual(t, -1, env.indexOf(resolveCall),
		"record must resolve to REJECTED/saga_failed; got %v", calls)
}

// TestEarlyFailureCompensatesOnlyCompletedSteps: a failure at the FIRST
// severance must not compensate steps that never ran. An over-eager
// compensator re-adds a guild membership the character never left, or restores
// a buddy that was never severed — both of which are corruption, not recovery.
func TestEarlyFailureCompensatesOnlyCompletedSteps(t *testing.T) {
	env := newWorldTransferTestEnv(t)
	fx := defaultWorldTransferFixture()

	s := env.runToFailureAt(fx, wtStepGuild)
	require.NoError(t, env.compensator.CompensateFailedStep(s))

	calls := env.recorded()
	assert.Equal(t, 0, env.countPrefix("buddy-restore:"),
		"compensator restored buddies that were never severed; got %v", calls)
	assert.Equal(t, 0, env.countPrefix("guild-rejoin:"),
		"compensator re-added a guild the character never left; got %v", calls)
	assert.Equal(t, 0, env.countPrefix("world-change:"),
		"character world changed despite an early failure; got %v", calls)

	// The record is still resolved — the saga is dead either way, and a record
	// left PENDING means an unrefunded coupon.
	assert.Equal(t, 1, env.countPrefix("resolve:"))
	assert.NotEqual(t, -1, env.indexOf(fmt.Sprintf("resolve:%d:%s:REJECTED:saga_failed", fx.CharacterId, fx.PendingChangeId)))
}

// TestWorldTransferRollbacksRestoreTheSourceWorldFirst pins the ordering
// requirement on the one shape that actually needs it: every step Completed
// (the shape the timeout backstop sees when a saga finished its steps but
// never went terminal). The world must come back to SourceWorldId, and it must
// come back BEFORE the guild and buddy rows are restored — those services key
// their rows off the character's world.
func TestWorldTransferRollbacksRestoreTheSourceWorldFirst(t *testing.T) {
	env := newWorldTransferTestEnv(t)
	fx := defaultWorldTransferFixture()

	tx := uuid.New()
	s := buildWorldTransferSaga(t, tx, fx)
	require.NoError(t, GetCache().Put(env.ctx, s))
	t.Cleanup(func() { GetCache().Remove(env.ctx, tx) })

	p := NewProcessor(env.logger, env.ctx)
	for i := 0; i < 5; i++ {
		require.NoError(t, p.MarkEarliestPendingStepCompleted(tx))
	}
	completed, err := p.GetById(tx)
	require.NoError(t, err)
	for _, st := range completed.Steps() {
		require.Equal(t, Completed, st.Status(), "step %s", st.StepId())
	}

	env.compensator.(*CompensatorImpl).DispatchWorldTransferRollbacks(completed)

	calls := env.recorded()

	worldIdx := env.indexOf(fmt.Sprintf("world-change:%d:%d", fx.CharacterId, fx.SourceWorldId))
	require.NotEqual(t, -1, worldIdx, "world was not restored to the SOURCE world; got %v", calls)
	assert.Equal(t, 1, env.countPrefix("world-change:"))

	buddyIdx := env.indexOf(fmt.Sprintf("buddy-restore:%d:%d", fx.CharacterId, fx.BuddyIds[0]))
	guildIdx := env.indexOf(fmt.Sprintf("guild-rejoin:%d:%d:%d", fx.CharacterId, fx.GuildId, fx.GuildTitle))
	require.NotEqual(t, -1, buddyIdx)
	require.NotEqual(t, -1, guildIdx)

	assert.Less(t, worldIdx, buddyIdx, "world restore must precede buddy restore; got %v", calls)
	assert.Less(t, buddyIdx, guildIdx, "buddy restore must precede guild rejoin; got %v", calls)

	// The party step completed and is deliberately not compensated. It must be
	// LOGGED as a deliberate skip, so an operator reading the compensation
	// trail can tell "not restorable" from "nobody wrote the case".
	found := false
	for _, e := range env.hook.AllEntries() {
		if e.Data["step_id"] == wtStepParty && strings.Contains(e.Message, "not restorable") {
			found = true
			break
		}
	}
	assert.True(t, found, "the deliberate party no-op must be logged, not silent")
}

// TestWorldTransferIsClassifiedForTimeout: timer.go's own doc comment records
// the defect this pins — a saga type routed to a bespoke compensator by
// CompensateFailedStep but missing from the timeout lists rolls back NOTHING
// when it times out. For world_transfer that means a character stranded
// guildless, partyless and buddyless.
func TestWorldTransferIsClassifiedForTimeout(t *testing.T) {
	var inAll, inReverse bool
	for _, ty := range allSagaTypes {
		if ty == WorldTransfer {
			inAll = true
		}
	}
	for _, ty := range reverseWalkSagaTypes {
		if ty == WorldTransfer {
			inReverse = true
		}
	}
	assert.True(t, inAll, "WorldTransfer must appear in allSagaTypes")
	assert.True(t, inReverse, "WorldTransfer must appear in reverseWalkSagaTypes")
}

// And the switch must actually have an arm — a type in the list with no case
// falls to default and returns false, dispatching no inverse.
func TestWorldTransferTimeoutDispatchesRollbacks(t *testing.T) {
	logger, _ := logtest.NewNullLogger()
	env := newWorldTransferTestEnv(t)
	fx := defaultWorldTransferFixture()

	s := buildWorldTransferSaga(t, uuid.New(), fx)
	if !dispatchTimeoutRollbacks(logger, env.ctx, s) {
		t.Fatal("dispatchTimeoutRollbacks returned false: world_transfer has no timeout arm")
	}
}
