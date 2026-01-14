package saga

import (
	"atlas-saga-orchestrator/saga"
	"context"
	"testing"

	tenant "github.com/Chronicle20/atlas-tenant"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"github.com/sirupsen/logrus/hooks/test"
	"github.com/stretchr/testify/assert"
)

func TestHandleSagaCommand(t *testing.T) {
	tests := []struct {
		name     string
		saga     saga.Saga
		validate func(t *testing.T, logger *logrus.Logger, hook *test.Hook)
	}{
		{
			name: "Successfully handles saga command",
			saga: func() saga.Saga {
				s, _ := saga.NewBuilder().
					SetTransactionId(uuid.New()).
					SetSagaType(saga.QuestReward).
					SetInitiatedBy("test-initiator").
					AddStep("step-1", saga.Pending, saga.AwardInventory, saga.AwardItemActionPayload{
						CharacterId: 12345,
						Item: saga.ItemPayload{
							TemplateId: 1000000,
							Quantity:   1,
						},
					}).
					Build()
				return s
			}(),
			validate: func(t *testing.T, logger *logrus.Logger, hook *test.Hook) {
				// Verify logs contain expected fields
				assert.True(t, len(hook.Entries) >= 1, "Should have at least one log entry")

				// Find the "Handling saga command" log entry
				var foundHandlingLog bool
				for _, entry := range hook.Entries {
					if entry.Message == "Handling saga command" {
						foundHandlingLog = true
						assert.Contains(t, entry.Data, "transaction_id")
						assert.Contains(t, entry.Data, "saga_type")
						assert.Contains(t, entry.Data, "initiated_by")
						assert.Contains(t, entry.Data, "steps_count")
						break
					}
				}
				assert.True(t, foundHandlingLog, "Should have 'Handling saga command' log entry")
			},
		},
		{
			name: "Handles saga with multiple steps",
			saga: func() saga.Saga {
				s, _ := saga.NewBuilder().
					SetTransactionId(uuid.New()).
					SetSagaType(saga.InventoryTransaction).
					SetInitiatedBy("multi-step-test").
					AddStep("step-1", saga.Pending, saga.AwardInventory, saga.AwardItemActionPayload{
						CharacterId: 12345,
						Item: saga.ItemPayload{
							TemplateId: 1000000,
							Quantity:   1,
						},
					}).
					AddStep("step-2", saga.Pending, saga.AwardMesos, saga.AwardMesosPayload{
						CharacterId: 12345,
						Amount:      1000,
					}).
					Build()
				return s
			}(),
			validate: func(t *testing.T, logger *logrus.Logger, hook *test.Hook) {
				// Verify the saga was processed
				assert.True(t, len(hook.Entries) >= 1, "Should have at least one log entry")

				// Find the log entry with steps_count
				for _, entry := range hook.Entries {
					if entry.Message == "Handling saga command" {
						stepsCount := entry.Data["steps_count"]
						assert.Equal(t, 2, stepsCount, "Should have 2 steps")
						break
					}
				}
			},
		},
		{
			name: "Handles empty saga (no steps)",
			saga: func() saga.Saga {
				s, _ := saga.NewBuilder().
					SetTransactionId(uuid.New()).
					SetSagaType(saga.CharacterCreation).
					SetInitiatedBy("empty-saga-test").
					Build()
				return s
			}(),
			validate: func(t *testing.T, logger *logrus.Logger, hook *test.Hook) {
				assert.True(t, len(hook.Entries) >= 1, "Should have at least one log entry")

				// Find the log entry with steps_count
				for _, entry := range hook.Entries {
					if entry.Message == "Handling saga command" {
						stepsCount := entry.Data["steps_count"]
						assert.Equal(t, 0, stepsCount, "Should have 0 steps")
						break
					}
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup
			logger, hook := test.NewNullLogger()
			logger.SetLevel(logrus.DebugLevel)

			ctx := context.Background()
			te, _ := tenant.Create(uuid.New(), "GMS", 83, 1)
			tctx := tenant.WithContext(ctx, te)

			// Execute
			handleSagaCommand(logger, tctx, tt.saga)

			// Validate
			tt.validate(t, logger, hook)
		})
	}
}

func TestHandleSagaCommandLogging(t *testing.T) {
	// Setup
	logger, hook := test.NewNullLogger()
	logger.SetLevel(logrus.DebugLevel)

	ctx := context.Background()
	te, _ := tenant.Create(uuid.New(), "GMS", 83, 1)
	tctx := tenant.WithContext(ctx, te)

	transactionId := uuid.New()
	testSaga, _ := saga.NewBuilder().
		SetTransactionId(transactionId).
		SetSagaType(saga.QuestReward).
		SetInitiatedBy("logging-test").
		AddStep("test-step", saga.Pending, saga.AwardMesos, saga.AwardMesosPayload{
			CharacterId: 12345,
			Amount:      1000,
		}).
		Build()

	// Execute
	handleSagaCommand(logger, tctx, testSaga)

	// Verify all expected log fields are present
	var foundHandlingLog bool
	for _, entry := range hook.Entries {
		if entry.Message == "Handling saga command" {
			foundHandlingLog = true

			// Verify transaction ID
			tid, ok := entry.Data["transaction_id"]
			assert.True(t, ok, "Should have transaction_id field")
			assert.Equal(t, transactionId.String(), tid)

			// Verify saga type
			sagaType, ok := entry.Data["saga_type"]
			assert.True(t, ok, "Should have saga_type field")
			assert.Equal(t, saga.QuestReward, sagaType)

			// Verify initiated by
			initiatedBy, ok := entry.Data["initiated_by"]
			assert.True(t, ok, "Should have initiated_by field")
			assert.Equal(t, "logging-test", initiatedBy)

			// Verify steps count
			stepsCount, ok := entry.Data["steps_count"]
			assert.True(t, ok, "Should have steps_count field")
			assert.Equal(t, 1, stepsCount)

			break
		}
	}
	assert.True(t, foundHandlingLog, "Should have found 'Handling saga command' log entry")
}
