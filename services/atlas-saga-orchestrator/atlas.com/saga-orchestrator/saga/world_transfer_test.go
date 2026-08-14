package saga

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	logtest "github.com/sirupsen/logrus/hooks/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

func worldTransferTestCtx(t *testing.T) context.Context {
	t.Helper()
	tm, err := tenant.Create(uuid.New(), "GMS", 83, 1)
	require.NoError(t, err)
	return tenant.WithContext(context.Background(), tm)
}

// TestWorldTransferActionsAllResolveToHandlers guards against the class of
// bug that leaves a saga wedged indistinguishably from a slow downstream: an
// action with no case in GetHandler's switch. See task-227 Task 13.
func TestWorldTransferActionsAllResolveToHandlers(t *testing.T) {
	h := &HandlerImpl{}
	for _, a := range []Action{
		ValidateWorldTransfer, LeaveGuildForTransfer, LeavePartyForTransfer,
		SeverBuddiesForTransfer, ChangeCharacterWorld,
	} {
		if _, ok := h.GetHandler(a); !ok {
			t.Fatalf("action %s has no handler", a)
		}
	}
}

// TestWorldTransferStepsUnmarshalToConcretePayloads asserts payload unmarshal
// produces the concrete type, not map[string]interface{} — every handler
// type-asserts and returns "invalid payload" otherwise.
func TestWorldTransferStepsUnmarshalToConcretePayloads(t *testing.T) {
	raw := `{"action":"change_character_world","payload":{"characterId":1,"sourceWorldId":0,"destinationWorldId":1,"pendingChangeId":"` + uuid.New().String() + `"}}`
	var st Step[any]
	if err := json.Unmarshal([]byte(raw), &st); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if _, ok := st.Payload().(ChangeCharacterWorldPayload); !ok {
		t.Fatalf("payload type = %T, want ChangeCharacterWorldPayload", st.Payload())
	}
}

