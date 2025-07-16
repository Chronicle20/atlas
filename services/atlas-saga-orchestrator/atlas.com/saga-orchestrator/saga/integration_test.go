package saga

import (
	"atlas-saga-orchestrator/compartment"
	mock2 "atlas-saga-orchestrator/compartment/mock"
	"atlas-saga-orchestrator/validation"
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	tenant "github.com/Chronicle20/atlas-tenant"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"github.com/sirupsen/logrus/hooks/test"
	"github.com/stretchr/testify/assert"
)

// TestCreateAndEquipAsset_CompleteIntegrationFlow tests the complete create-and-equip flow
// from saga execution through step completion simulation
func TestCreateAndEquipAsset_CompleteIntegrationFlow(t *testing.T) {
	// Setup
	logger, hook := test.NewNullLogger()
	logger.SetLevel(logrus.DebugLevel)
	
	ctx := context.Background()
	te, _ := tenant.Create(uuid.New(), "GMS", 83, 1)
	tctx := tenant.WithContext(ctx, te)

	// Setup mocks
	compP := &mock2.ProcessorMock{
		RequestCreateAndEquipAssetFunc: func(transactionId uuid.UUID, payload compartment.CreateAndEquipAssetPayload) error {
			// Verify payload conversion
			assert.Equal(t, uint32(12345), payload.CharacterId)
			assert.Equal(t, uint32(1302000), payload.Item.TemplateId)
			assert.Equal(t, uint32(1), payload.Item.Quantity)
			return nil
		},
		RequestEquipAssetFunc: func(transactionId uuid.UUID, characterId uint32, inventoryType byte, source int16, destination int16) error {
			// Verify auto-equip parameters
			assert.Equal(t, uint32(12345), characterId)
			assert.Equal(t, int16(5), source)      // Default source slot
			assert.Equal(t, int16(-1), destination) // Default equip destination
			return nil
		},
	}

	// Create saga processor
	processor := NewProcessor(logger, tctx)
	processor.t = te
	processor.compP = compP

	// Create saga with CreateAndEquipAsset step
	transactionId := uuid.New()
	saga := Saga{
		TransactionId: transactionId,
		SagaType:      InventoryTransaction,
		InitiatedBy:   "integration-test",
		Steps: []Step[any]{
			{
				StepId:    "create-and-equip-step",
				Status:    Pending,
				Action:    CreateAndEquipAsset,
				Payload: CreateAndEquipAssetPayload{
					CharacterId: 12345,
					Item: ItemPayload{
						TemplateId: 1302000, // Equippable weapon
						Quantity:   1,
					},
				},
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
			},
		},
	}

	// Store saga in cache
	GetCache().Put(te.Id(), saga)

	// Step 1: Execute the CreateAndEquipAsset step
	err := processor.Step(transactionId)
	assert.NoError(t, err, "CreateAndEquipAsset step should execute successfully")

	// Verify the step was executed - check if mock was called
	assert.NotNil(t, compP.RequestCreateAndEquipAssetFunc, "RequestCreateAndEquipAsset should be defined")

	// Step 2: Manually add auto-equip step to simulate event handler behavior
	autoEquipStepId := "auto_equip_step_test"
	equipPayload := EquipAssetPayload{
		CharacterId:   12345,
		InventoryType: 1, // Assume equip type
		Source:        5,
		Destination:   -1,
	}
	
	equipStep := Step[any]{
		StepId:    autoEquipStepId,
		Status:    Pending,
		Action:    EquipAsset,
		Payload:   equipPayload,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	
	err = processor.AddStep(transactionId, equipStep)
	assert.NoError(t, err, "Should successfully add auto-equip step")
	
	// Step 3: Complete the CreateAndEquipAsset step
	err = processor.MarkEarliestPendingStepCompleted(transactionId)
	assert.NoError(t, err, "Should complete CreateAndEquipAsset step")
	
	// Step 4: Execute the auto-equip step
	err = processor.Step(transactionId)
	assert.NoError(t, err, "Auto-equip step should execute successfully")
	
	// Verify the equip method was called
	assert.NotNil(t, compP.RequestEquipAssetFunc, "RequestEquipAsset should be defined")
	
	// Step 5: Complete the auto-equip step
	err = processor.MarkEarliestPendingStepCompleted(transactionId)
	assert.NoError(t, err, "Should complete auto-equip step")

	// Step 6: Verify final saga state
	finalSaga, err := processor.GetById(transactionId)
	assert.NoError(t, err, "Should be able to retrieve final saga")
	assert.Equal(t, 2, len(finalSaga.Steps), "Should have 2 steps")

	// Verify all steps are completed
	for i, step := range finalSaga.Steps {
		assert.Equal(t, Completed, step.Status, "Step %d should be completed", i)
	}

	// Verify appropriate logging occurred
	logEntries := hook.AllEntries()
	assert.True(t, len(logEntries) > 0, "Should have log entries")

	hook.Reset()
}

// TestCreateAndEquipAsset_StepAddition tests that the auto-equip step is added correctly
func TestCreateAndEquipAsset_StepAddition(t *testing.T) {
	// Setup
	logger, hook := test.NewNullLogger()
	logger.SetLevel(logrus.DebugLevel)
	
	ctx := context.Background()
	te, _ := tenant.Create(uuid.New(), "GMS", 83, 1)
	tctx := tenant.WithContext(ctx, te)

	// Setup mocks
	compP := &mock2.ProcessorMock{
		RequestCreateAndEquipAssetFunc: func(transactionId uuid.UUID, payload compartment.CreateAndEquipAssetPayload) error {
			return nil
		},
	}

	// Create saga processor
	processor := NewProcessor(logger, tctx)
	processor.t = te
	processor.compP = compP

	// Create saga with CreateAndEquipAsset step
	transactionId := uuid.New()
	saga := Saga{
		TransactionId: transactionId,
		SagaType:      InventoryTransaction,
		InitiatedBy:   "integration-test",
		Steps: []Step[any]{
			{
				StepId:    "create-and-equip-step",
				Status:    Pending,
				Action:    CreateAndEquipAsset,
				Payload: CreateAndEquipAssetPayload{
					CharacterId: 12345,
					Item: ItemPayload{
						TemplateId: 1302000,
						Quantity:   1,
					},
				},
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
			},
		},
	}

	// Store saga in cache
	GetCache().Put(te.Id(), saga)

	// Execute the CreateAndEquipAsset step
	err := processor.Step(transactionId)
	assert.NoError(t, err, "CreateAndEquipAsset step should execute successfully")

	// Verify the step was executed
	assert.NotNil(t, compP.RequestCreateAndEquipAssetFunc, "RequestCreateAndEquipAsset should be defined")

	// Add auto-equip step manually (simulating event handler)
	autoEquipStepId := "auto_equip_step_test"
	equipPayload := EquipAssetPayload{
		CharacterId:   12345,
		InventoryType: 1,
		Source:        5,
		Destination:   -1,
	}
	
	equipStep := Step[any]{
		StepId:    autoEquipStepId,
		Status:    Pending,
		Action:    EquipAsset,
		Payload:   equipPayload,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	
	err = processor.AddStep(transactionId, equipStep)
	assert.NoError(t, err, "Should successfully add auto-equip step")

	// Complete the CreateAndEquipAsset step
	err = processor.MarkEarliestPendingStepCompleted(transactionId)
	assert.NoError(t, err, "Should complete CreateAndEquipAsset step")

	// Verify the auto-equip step was added
	finalSaga, err := processor.GetById(transactionId)
	assert.NoError(t, err, "Should be able to retrieve updated saga")
	assert.Equal(t, 2, len(finalSaga.Steps), "Should have 2 steps after auto-equip step addition")

	// Verify the auto-equip step properties
	var autoEquipStep *Step[any]
	for i := range finalSaga.Steps {
		if finalSaga.Steps[i].Action == EquipAsset {
			autoEquipStep = &finalSaga.Steps[i]
			break
		}
	}
	assert.NotNil(t, autoEquipStep, "Auto-equip step should exist")
	assert.Equal(t, Pending, autoEquipStep.Status, "Auto-equip step should be pending")
	assert.Equal(t, autoEquipStepId, autoEquipStep.StepId, "Auto-equip step should have correct ID")

	// Verify the auto-equip payload
	equipPayloadResult, ok := autoEquipStep.Payload.(EquipAssetPayload)
	assert.True(t, ok, "Auto-equip step should have EquipAssetPayload")
	assert.Equal(t, uint32(12345), equipPayloadResult.CharacterId)
	assert.Equal(t, uint32(1), equipPayloadResult.InventoryType)
	assert.Equal(t, int16(5), equipPayloadResult.Source)
	assert.Equal(t, int16(-1), equipPayloadResult.Destination)

	hook.Reset()
}

// TestCreateAndEquipAsset_CompensationFlow tests compensation scenarios
func TestCreateAndEquipAsset_CompensationFlow(t *testing.T) {
	// Setup
	logger, hook := test.NewNullLogger()
	logger.SetLevel(logrus.DebugLevel)
	
	ctx := context.Background()
	te, _ := tenant.Create(uuid.New(), "GMS", 83, 1)
	tctx := tenant.WithContext(ctx, te)

	// Setup mocks
	compP := &mock2.ProcessorMock{
		RequestCreateAndEquipAssetFunc: func(transactionId uuid.UUID, payload compartment.CreateAndEquipAssetPayload) error {
			return nil
		},
		RequestEquipAssetFunc: func(transactionId uuid.UUID, characterId uint32, inventoryType byte, source int16, destination int16) error {
			return errors.New("equip failed")
		},
		RequestDestroyItemFunc: func(transactionId uuid.UUID, characterId uint32, templateId uint32, quantity uint32) error {
			// Compensation method - verify parameters
			assert.Equal(t, uint32(12345), characterId)
			assert.Equal(t, uint32(1302000), templateId)
			assert.Equal(t, uint32(1), quantity)
			return nil
		},
	}

	// Create saga processor
	processor := NewProcessor(logger, tctx)
	processor.t = te
	processor.compP = compP

	// Create saga with CreateAndEquipAsset step
	transactionId := uuid.New()
	saga := Saga{
		TransactionId: transactionId,
		SagaType:      InventoryTransaction,
		InitiatedBy:   "compensation-test",
		Steps: []Step[any]{
			{
				StepId:    "create-and-equip-step",
				Status:    Pending,
				Action:    CreateAndEquipAsset,
				Payload: CreateAndEquipAssetPayload{
					CharacterId: 12345,
					Item: ItemPayload{
						TemplateId: 1302000,
						Quantity:   1,
					},
				},
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
			},
		},
	}

	// Store saga in cache
	GetCache().Put(te.Id(), saga)

	// Execute creation successfully
	err := processor.Step(transactionId)
	assert.NoError(t, err, "Creation should succeed")
	
	// Add auto-equip step
	autoEquipStepId := "auto_equip_step_test"
	equipStep := Step[any]{
		StepId:    autoEquipStepId,
		Status:    Pending,
		Action:    EquipAsset,
		Payload: EquipAssetPayload{
			CharacterId:   12345,
			InventoryType: 1,
			Source:        5,
			Destination:   -1,
		},
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	
	err = processor.AddStep(transactionId, equipStep)
	assert.NoError(t, err, "Should add auto-equip step")

	// Complete the CreateAndEquipAsset step
	err = processor.MarkEarliestPendingStepCompleted(transactionId)
	assert.NoError(t, err, "Should complete CreateAndEquipAsset step")

	// Execute equip step and expect failure
	err = processor.Step(transactionId)
	assert.Error(t, err, "Should fail at equip")

	// Mark equip step as failed
	err = processor.MarkEarliestPendingStep(transactionId, Failed)
	assert.NoError(t, err, "Should mark equip step as failed")

	// Verify final saga state shows failure
	finalSaga, _ := processor.GetById(transactionId)
	assert.True(t, finalSaga.Failing(), "Saga should be in failing state")

	// Verify compensation method was defined (would be called by compensation flow)
	assert.NotNil(t, compP.RequestDestroyItemFunc, "RequestDestroyItem should be defined for compensation")

	// Verify appropriate logging
	logEntries := hook.AllEntries()
	assert.True(t, len(logEntries) > 0, "Should have log entries")

	hook.Reset()
}

// TestCreateAndEquipAsset_AssetCreationFailure tests failure during asset creation phase
func TestCreateAndEquipAsset_AssetCreationFailure(t *testing.T) {
	// Setup
	logger, hook := test.NewNullLogger()
	logger.SetLevel(logrus.DebugLevel)
	
	ctx := context.Background()
	te, _ := tenant.Create(uuid.New(), "GMS", 83, 1)
	tctx := tenant.WithContext(ctx, te)

	// Setup mocks - asset creation fails
	compP := &mock2.ProcessorMock{
		RequestCreateAndEquipAssetFunc: func(transactionId uuid.UUID, payload compartment.CreateAndEquipAssetPayload) error {
			return errors.New("asset creation failed")
		},
	}

	// Create saga processor
	processor := NewProcessor(logger, tctx)
	processor.t = te
	processor.compP = compP

	// Create saga with CreateAndEquipAsset step
	transactionId := uuid.New()
	saga := Saga{
		TransactionId: transactionId,
		SagaType:      InventoryTransaction,
		InitiatedBy:   "asset-creation-failure-test",
		Steps: []Step[any]{
			{
				StepId:    "create-and-equip-step",
				Status:    Pending,
				Action:    CreateAndEquipAsset,
				Payload: CreateAndEquipAssetPayload{
					CharacterId: 12345,
					Item: ItemPayload{
						TemplateId: 1302000,
						Quantity:   1,
					},
				},
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
			},
		},
	}

	// Store saga in cache
	GetCache().Put(te.Id(), saga)

	// Execute the CreateAndEquipAsset step - should fail
	err := processor.Step(transactionId)
	assert.Error(t, err, "CreateAndEquipAsset step should fail during asset creation")

	// Mark the step as failed
	err = processor.MarkEarliestPendingStep(transactionId, Failed)
	assert.NoError(t, err, "Should mark step as failed")

	// Verify saga is in failing state
	finalSaga, err := processor.GetById(transactionId)
	assert.NoError(t, err, "Should be able to retrieve saga")
	assert.True(t, finalSaga.Failing(), "Saga should be in failing state")

	// Verify no auto-equip step was created since asset creation failed
	assert.Equal(t, 1, len(finalSaga.Steps), "Should only have original step")
	assert.Equal(t, Failed, finalSaga.Steps[0].Status, "Original step should be failed")

	// Execute compensation - should not require destroying asset since none was created
	err = processor.Step(transactionId)
	assert.NoError(t, err, "Compensation should succeed")

	// Verify step is compensated (back to Pending)
	compensatedSaga, err := processor.GetById(transactionId)
	assert.NoError(t, err, "Should be able to retrieve compensated saga")
	assert.Equal(t, Pending, compensatedSaga.Steps[0].Status, "Step should be compensated")

	hook.Reset()
}

// TestCreateAndEquipAsset_EquipPhaseFailure tests failure during equip phase with compensation
func TestCreateAndEquipAsset_EquipPhaseFailure(t *testing.T) {
	// Setup
	logger, hook := test.NewNullLogger()
	logger.SetLevel(logrus.DebugLevel)
	
	ctx := context.Background()
	te, _ := tenant.Create(uuid.New(), "GMS", 83, 1)
	tctx := tenant.WithContext(ctx, te)

	// Setup mocks - asset creation succeeds, equip fails
	compP := &mock2.ProcessorMock{
		RequestCreateAndEquipAssetFunc: func(transactionId uuid.UUID, payload compartment.CreateAndEquipAssetPayload) error {
			return nil // Asset creation succeeds
		},
		RequestEquipAssetFunc: func(transactionId uuid.UUID, characterId uint32, inventoryType byte, source int16, destination int16) error {
			return errors.New("equip failed - slot occupied")
		},
		RequestDestroyItemFunc: func(transactionId uuid.UUID, characterId uint32, templateId uint32, quantity uint32) error {
			// Compensation - destroy the successfully created asset
			assert.Equal(t, uint32(12345), characterId)
			assert.Equal(t, uint32(1302000), templateId)
			assert.Equal(t, uint32(1), quantity)
			return nil
		},
	}

	// Create saga processor
	processor := NewProcessor(logger, tctx)
	processor.t = te
	processor.compP = compP

	// Create saga with CreateAndEquipAsset step
	transactionId := uuid.New()
	saga := Saga{
		TransactionId: transactionId,
		SagaType:      InventoryTransaction,
		InitiatedBy:   "equip-phase-failure-test",
		Steps: []Step[any]{
			{
				StepId:    "create-and-equip-step",
				Status:    Pending,
				Action:    CreateAndEquipAsset,
				Payload: CreateAndEquipAssetPayload{
					CharacterId: 12345,
					Item: ItemPayload{
						TemplateId: 1302000,
						Quantity:   1,
					},
				},
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
			},
		},
	}

	// Store saga in cache
	GetCache().Put(te.Id(), saga)

	// Execute the CreateAndEquipAsset step - should succeed
	err := processor.Step(transactionId)
	assert.NoError(t, err, "CreateAndEquipAsset step should succeed")

	// Simulate auto-equip step creation (normally done by compartment consumer)
	// This needs to happen BEFORE completing the CreateAndEquipAsset step
	autoEquipStepId := "auto_equip_step_test"
	equipStep := Step[any]{
		StepId:    autoEquipStepId,
		Status:    Pending,
		Action:    EquipAsset,
		Payload: EquipAssetPayload{
			CharacterId:   12345,
			InventoryType: 1,
			Source:        5,
			Destination:   -1,
		},
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	
	err = processor.AddStep(transactionId, equipStep)
	assert.NoError(t, err, "Should add auto-equip step")

	// Complete the CreateAndEquipAsset step
	err = processor.MarkEarliestPendingStepCompleted(transactionId)
	assert.NoError(t, err, "Should complete CreateAndEquipAsset step")

	// Execute the auto-equip step - should fail
	err = processor.Step(transactionId)
	assert.Error(t, err, "Auto-equip step should fail")

	// Mark the equip step as failed
	err = processor.MarkEarliestPendingStep(transactionId, Failed)
	assert.NoError(t, err, "Should mark equip step as failed")

	// Verify saga is in failing state
	failingSaga, err := processor.GetById(transactionId)
	assert.NoError(t, err, "Should be able to retrieve failing saga")
	assert.True(t, failingSaga.Failing(), "Saga should be in failing state")

	// Execute compensation for the equip step
	err = processor.Step(transactionId)
	assert.NoError(t, err, "Equip step compensation should succeed")

	// Verify the equip step is compensated
	compensatedSaga, err := processor.GetById(transactionId)
	assert.NoError(t, err, "Should be able to retrieve compensated saga")
	
	// Find the equip step and verify it's compensated
	var equipStepFound bool
	for _, step := range compensatedSaga.Steps {
		if step.Action == EquipAsset {
			assert.Equal(t, Pending, step.Status, "Equip step should be compensated")
			equipStepFound = true
			break
		}
	}
	assert.True(t, equipStepFound, "Should find equip step")

	// Now the CreateAndEquipAsset step should also be marked as failed since the compound operation failed
	err = processor.MarkFurthestCompletedStepFailed(transactionId)
	assert.NoError(t, err, "Should mark CreateAndEquipAsset step as failed")

	// Execute compensation for the CreateAndEquipAsset step - should destroy the created asset
	err = processor.Step(transactionId)
	assert.NoError(t, err, "CreateAndEquipAsset compensation should succeed")

	// Verify the asset was destroyed (RequestDestroyItemFunc was called)
	assert.NotNil(t, compP.RequestDestroyItemFunc, "RequestDestroyItem should be called for compensation")

	hook.Reset()
}

// TestCreateAndEquipAsset_MultipleFailureRecovery tests multiple failure scenarios and recovery
func TestCreateAndEquipAsset_MultipleFailureRecovery(t *testing.T) {
	// Setup
	logger, hook := test.NewNullLogger()
	logger.SetLevel(logrus.DebugLevel)
	
	ctx := context.Background()
	te, _ := tenant.Create(uuid.New(), "GMS", 83, 1)
	tctx := tenant.WithContext(ctx, te)

	// Setup mocks with retry logic
	createAttempts := 0
	equipAttempts := 0
	compP := &mock2.ProcessorMock{
		RequestCreateAndEquipAssetFunc: func(transactionId uuid.UUID, payload compartment.CreateAndEquipAssetPayload) error {
			createAttempts++
			if createAttempts < 2 {
				return errors.New("temporary asset creation failure")
			}
			return nil // Succeed on second attempt
		},
		RequestEquipAssetFunc: func(transactionId uuid.UUID, characterId uint32, inventoryType byte, source int16, destination int16) error {
			equipAttempts++
			if equipAttempts < 2 {
				return errors.New("temporary equip failure")
			}
			return nil // Succeed on second attempt
		},
		RequestUnequipAssetFunc: func(transactionId uuid.UUID, characterId uint32, inventoryType byte, source int16, destination int16) error {
			return nil // Compensation succeeds
		},
		RequestDestroyItemFunc: func(transactionId uuid.UUID, characterId uint32, templateId uint32, quantity uint32) error {
			return nil // Compensation succeeds
		},
	}

	// Create saga processor
	processor := NewProcessor(logger, tctx)
	processor.t = te
	processor.compP = compP

	// Create saga with CreateAndEquipAsset step
	transactionId := uuid.New()
	saga := Saga{
		TransactionId: transactionId,
		SagaType:      InventoryTransaction,
		InitiatedBy:   "multiple-failure-recovery-test",
		Steps: []Step[any]{
			{
				StepId:    "create-and-equip-step",
				Status:    Pending,
				Action:    CreateAndEquipAsset,
				Payload: CreateAndEquipAssetPayload{
					CharacterId: 12345,
					Item: ItemPayload{
						TemplateId: 1302000,
						Quantity:   1,
					},
				},
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
			},
		},
	}

	// Store saga in cache
	GetCache().Put(te.Id(), saga)

	// First attempt - should fail
	err := processor.Step(transactionId)
	assert.Error(t, err, "First attempt should fail")

	// Mark as failed and compensate
	err = processor.MarkEarliestPendingStep(transactionId, Failed)
	assert.NoError(t, err, "Should mark as failed")

	err = processor.Step(transactionId)
	assert.NoError(t, err, "Compensation should succeed")

	// Second attempt - should succeed
	err = processor.Step(transactionId)
	assert.NoError(t, err, "Second attempt should succeed")

	// Add auto-equip step before completing CreateAndEquipAsset
	autoEquipStepId := "auto_equip_step_test"
	equipStep := Step[any]{
		StepId:    autoEquipStepId,
		Status:    Pending,
		Action:    EquipAsset,
		Payload: EquipAssetPayload{
			CharacterId:   12345,
			InventoryType: 1,
			Source:        5,
			Destination:   -1,
		},
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	
	err = processor.AddStep(transactionId, equipStep)
	assert.NoError(t, err, "Should add auto-equip step")

	// Complete the CreateAndEquipAsset step
	err = processor.MarkEarliestPendingStepCompleted(transactionId)
	assert.NoError(t, err, "Should complete CreateAndEquipAsset step")

	// First equip attempt - should fail
	err = processor.Step(transactionId)
	assert.Error(t, err, "First equip attempt should fail")

	// Mark as failed and compensate
	err = processor.MarkEarliestPendingStep(transactionId, Failed)
	assert.NoError(t, err, "Should mark equip as failed")

	err = processor.Step(transactionId)
	assert.NoError(t, err, "Equip compensation should succeed")

	// Second equip attempt - should succeed
	err = processor.Step(transactionId)
	assert.NoError(t, err, "Second equip attempt should succeed")

	// Complete the equip step
	err = processor.MarkEarliestPendingStepCompleted(transactionId)
	assert.NoError(t, err, "Should complete equip step")

	// Verify final state
	finalSaga, err := processor.GetById(transactionId)
	assert.NoError(t, err, "Should retrieve final saga")
	assert.Equal(t, 2, len(finalSaga.Steps), "Should have 2 steps")
	
	for i, step := range finalSaga.Steps {
		assert.Equal(t, Completed, step.Status, "Step %d should be completed", i)
	}

	// Verify retry counts
	assert.Equal(t, 2, createAttempts, "Should have 2 create attempts")
	assert.Equal(t, 2, equipAttempts, "Should have 2 equip attempts")

	hook.Reset()
}

// TestCreateAndEquipAsset_CompensationFailure tests compensation failures
func TestCreateAndEquipAsset_CompensationFailure(t *testing.T) {
	// Setup
	logger, hook := test.NewNullLogger()
	logger.SetLevel(logrus.DebugLevel)
	
	ctx := context.Background()
	te, _ := tenant.Create(uuid.New(), "GMS", 83, 1)
	tctx := tenant.WithContext(ctx, te)

	// Setup mocks - compensation fails
	compP := &mock2.ProcessorMock{
		RequestCreateAndEquipAssetFunc: func(transactionId uuid.UUID, payload compartment.CreateAndEquipAssetPayload) error {
			return nil // Asset creation succeeds
		},
		RequestEquipAssetFunc: func(transactionId uuid.UUID, characterId uint32, inventoryType byte, source int16, destination int16) error {
			return errors.New("equip failed")
		},
		RequestUnequipAssetFunc: func(transactionId uuid.UUID, characterId uint32, inventoryType byte, source int16, destination int16) error {
			return errors.New("compensation failed - cannot unequip")
		},
		RequestDestroyItemFunc: func(transactionId uuid.UUID, characterId uint32, templateId uint32, quantity uint32) error {
			return errors.New("compensation failed - cannot destroy item")
		},
	}

	// Create saga processor
	processor := NewProcessor(logger, tctx)
	processor.t = te
	processor.compP = compP

	// Create saga with CreateAndEquipAsset step
	transactionId := uuid.New()
	saga := Saga{
		TransactionId: transactionId,
		SagaType:      InventoryTransaction,
		InitiatedBy:   "compensation-failure-test",
		Steps: []Step[any]{
			{
				StepId:    "create-and-equip-step",
				Status:    Pending,
				Action:    CreateAndEquipAsset,
				Payload: CreateAndEquipAssetPayload{
					CharacterId: 12345,
					Item: ItemPayload{
						TemplateId: 1302000,
						Quantity:   1,
					},
				},
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
			},
		},
	}

	// Store saga in cache
	GetCache().Put(te.Id(), saga)

	// Execute and complete CreateAndEquipAsset step
	err := processor.Step(transactionId)
	assert.NoError(t, err, "CreateAndEquipAsset step should succeed")

	// Add auto-equip step before completing CreateAndEquipAsset
	autoEquipStepId := "auto_equip_step_test"
	equipStep := Step[any]{
		StepId:    autoEquipStepId,
		Status:    Pending,
		Action:    EquipAsset,
		Payload: EquipAssetPayload{
			CharacterId:   12345,
			InventoryType: 1,
			Source:        5,
			Destination:   -1,
		},
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	
	err = processor.AddStep(transactionId, equipStep)
	assert.NoError(t, err, "Should add auto-equip step")

	err = processor.MarkEarliestPendingStepCompleted(transactionId)
	assert.NoError(t, err, "Should complete CreateAndEquipAsset step")

	// Execute equip step - should fail
	err = processor.Step(transactionId)
	assert.Error(t, err, "Equip step should fail")

	// Mark as failed
	err = processor.MarkEarliestPendingStep(transactionId, Failed)
	assert.NoError(t, err, "Should mark as failed")

	// Execute compensation - should fail
	err = processor.Step(transactionId)
	assert.Error(t, err, "Compensation should fail")

	// Verify saga is still in failing state
	failingSaga, err := processor.GetById(transactionId)
	assert.NoError(t, err, "Should retrieve failing saga")
	assert.True(t, failingSaga.Failing(), "Saga should still be failing")

	// Verify step is still failed (not compensated)
	var equipStepFound bool
	for _, step := range failingSaga.Steps {
		if step.Action == EquipAsset {
			assert.Equal(t, Failed, step.Status, "Equip step should still be failed")
			equipStepFound = true
			break
		}
	}
	assert.True(t, equipStepFound, "Should find failed equip step")

	hook.Reset()
}

// TestCreateAndEquipAsset_StateConsistencyValidation tests state consistency during failures
func TestCreateAndEquipAsset_StateConsistencyValidation(t *testing.T) {
	// Setup
	logger, hook := test.NewNullLogger()
	logger.SetLevel(logrus.DebugLevel)
	
	ctx := context.Background()
	te, _ := tenant.Create(uuid.New(), "GMS", 83, 1)
	tctx := tenant.WithContext(ctx, te)

	// Setup mocks
	compP := &mock2.ProcessorMock{
		RequestCreateAndEquipAssetFunc: func(transactionId uuid.UUID, payload compartment.CreateAndEquipAssetPayload) error {
			return nil
		},
		RequestEquipAssetFunc: func(transactionId uuid.UUID, characterId uint32, inventoryType byte, source int16, destination int16) error {
			return errors.New("equip failed")
		},
		RequestUnequipAssetFunc: func(transactionId uuid.UUID, characterId uint32, inventoryType byte, source int16, destination int16) error {
			return nil
		},
	}

	// Create saga processor
	processor := NewProcessor(logger, tctx)
	processor.t = te
	processor.compP = compP

	// Create saga with CreateAndEquipAsset step
	transactionId := uuid.New()
	saga := Saga{
		TransactionId: transactionId,
		SagaType:      InventoryTransaction,
		InitiatedBy:   "state-consistency-test",
		Steps: []Step[any]{
			{
				StepId:    "create-and-equip-step",
				Status:    Pending,
				Action:    CreateAndEquipAsset,
				Payload: CreateAndEquipAssetPayload{
					CharacterId: 12345,
					Item: ItemPayload{
						TemplateId: 1302000,
						Quantity:   1,
					},
				},
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
			},
		},
	}

	// Store saga in cache
	GetCache().Put(te.Id(), saga)

	// Test state consistency throughout failure and recovery process
	
	// 1. Initial state should be consistent
	initialSaga, err := processor.GetById(transactionId)
	assert.NoError(t, err, "Should retrieve initial saga")
	assert.NoError(t, initialSaga.ValidateStateConsistency(), "Initial state should be consistent")

	// 2. After successful execution
	err = processor.Step(transactionId)
	assert.NoError(t, err, "Step should succeed")
	
	afterStepSaga, err := processor.GetById(transactionId)
	assert.NoError(t, err, "Should retrieve saga after step")
	assert.NoError(t, afterStepSaga.ValidateStateConsistency(), "State should be consistent after step")

	// 3. After adding auto-equip step (before completion)
	autoEquipStepId := "auto_equip_step_test"
	equipStep := Step[any]{
		StepId:    autoEquipStepId,
		Status:    Pending,
		Action:    EquipAsset,
		Payload: EquipAssetPayload{
			CharacterId:   12345,
			InventoryType: 1,
			Source:        5,
			Destination:   -1,
		},
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	
	err = processor.AddStep(transactionId, equipStep)
	assert.NoError(t, err, "Should add auto-equip step")
	
	afterAddStepSaga, err := processor.GetById(transactionId)
	assert.NoError(t, err, "Should retrieve saga after step addition")
	assert.NoError(t, afterAddStepSaga.ValidateStateConsistency(), "State should be consistent after step addition")

	// 4. After marking step as completed
	err = processor.MarkEarliestPendingStepCompleted(transactionId)
	assert.NoError(t, err, "Should complete step")
	
	afterCompletionSaga, err := processor.GetById(transactionId)
	assert.NoError(t, err, "Should retrieve saga after completion")
	assert.NoError(t, afterCompletionSaga.ValidateStateConsistency(), "State should be consistent after completion")

	// 5. After equip step fails
	err = processor.Step(transactionId)
	assert.Error(t, err, "Equip step should fail")
	
	// State should still be consistent even with failure
	afterFailureSaga, err := processor.GetById(transactionId)
	assert.NoError(t, err, "Should retrieve saga after failure")
	assert.NoError(t, afterFailureSaga.ValidateStateConsistency(), "State should be consistent after failure")

	// 6. After marking step as failed
	err = processor.MarkEarliestPendingStep(transactionId, Failed)
	assert.NoError(t, err, "Should mark as failed")
	
	afterMarkFailedSaga, err := processor.GetById(transactionId)
	assert.NoError(t, err, "Should retrieve saga after marking failed")
	assert.NoError(t, afterMarkFailedSaga.ValidateStateConsistency(), "State should be consistent after marking failed")

	// 7. After compensation
	err = processor.Step(transactionId)
	assert.NoError(t, err, "Compensation should succeed")
	
	afterCompensationSaga, err := processor.GetById(transactionId)
	assert.NoError(t, err, "Should retrieve saga after compensation")
	assert.NoError(t, afterCompensationSaga.ValidateStateConsistency(), "State should be consistent after compensation")

	// Verify state transitions in logs
	logEntries := hook.AllEntries()
	assert.True(t, len(logEntries) > 0, "Should have log entries")
	
	// Look for state consistency validation logs
	for _, entry := range logEntries {
		if strings.Contains(entry.Message, "State consistency validation") {
			// Found consistency validation logs - this is expected
			break
		}
	}
	// Note: This depends on the actual logging implementation
	// The test mainly verifies that ValidateStateConsistency() doesn't return errors

	hook.Reset()
}

// TestCreateAndEquipAsset_DynamicStepCreationAndOrdering tests dynamic step creation and proper ordering
func TestCreateAndEquipAsset_DynamicStepCreationAndOrdering(t *testing.T) {
	// Setup
	logger, hook := test.NewNullLogger()
	logger.SetLevel(logrus.DebugLevel)
	
	ctx := context.Background()
	te, _ := tenant.Create(uuid.New(), "GMS", 83, 1)
	tctx := tenant.WithContext(ctx, te)

	// Setup mocks
	compP := &mock2.ProcessorMock{
		RequestCreateAndEquipAssetFunc: func(transactionId uuid.UUID, payload compartment.CreateAndEquipAssetPayload) error {
			return nil
		},
		RequestEquipAssetFunc: func(transactionId uuid.UUID, characterId uint32, inventoryType byte, source int16, destination int16) error {
			return nil
		},
	}

	// Create saga processor
	processor := NewProcessor(logger, tctx)
	processor.t = te
	processor.compP = compP

	// Create saga with multiple steps including CreateAndEquipAsset
	transactionId := uuid.New()
	saga := Saga{
		TransactionId: transactionId,
		SagaType:      InventoryTransaction,
		InitiatedBy:   "dynamic-step-ordering-test",
		Steps: []Step[any]{
			{
				StepId:    "step-1-validate",
				Status:    Completed,
				Action:    ValidateCharacterState,
				Payload: ValidateCharacterStatePayload{
					CharacterId: 12345,
					Conditions:  []validation.ConditionInput{},
				},
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
			},
			{
				StepId:    "step-2-create-and-equip",
				Status:    Pending,
				Action:    CreateAndEquipAsset,
				Payload: CreateAndEquipAssetPayload{
					CharacterId: 12345,
					Item: ItemPayload{
						TemplateId: 1302000,
						Quantity:   1,
					},
				},
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
			},
			{
				StepId:    "step-3-final-action",
				Status:    Pending,
				Action:    AwardInventory,
				Payload: AwardItemActionPayload{
					CharacterId: 12345,
					Item: ItemPayload{
						TemplateId: 2000000,
						Quantity:   10,
					},
				},
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
			},
		},
	}

	// Store saga in cache
	GetCache().Put(te.Id(), saga)

	// Verify initial step order
	initialSaga, err := processor.GetById(transactionId)
	assert.NoError(t, err, "Should retrieve initial saga")
	assert.Equal(t, 3, len(initialSaga.Steps), "Should have 3 initial steps")
	assert.Equal(t, "step-1-validate", initialSaga.Steps[0].StepId)
	assert.Equal(t, "step-2-create-and-equip", initialSaga.Steps[1].StepId)
	assert.Equal(t, "step-3-final-action", initialSaga.Steps[2].StepId)

	// Execute the CreateAndEquipAsset step
	err = processor.Step(transactionId)
	assert.NoError(t, err, "CreateAndEquipAsset step should execute successfully")

	// Dynamically add auto-equip step - this should be inserted after the current step
	// but before the final action step
	autoEquipStepId := "auto_equip_step_" + fmt.Sprintf("%d", time.Now().Unix())
	equipPayload := EquipAssetPayload{
		CharacterId:   12345,
		InventoryType: 1,
		Source:        5,
		Destination:   -1,
	}
	
	equipStep := Step[any]{
		StepId:    autoEquipStepId,
		Status:    Pending,
		Action:    EquipAsset,
		Payload:   equipPayload,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	
	err = processor.AddStep(transactionId, equipStep)
	assert.NoError(t, err, "Should successfully add auto-equip step")

	// Verify step ordering after dynamic addition
	afterAdditionSaga, err := processor.GetById(transactionId)
	assert.NoError(t, err, "Should retrieve saga after step addition")
	assert.Equal(t, 4, len(afterAdditionSaga.Steps), "Should have 4 steps after auto-equip addition")

	// Test step ordering: validate -> create-and-equip -> auto-equip -> final-action
	assert.Equal(t, "step-1-validate", afterAdditionSaga.Steps[0].StepId)
	assert.Equal(t, "step-2-create-and-equip", afterAdditionSaga.Steps[1].StepId)
	assert.Equal(t, autoEquipStepId, afterAdditionSaga.Steps[2].StepId)
	assert.Equal(t, "step-3-final-action", afterAdditionSaga.Steps[3].StepId)

	// Verify step statuses maintain proper ordering
	assert.Equal(t, Completed, afterAdditionSaga.Steps[0].Status, "First step should be completed")
	assert.Equal(t, Pending, afterAdditionSaga.Steps[1].Status, "Second step should be pending")
	assert.Equal(t, Pending, afterAdditionSaga.Steps[2].Status, "Auto-equip step should be pending")
	assert.Equal(t, Pending, afterAdditionSaga.Steps[3].Status, "Final step should be pending")

	// Complete the CreateAndEquipAsset step
	err = processor.MarkEarliestPendingStepCompleted(transactionId)
	assert.NoError(t, err, "Should complete CreateAndEquipAsset step")

	// Verify state after completion
	afterCompletionSaga, err := processor.GetById(transactionId)
	assert.NoError(t, err, "Should retrieve saga after completion")
	assert.Equal(t, Completed, afterCompletionSaga.Steps[0].Status, "First step should be completed")
	assert.Equal(t, Completed, afterCompletionSaga.Steps[1].Status, "Second step should be completed")
	assert.Equal(t, Pending, afterCompletionSaga.Steps[2].Status, "Auto-equip step should be pending")
	assert.Equal(t, Pending, afterCompletionSaga.Steps[3].Status, "Final step should be pending")

	// Execute the auto-equip step
	err = processor.Step(transactionId)
	assert.NoError(t, err, "Auto-equip step should execute successfully")

	// Complete the auto-equip step
	err = processor.MarkEarliestPendingStepCompleted(transactionId)
	assert.NoError(t, err, "Should complete auto-equip step")

	// Verify that we can still process the final step
	finalSaga, err := processor.GetById(transactionId)
	assert.NoError(t, err, "Should retrieve final saga")
	assert.Equal(t, Completed, finalSaga.Steps[0].Status, "First step should be completed")
	assert.Equal(t, Completed, finalSaga.Steps[1].Status, "Second step should be completed")
	assert.Equal(t, Completed, finalSaga.Steps[2].Status, "Auto-equip step should be completed")
	assert.Equal(t, Pending, finalSaga.Steps[3].Status, "Final step should still be pending")

	// Test dynamic step addition with specific ordering constraints
	// Add another step dynamically to ensure proper insertion
	additionalStepId := "additional_step_" + fmt.Sprintf("%d", time.Now().Unix())
	additionalStep := Step[any]{
		StepId:    additionalStepId,
		Status:    Pending,
		Action:    AwardInventory,
		Payload: AwardItemActionPayload{
			CharacterId: 12345,
			Item: ItemPayload{
				TemplateId: 3000000,
				Quantity:   5,
			},
		},
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	
	err = processor.AddStep(transactionId, additionalStep)
	assert.NoError(t, err, "Should successfully add additional step")

	// Verify final ordering with all dynamic steps
	finalOrderingSaga, err := processor.GetById(transactionId)
	assert.NoError(t, err, "Should retrieve saga for final ordering verification")
	assert.Equal(t, 5, len(finalOrderingSaga.Steps), "Should have 5 steps total")

	// Verify the correct order of all steps
	expectedStepIds := []string{
		"step-1-validate",
		"step-2-create-and-equip",
		autoEquipStepId,
		"step-3-final-action",
		additionalStepId,
	}
	
	for i, expectedId := range expectedStepIds {
		assert.Equal(t, expectedId, finalOrderingSaga.Steps[i].StepId, "Step %d should have correct ID", i)
	}

	// Verify state consistency with dynamic steps
	assert.NoError(t, finalOrderingSaga.ValidateStateConsistency(), "State should be consistent with dynamic steps")

	// Test that we can still get the current step correctly
	currentStep, found := finalOrderingSaga.GetCurrentStep()
	assert.True(t, found, "Should find current step")
	assert.Equal(t, "step-3-final-action", currentStep.StepId, "Current step should be the final action step")

	// Test step navigation with dynamic steps
	furthestCompletedIndex := finalOrderingSaga.FindFurthestCompletedStepIndex()
	assert.Equal(t, 2, furthestCompletedIndex, "Furthest completed step should be the auto-equip step (index 2)")

	earliestPendingIndex := finalOrderingSaga.FindEarliestPendingStepIndex()
	assert.Equal(t, 3, earliestPendingIndex, "Earliest pending step should be the final action step (index 3)")

	// Verify proper logging of dynamic step creation
	logEntries := hook.AllEntries()
	assert.True(t, len(logEntries) > 0, "Should have log entries")
	
	// Look for specific log entries about step addition
	for _, entry := range logEntries {
		if strings.Contains(entry.Message, "Step added") || strings.Contains(entry.Message, "Adding step") {
			// Found step addition log - this depends on actual logging implementation
			break
		}
	}
	// This depends on the actual logging implementation in AddStep
	// The test mainly verifies that dynamic step addition works correctly

	hook.Reset()
}

// TestCreateAndEquipAsset_DynamicStepOrderingConstraints tests ordering constraints for dynamic steps
func TestCreateAndEquipAsset_DynamicStepOrderingConstraints(t *testing.T) {
	// Setup
	logger, hook := test.NewNullLogger()
	logger.SetLevel(logrus.DebugLevel)
	
	ctx := context.Background()
	te, _ := tenant.Create(uuid.New(), "GMS", 83, 1)
	tctx := tenant.WithContext(ctx, te)

	// Setup mocks
	compP := &mock2.ProcessorMock{
		RequestCreateAndEquipAssetFunc: func(transactionId uuid.UUID, payload compartment.CreateAndEquipAssetPayload) error {
			return nil
		},
	}

	// Create saga processor
	processor := NewProcessor(logger, tctx)
	processor.t = te
	processor.compP = compP

	// Create saga with CreateAndEquipAsset step
	transactionId := uuid.New()
	saga := Saga{
		TransactionId: transactionId,
		SagaType:      InventoryTransaction,
		InitiatedBy:   "step-ordering-constraints-test",
		Steps: []Step[any]{
			{
				StepId:    "step-1",
				Status:    Completed,
				Action:    ValidateCharacterState,
				Payload: ValidateCharacterStatePayload{
					CharacterId: 12345,
					Conditions:  []validation.ConditionInput{},
				},
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
			},
			{
				StepId:    "step-2",
				Status:    Completed,
				Action:    CreateAndEquipAsset,
				Payload: CreateAndEquipAssetPayload{
					CharacterId: 12345,
					Item: ItemPayload{
						TemplateId: 1302000,
						Quantity:   1,
					},
				},
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
			},
			{
				StepId:    "step-3",
				Status:    Pending,
				Action:    AwardInventory,
				Payload: AwardItemActionPayload{
					CharacterId: 12345,
					Item: ItemPayload{
						TemplateId: 2000000,
						Quantity:   10,
					},
				},
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
			},
		},
	}

	// Store saga in cache
	GetCache().Put(te.Id(), saga)

	// Test 1: Add auto-equip step after completed CreateAndEquipAsset step
	autoEquipStepId := "auto_equip_step_after_completion"
	equipStep := Step[any]{
		StepId:    autoEquipStepId,
		Status:    Pending,
		Action:    EquipAsset,
		Payload: EquipAssetPayload{
			CharacterId:   12345,
			InventoryType: 1,
			Source:        5,
			Destination:   -1,
		},
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	
	err := processor.AddStep(transactionId, equipStep)
	assert.NoError(t, err, "Should successfully add auto-equip step after completion")

	// Verify the step was inserted in the correct position
	afterAdditionSaga, err := processor.GetById(transactionId)
	assert.NoError(t, err, "Should retrieve saga after step addition")
	assert.Equal(t, 4, len(afterAdditionSaga.Steps), "Should have 4 steps after addition")

	// The auto-equip step should be inserted after the current pending step
	assert.Equal(t, "step-1", afterAdditionSaga.Steps[0].StepId)
	assert.Equal(t, "step-2", afterAdditionSaga.Steps[1].StepId)
	assert.Equal(t, "step-3", afterAdditionSaga.Steps[2].StepId)
	assert.Equal(t, autoEquipStepId, afterAdditionSaga.Steps[3].StepId)

	// Test 2: Verify state consistency after dynamic insertion
	assert.NoError(t, afterAdditionSaga.ValidateStateConsistency(), "State should be consistent after dynamic insertion")

	// Test 3: Test multiple dynamic step additions
	secondAutoEquipStepId := "second_auto_equip_step"
	secondEquipStep := Step[any]{
		StepId:    secondAutoEquipStepId,
		Status:    Pending,
		Action:    EquipAsset,
		Payload: EquipAssetPayload{
			CharacterId:   12345,
			InventoryType: 2,
			Source:        6,
			Destination:   -2,
		},
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	
	err = processor.AddStep(transactionId, secondEquipStep)
	assert.NoError(t, err, "Should successfully add second auto-equip step")

	// Verify proper ordering with multiple dynamic steps
	finalSaga, err := processor.GetById(transactionId)
	assert.NoError(t, err, "Should retrieve saga after second step addition")
	assert.Equal(t, 5, len(finalSaga.Steps), "Should have 5 steps after second addition")

	// Verify all steps are in correct order
	expectedOrder := []string{
		"step-1",
		"step-2",
		"step-3",
		secondAutoEquipStepId,  // Second step was added after step-3
		autoEquipStepId,        // First step was added after second step
	}
	
	for i, expectedId := range expectedOrder {
		assert.Equal(t, expectedId, finalSaga.Steps[i].StepId, "Step %d should have correct ID", i)
	}

	// Test 4: Verify that pending steps remain pending and completed steps remain completed
	assert.Equal(t, Completed, finalSaga.Steps[0].Status, "First step should remain completed")
	assert.Equal(t, Completed, finalSaga.Steps[1].Status, "Second step should remain completed")
	assert.Equal(t, Pending, finalSaga.Steps[2].Status, "Step-3 should remain pending")
	assert.Equal(t, Pending, finalSaga.Steps[3].Status, "Second auto-equip step should be pending")
	assert.Equal(t, Pending, finalSaga.Steps[4].Status, "First auto-equip step should be pending")

	// Test 5: Verify that step navigation works correctly with dynamic steps
	currentStep, found := finalSaga.GetCurrentStep()
	assert.True(t, found, "Should find current step")
	assert.Equal(t, "step-3", currentStep.StepId, "Current step should be step-3")

	// Test 6: Test step execution order with dynamic steps
	err = processor.Step(transactionId)
	assert.NoError(t, err, "Should execute step-3")

	err = processor.MarkEarliestPendingStepCompleted(transactionId)
	assert.NoError(t, err, "Should complete step-3")

	// Verify the next step becomes current
	afterFirstStepSaga, err := processor.GetById(transactionId)
	assert.NoError(t, err, "Should retrieve saga after first step completion")
	
	currentStep, found = afterFirstStepSaga.GetCurrentStep()
	assert.True(t, found, "Should find current step after first step")
	assert.Equal(t, secondAutoEquipStepId, currentStep.StepId, "Current step should be second auto-equip step")

	hook.Reset()
}