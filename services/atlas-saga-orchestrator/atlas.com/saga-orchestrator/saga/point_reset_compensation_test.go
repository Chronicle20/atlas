package saga

import (
	compmock "atlas-saga-orchestrator/compartment/mock"
	"context"
	"testing"
	"time"

	"github.com/Chronicle20/atlas/libs/atlas-constants/channel"
	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"github.com/sirupsen/logrus/hooks/test"
	"github.com/stretchr/testify/assert"
)

// testTenantContext returns a context carrying a GMS v83 tenant, matching the
// pattern used by the PetEvolution compensation tests.
func testTenantContext() context.Context {
	te, _ := tenant.Create(uuid.New(), "GMS", 83, 1)
	return tenant.WithContext(context.Background(), te)
}

// TestPointResetCompensationReawardsDestroyedItem verifies the point_reset
// reverse-walk (task-126, shape B): when a transfer_ap step fails, the
// already-completed destroy_asset (the consumed AP Reset item) is re-awarded
// via RequestCreateItem with the destroyed template id and quantity.
//
// DispatchPointResetRollbacks is exercised directly (mirroring the
// PetEvolution compensation tests) to avoid the EmitSagaFailed Kafka path; no
// broker runs in the test environment.
func TestPointResetCompensationReawardsDestroyedItem(t *testing.T) {
	logger, _ := test.NewNullLogger()
	logger.SetLevel(logrus.DebugLevel)

	const (
		testCharId  = uint32(88001)
		apResetItem = uint32(5050000)
		testWorldId = world.Id(0)
		testChannel = channel.Id(1)
	)

	type createCall struct {
		CharacterId uint32
		TemplateId  uint32
		Quantity    uint32
	}
	var createItemCalls []createCall
	compP := &compmock.ProcessorMock{
		RequestCreateItemFunc: func(_ uuid.UUID, characterId uint32, templateId uint32, quantity uint32, _ time.Time) error {
			createItemCalls = append(createItemCalls, createCall{
				CharacterId: characterId,
				TemplateId:  templateId,
				Quantity:    quantity,
			})
			return nil
		},
	}

	transactionId := uuid.New()
	s, err := NewBuilder().
		SetTransactionId(transactionId).
		SetSagaType(PointReset).
		SetInitiatedBy("point-reset-compensation-test").
		AddStep("destroy_asset", Completed, DestroyAsset, DestroyAssetPayload{
			CharacterId: testCharId,
			TemplateId:  apResetItem,
			Quantity:    1,
			RemoveAll:   false,
		}).
		AddStep("transfer_ap", Failed, TransferAP, TransferAPPayload{
			CharacterId: testCharId,
			WorldId:     testWorldId,
			ChannelId:   testChannel,
			From:        "HP",
			To:          "STRENGTH",
		}).
		Build()
	assert.NoError(t, err, "saga build should not fail")

	compensator := NewCompensator(logger, testTenantContext()).
		WithCompartmentProcessor(compP)

	compensator.DispatchPointResetRollbacks(s)

	assert.Equal(t, 1, len(createItemCalls), "consumed item should be re-awarded exactly once")
	if len(createItemCalls) == 1 {
		assert.Equal(t, testCharId, createItemCalls[0].CharacterId, "re-award must target the test character")
		assert.Equal(t, apResetItem, createItemCalls[0].TemplateId, "re-awarded item must be the destroyed AP Reset item")
		assert.Equal(t, uint32(1), createItemCalls[0].Quantity, "re-awarded quantity must match the destroyed quantity")
	}
}

// TestPointResetCompensationSkipsUncompletedDestroy verifies that a destroy
// step that never completed is NOT re-awarded (the reverse-walk only inverts
// Completed mutations).
func TestPointResetCompensationSkipsUncompletedDestroy(t *testing.T) {
	logger, _ := test.NewNullLogger()

	var createCount int
	compP := &compmock.ProcessorMock{
		RequestCreateItemFunc: func(_ uuid.UUID, _ uint32, _ uint32, _ uint32, _ time.Time) error {
			createCount++
			return nil
		},
	}

	s, err := NewBuilder().
		SetTransactionId(uuid.New()).
		SetSagaType(PointReset).
		SetInitiatedBy("point-reset-compensation-test").
		AddStep("destroy_asset", Failed, DestroyAsset, DestroyAssetPayload{
			CharacterId: 88002,
			TemplateId:  5050001,
			Quantity:    1,
		}).
		Build()
	assert.NoError(t, err)

	NewCompensator(logger, testTenantContext()).
		WithCompartmentProcessor(compP).
		DispatchPointResetRollbacks(s)

	assert.Equal(t, 0, createCount, "an uncompleted destroy must not be re-awarded")
}

// TestPointResetFailureFields verifies the Task 14 error-threading contract:
// the failed step's result map { errorCode, errorDetail } is surfaced as the
// saga-failed event's ErrorCode and Reason (reason = errorDetail), with a
// fallback to ErrorCodeUnknown + a generic reason when the keys are absent.
func TestPointResetFailureFields(t *testing.T) {
	transactionId := uuid.New()

	// Threaded case: result map carries the service's code + detail.
	s, err := NewBuilder().
		SetTransactionId(transactionId).
		SetSagaType(PointReset).
		SetInitiatedBy("point-reset-error-threading-test").
		AddStep("destroy_asset", Completed, DestroyAsset, DestroyAssetPayload{
			CharacterId: 88003,
			TemplateId:  5050000,
			Quantity:    1,
		}).
		AddStep("transfer_ap", Failed, TransferAP, TransferAPPayload{
			CharacterId: 88003,
			From:        "HP",
			To:          "STRENGTH",
		}).
		Build()
	assert.NoError(t, err)

	s, err = s.WithStepResult(1, map[string]any{"errorCode": "POOL_BELOW_JOB_MINIMUM", "errorDetail": "HP"})
	assert.NoError(t, err)

	failed, _ := s.StepAt(1)
	code, reason := pointResetFailureFields(failed)
	assert.Equal(t, "POOL_BELOW_JOB_MINIMUM", code, "errorCode must thread the service's machine-readable code")
	assert.Equal(t, "HP", reason, "reason must equal errorDetail (the stat-name detail carrier)")

	// Fallback case: no result map → ErrorCodeUnknown + generic reason.
	failedNoResult, _ := s.StepAt(0)
	fbCode, fbReason := pointResetFailureFields(failedNoResult)
	assert.Equal(t, "UNKNOWN", fbCode, "missing result map must fall back to ErrorCodeUnknown")
	assert.Contains(t, fbReason, "Point reset failed at step", "fallback reason must be the generic descriptor")
}