// TestWorldTransferAllFivePayloadsUnmarshalToConcreteTypes extends the brief's
// single-payload check across all five actions, since a switch-case typo
// (e.g. wrong payload type on the right action) would not be caught by
// TestWorldTransferStepsUnmarshalToConcretePayloads alone.
func TestWorldTransferAllFivePayloadsUnmarshalToConcreteTypes(t *testing.T) {
	pendingChangeId := uuid.New().String()
	cases := []struct {
		action  string
		payload string
		assert  func(t *testing.T, got any)
	}{
		{
			action:  "validate_world_transfer",
			payload: `{"characterId":1,"sourceWorldId":0,"destinationWorldId":1,"pendingChangeId":"` + pendingChangeId + `"}`,
			assert: func(t *testing.T, got any) {
				if _, ok := got.(ValidateWorldTransferPayload); !ok {
					t.Fatalf("payload type = %T, want ValidateWorldTransferPayload", got)
				}
			},
		},
		{
			action:  "leave_guild_for_transfer",
			payload: `{"characterId":1,"worldId":0,"guildId":5,"title":3}`,
			assert: func(t *testing.T, got any) {
				if _, ok := got.(LeaveGuildForTransferPayload); !ok {
					t.Fatalf("payload type = %T, want LeaveGuildForTransferPayload", got)
				}
			},
		},
		{
			action:  "leave_party_for_transfer",
			payload: `{"characterId":1,"worldId":0,"partyId":9}`,
			assert: func(t *testing.T, got any) {
				if _, ok := got.(LeavePartyForTransferPayload); !ok {
					t.Fatalf("payload type = %T, want LeavePartyForTransferPayload", got)
				}
			},
		},
		{
			action:  "sever_buddies_for_transfer",
			payload: `{"characterId":1,"worldId":0,"buddyIds":[2,3]}`,
			assert: func(t *testing.T, got any) {
				if _, ok := got.(SeverBuddiesForTransferPayload); !ok {
					t.Fatalf("payload type = %T, want SeverBuddiesForTransferPayload", got)
				}
			},
		},
		{
			action:  "change_character_world",
			payload: `{"characterId":1,"sourceWorldId":0,"destinationWorldId":1,"pendingChangeId":"` + pendingChangeId + `"}`,
			assert: func(t *testing.T, got any) {
				if _, ok := got.(ChangeCharacterWorldPayload); !ok {
					t.Fatalf("payload type = %T, want ChangeCharacterWorldPayload", got)
				}
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.action, func(t *testing.T) {
			raw := `{"action":"` + tc.action + `","payload":` + tc.payload + `}`
			var st Step[any]
			if err := json.Unmarshal([]byte(raw), &st); err != nil {
				t.Fatalf("unmarshal: %v", err)
			}
			tc.assert(t, st.Payload())
		})
	}
}

// TestSeverBuddiesForTransferEmptyBuddyIdsSelfCompletesCleanly asserts the
// "no buddies to sever" branch — the only one of the four self-completing
// world-transfer branches that runs in this package's own tests without a
// downstream processor call — succeeds cleanly: nil error, step Completed.
// Fix round 1 finding 2.
func TestSeverBuddiesForTransferEmptyBuddyIdsSelfCompletesCleanly(t *testing.T) {
	logger, hook := logtest.NewNullLogger()
	logger.SetLevel(logrus.DebugLevel)
	ctx := worldTransferTestCtx(t)

	tx := uuid.New()
	s, err := NewBuilder().
		SetTransactionId(tx).
		SetSagaType(InventoryTransaction).
		SetInitiatedBy("world-transfer-test").
		AddStep("sever_buddies", Pending, SeverBuddiesForTransfer, SeverBuddiesForTransferPayload{
			CharacterId: 1, WorldId: 0, BuddyIds: nil,
		}).
		Build()
	require.NoError(t, err)
	require.NoError(t, GetCache().Put(ctx, s))

	step, found := s.GetCurrentStep()
	require.True(t, found)

	err = NewHandler(logger, ctx).handleSeverBuddiesForTransfer(s, step)
	require.NoError(t, err, "an empty BuddyIds self-complete must not error")

	// The step was the saga's only step, so completing it makes the saga
	// itself terminal and it is evicted from the cache (Step()'s "no steps
	// remaining" branch) — a GetById lookup afterward correctly reports "not
	// found", which is the completion side effect, not a failure. Assert
	// completion directly off the debug log MarkEarliestPendingStepWithResult
	// emits on a successful write instead.
	completed := false
	for _, e := range hook.AllEntries() {
		if e.Data["step_id"] == "sever_buddies" && e.Message == "Marked earliest pending step as [completed]." {
			completed = true
			break
		}
	}
	assert.True(t, completed, "expected the sever_buddies step to be marked Completed")
}

// TestSelfCompletingWorldTransferStepSurfacesStepCompletedFailure proves fix
// round 1 finding 2: a self-completing branch's StepCompleted error must
// surface, not be swallowed by `_ = ...`. It forces a REAL StepCompleted
// failure (not a stub) by loading a saga whose stored step ordering is
// already invalid — step 2 Completed while step 1 is still Pending — so that
// marking step 0 (the pending leave_guild_for_transfer step, GuildId == 0)
// Completed leaves the saga with a Completed step positioned after a step
// that is still Pending. Saga.ValidateStateConsistency rejects exactly that
// shape (saga/model.go ValidateStepOrdering), so
// MarkEarliestPendingStepWithResult — and therefore StepCompleted — returns
// a real, non-nil error via the ordinary write path, not a test double.
func TestSelfCompletingWorldTransferStepSurfacesStepCompletedFailure(t *testing.T) {
	logger, hook := logtest.NewNullLogger()
	logger.SetLevel(logrus.DebugLevel)
	ctx := worldTransferTestCtx(t)

	tx := uuid.New()
	s, err := NewBuilder().
		SetTransactionId(tx).
		SetSagaType(InventoryTransaction).
		SetInitiatedBy("world-transfer-test").
		AddStep("leave_guild", Pending, LeaveGuildForTransfer, LeaveGuildForTransferPayload{
			CharacterId: 1, WorldId: 0, GuildId: 0,
		}).
		AddStep("leave_party", Pending, LeavePartyForTransfer, LeavePartyForTransferPayload{
			CharacterId: 1, WorldId: 0, PartyId: 0,
		}).
		AddStep("sever_buddies", Completed, SeverBuddiesForTransfer, SeverBuddiesForTransferPayload{
			CharacterId: 1, WorldId: 0, BuddyIds: nil,
		}).
		Build()
	require.NoError(t, err)
	require.NoError(t, GetCache().Put(ctx, s))

	step, found := s.GetCurrentStep()
	require.True(t, found)
	require.Equal(t, LeaveGuildForTransfer, step.Action())

	err = NewHandler(logger, ctx).handleLeaveGuildForTransfer(s, step)
	require.Error(t, err, "a real StepCompleted failure must surface as an error, not be swallowed")

	// The error must have been logged with the saga/step identifiers the
	// sibling handlers use (h.logActionError's fields), not dropped silently.
	found = false
	for _, e := range hook.AllEntries() {
		if e.Data["transaction_id"] == tx.String() && e.Data["step_id"] == "leave_guild" {
			found = true
			break
		}
	}
	assert.True(t, found, "expected an error log carrying transaction_id/step_id for the failed self-complete")
}
