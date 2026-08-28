//go:build test

package playernpc

import (
	playernpcmsg "atlas-saga-orchestrator/kafka/message/playernpc"
	"atlas-saga-orchestrator/saga"
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus/hooks/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

// retainingCache wraps the real saga cache and swallows Remove, so a saga a
// compensator terminates stays inspectable afterward -- mirrors
// kafka/consumer/character/meso_error_result_test.go's seam, needed for the
// same reason: a single-step deploy_player_npc saga's COMMAND_FAILED
// cascades synchronously into compensation, which removes the saga from the
// cache before this test could otherwise inspect the failed step's result.
type retainingCache struct {
	saga.Cache
}

func (r retainingCache) Remove(ctx context.Context, transactionId uuid.UUID) bool {
	return true
}

// TestHandleCommandFailedEventThreadsErrorCode is the FR-6.3 assertion: the
// recorded result on the failed deploy_player_npc step carries errorCode
// equal to the event's Code, using a real design §8.3 value, so a
// conversation script can branch on pool_exhausted vs. map_full vs.
// ineligible without a new saga plumbing mechanism.
func TestHandleCommandFailedEventThreadsErrorCode(t *testing.T) {
	l, _ := test.NewNullLogger()
	te, err := tenant.Create(uuid.New(), "GMS", 83, 1)
	require.NoError(t, err)
	ctx := tenant.WithContext(context.Background(), te)

	realCache := saga.GetCache()
	saga.SetCache(retainingCache{Cache: realCache})
	t.Cleanup(func() { saga.SetCache(realCache) })

	tx := uuid.New()
	s, err := saga.NewBuilder().
		SetTransactionId(tx).
		SetSagaType(saga.RemoteNpcUse).
		SetInitiatedBy("errorcode-result-test").
		AddStep("s1", saga.Pending, saga.DeployPlayerNpc, saga.DeployPlayerNpcPayload{CharacterId: 1234}).
		Build()
	require.NoError(t, err)
	require.NoError(t, saga.GetCache().Put(ctx, s))
	t.Cleanup(func() { realCache.Remove(ctx, tx) })

	handleCommandFailedEvent(l, ctx, playernpcmsg.StatusEvent[playernpcmsg.StatusCommandOutcomeBody]{
		Type: playernpcmsg.EventTypeCommandFailed,
		Body: playernpcmsg.StatusCommandOutcomeBody{
			TransactionId: tx,
			CharacterId:   1234,
			CommandType:   playernpcmsg.CommandTypeDeploy,
			Code:          "pool_exhausted",
		},
	})

	got, ok := saga.GetCache().GetById(ctx, tx)
	require.True(t, ok, "saga must still be resolvable")
	step, ok := got.StepAt(0)
	require.True(t, ok)
	require.NotNil(t, step.Result(), "failed deploy_player_npc step must carry a result map")
	assert.Equal(t, "pool_exhausted", step.Result()["errorCode"])
}
