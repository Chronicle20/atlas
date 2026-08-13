//go:build test

package character

import (
	character2 "atlas-saga-orchestrator/kafka/message/character"
	"atlas-saga-orchestrator/saga"
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus/hooks/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

// retainingCache wraps the real saga cache and swallows Remove, so a saga
// that a compensator terminates (compensateMesoSackUse removes the saga from
// the cache as one of its first steps, once it wins the Compensating→Failed
// transition) stays inspectable afterward. This is a test-only interception
// of the existing saga.Cache seam (SetCache), not a new production API.
type retainingCache struct {
	saga.Cache
}

func (r retainingCache) Remove(ctx context.Context, transactionId uuid.UUID) bool {
	return true
}

// The meso-error handler used to call StepCompleted(tx, false) and drop
// Body.Error on the floor, so the meso_sack_use compensator had no way to tell
// a ceiling rejection from any other failure and would have rendered the
// generic message. It must thread the code onto the step's result map, exactly
// as handleCharacterApTransferErrorEvent already does; compensateMesoSackUse
// (saga/compensator.go) then reads that code via mesoSackErrorCode to render
// a meso-ceiling message instead of the generic failure.
//
// The handler call cascades synchronously into compensation, which removes
// the saga from the cache before this test can inspect it — hence
// retainingCache above.
func TestHandleCharacterMesoErrorEventThreadsErrorCode(t *testing.T) {
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
		SetSagaType(saga.MesoSackUse).
		SetInitiatedBy("meso-error-result-test").
		AddStep("consume_meso_sack", saga.Completed, saga.DestroyAsset, saga.DestroyAssetPayload{
			CharacterId: 4001, TemplateId: 5200000, Quantity: 1,
		}).
		AddStep("award_mesos", saga.Pending, saga.AwardMesos, saga.AwardMesosPayload{
			CharacterId: 4001, ActorId: 5200000, ActorType: "ITEM", Amount: 1000000, ShowEffect: true,
		}).
		Build()
	require.NoError(t, err)
	require.NoError(t, saga.GetCache().Put(ctx, s))
	t.Cleanup(func() { realCache.Remove(ctx, tx) })

	handleCharacterMesoErrorEvent(l, ctx, character2.StatusEvent[character2.StatusEventMesoErrorBody]{
		TransactionId: tx,
		CharacterId:   4001,
		Type:          character2.StatusEventTypeError,
		Body: character2.StatusEventMesoErrorBody{
			Error:  "MESO_OVERFLOW",
			Amount: 1000000,
		},
	})

	got, ok := saga.GetCache().GetById(ctx, tx)
	require.True(t, ok, "saga must still be resolvable")
	step, ok := got.StepAt(1)
	require.True(t, ok)
	require.NotNil(t, step.Result(), "failed award_mesos step must carry a result map")
	assert.Equal(t, "MESO_OVERFLOW", step.Result()["errorCode"])
}
