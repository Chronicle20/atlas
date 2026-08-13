//go:build test

package saga

import (
	compmock "atlas-saga-orchestrator/compartment/mock"
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"github.com/sirupsen/logrus/hooks/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Chronicle20/atlas/libs/atlas-constants/channel"
	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
)

const (
	mesoSackCharId  = uint32(77001)
	mesoSackItemId  = uint32(5200000)
	mesoSackAmount  = int32(1000000)
	mesoSackWorldId = world.Id(0)
	mesoSackChannel = channel.Id(1)
)

func newMesoSackSaga(t *testing.T, tx uuid.UUID, destroyStatus Status) Saga {
	t.Helper()
	s, err := NewBuilder().
		SetTransactionId(tx).
		SetSagaType(MesoSackUse).
		SetInitiatedBy("meso-sack-compensation-test").
		AddStep("consume_meso_sack", destroyStatus, DestroyAsset, DestroyAssetPayload{
			CharacterId: mesoSackCharId,
			TemplateId:  mesoSackItemId,
			Quantity:    1,
			RemoveAll:   false,
		}).
		AddStep("award_mesos", Failed, AwardMesos, AwardMesosPayload{
			CharacterId: mesoSackCharId,
			WorldId:     mesoSackWorldId,
			ChannelId:   mesoSackChannel,
			ActorId:     mesoSackItemId,
			ActorType:   "ITEM",
			Amount:      mesoSackAmount,
			ShowEffect:  true,
		}).
		Build()
	require.NoError(t, err)
	return s
}

// A failed award must refund the already-consumed sack — exactly once, with the
// destroyed template and quantity. Without this the player pays a cash item for
// nothing.
func TestMesoSackCompensationRefundsSack(t *testing.T) {
	logger, _ := test.NewNullLogger()
	logger.SetLevel(logrus.DebugLevel)

	type createCall struct {
		CharacterId uint32
		TemplateId  uint32
		Quantity    uint32
	}
	var calls []createCall
	compP := &compmock.ProcessorMock{
		RequestCreateItemFunc: func(_ uuid.UUID, characterId uint32, templateId uint32, quantity uint32, _ time.Time) error {
			calls = append(calls, createCall{characterId, templateId, quantity})
			return nil
		},
	}

	s := newMesoSackSaga(t, uuid.New(), Completed)
	NewCompensator(logger, testTenantContext()).
		WithCompartmentProcessor(compP).
		DispatchMesoSackRollbacks(s)

	require.Len(t, calls, 1, "consumed sack must be refunded exactly once")
	assert.Equal(t, mesoSackCharId, calls[0].CharacterId)
	assert.Equal(t, mesoSackItemId, calls[0].TemplateId)
	assert.Equal(t, uint32(1), calls[0].Quantity)
}

// A consume step that never completed committed nothing and has no inverse.
func TestMesoSackCompensationSkipsUncompletedConsume(t *testing.T) {
	logger, _ := test.NewNullLogger()

	var count int
	compP := &compmock.ProcessorMock{
		RequestCreateItemFunc: func(_ uuid.UUID, _ uint32, _ uint32, _ uint32, _ time.Time) error {
			count++
			return nil
		},
	}

	s := newMesoSackSaga(t, uuid.New(), Failed)
	NewCompensator(logger, testTenantContext()).
		WithCompartmentProcessor(compP).
		DispatchMesoSackRollbacks(s)

	assert.Equal(t, 0, count, "an uncompleted destroy must not be inverted")
}

// THE regression guard for the characterId-0 bug: EmitSagaFailed's default
// extractor only recognizes a CreateCharacter step, so a meso_sack_use failure
// would emit characterId 0, the channel's session lookup would miss, and the
// player would get silence AND stay input-locked.
//
// NOTE on the "confirm before writing" helpers from the task brief:
// Step[any] has no WithResult method — only Saga.WithStepResult(index, result)
// exists (verified in saga/model.go and exercised by
// saga/point_reset_compensation_test.go's TestPointResetFailureFields). This
// test therefore seeds the result via s.WithStepResult(1, ...) and re-reads
// the step with StepAt(1), rather than calling a nonexistent
// Step[any].WithResult. GetCache().Put(ctx, s) does exist as documented.
func TestMesoSackFailedEventCarriesRealCharacterIdAndErrorCode(t *testing.T) {
	logger, _ := test.NewNullLogger()

	type emitted struct {
		SagaType    string
		CharacterId uint32
		ErrorCode   string
		FailedStep  string
	}
	var got []emitted
	restore := SetEmitSagaFailedForTest(func(_ logrus.FieldLogger, _ context.Context, _ uuid.UUID, sagaType string, _ uint32, characterId uint32, errorCode string, _ string, failedStep string) error {
		got = append(got, emitted{sagaType, characterId, errorCode, failedStep})
		return nil
	})
	t.Cleanup(func() { SetEmitSagaFailedForTest(restore) })

	compP := &compmock.ProcessorMock{
		RequestCreateItemFunc: func(_ uuid.UUID, _ uint32, _ uint32, _ uint32, _ time.Time) error { return nil },
	}

	ctx := testTenantContext()
	tx := uuid.New()
	s := newMesoSackSaga(t, tx, Completed)
	require.NoError(t, GetCache().Put(ctx, s))
	t.Cleanup(func() { GetCache().Remove(ctx, tx) })
	require.True(t, GetCache().TryTransition(ctx, tx, SagaLifecyclePending, SagaLifecycleCompensating))

	s, err := s.WithStepResult(1, map[string]any{"errorCode": "MESO_OVERFLOW"})
	require.NoError(t, err)
	failedStep, ok := s.StepAt(1)
	require.True(t, ok)

	c := NewCompensator(logger, ctx).WithCompartmentProcessor(compP)
	require.NoError(t, c.(*CompensatorImpl).compensateMesoSackUse(s, failedStep))

	require.Len(t, got, 1, "exactly one saga-failed emission")
	assert.Equal(t, "meso_sack_use", got[0].SagaType)
	assert.Equal(t, mesoSackCharId, got[0].CharacterId, "characterId must be the payload's, never 0")
	assert.Equal(t, "MESO_OVERFLOW", got[0].ErrorCode)
	assert.Equal(t, "award_mesos", got[0].FailedStep)
}
