package e2e

import (
	"atlas-saga-orchestrator/character"
	"atlas-saga-orchestrator/character/mock"
	"atlas-saga-orchestrator/compartment"
	mock2 "atlas-saga-orchestrator/compartment/mock"
	compartmentConsumer "atlas-saga-orchestrator/kafka/consumer/compartment"
	compartmentMessage "atlas-saga-orchestrator/kafka/message/compartment"
	"atlas-saga-orchestrator/saga"
	"atlas-saga-orchestrator/validation"
	mock3 "atlas-saga-orchestrator/validation/mock"
	"context"
	"errors"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/Chronicle20/atlas-tenant"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"github.com/sirupsen/logrus/hooks/test"
	"github.com/stretchr/testify/assert"
)

// E2ETestEnvironment provides a controlled environment for end-to-end testing
type E2ETestEnvironment struct {
	logger        logrus.FieldLogger
	ctx           context.Context
	tenant        tenant.Tenant
	sagaProcessor *saga.ProcessorImpl
	charP         *mock.ProcessorMock
	compP         *mock2.ProcessorMock
	validP        *mock3.ProcessorMock
	eventChannel  chan interface{}
	wg            sync.WaitGroup
}

// setupE2EEnvironment creates a complete test environment
func setupE2EEnvironment(t *testing.T) *E2ETestEnvironment {
	logger, _ := test.NewNullLogger()
	logger.SetLevel(logrus.DebugLevel)

	ctx := context.Background()
	te, err := tenant.Create(uuid.New(), "GMS", 83, 1)
	assert.NoError(t, err)
	tctx := tenant.WithContext(ctx, te)

	// Create mock processors
	charP := &mock.ProcessorMock{}
	compP := &mock2.ProcessorMock{}
	validP := &mock3.ProcessorMock{}

	// Setup saga processor
	sagaProcessor := saga.NewProcessor(logger, tctx)
	sagaProcessor.SetTenant(te)
	sagaProcessor.SetCharacterProcessor(charP)
	sagaProcessor.SetCompartmentProcessor(compP)
	sagaProcessor.SetValidationProcessor(validP)

	return &E2ETestEnvironment{
		logger:        logger,
		ctx:           tctx,
		tenant:        te,
		sagaProcessor: sagaProcessor,
		charP:         charP,
		compP:         compP,
		validP:        validP,
		eventChannel:  make(chan interface{}, 100),
	}
}

// simulateKafkaEvent simulates a Kafka event being processed
func (env *E2ETestEnvironment) simulateKafkaEvent(event interface{}) {
	env.eventChannel <- event
}

// processKafkaEvents processes events from the channel
func (env *E2ETestEnvironment) processKafkaEvents() {
	for event := range env.eventChannel {
		switch e := event.(type) {
		case compartmentMessage.StatusEvent[compartmentMessage.CreatedStatusEventBody]:
			compartmentConsumer.HandleCompartmentCreatedEvent(env.logger, env.ctx, e)
		case compartmentMessage.StatusEvent[compartmentMessage.CreationFailedStatusEventBody]:
			compartmentConsumer.HandleCompartmentCreationFailedEvent(env.logger, env.ctx, e)
		case compartmentMessage.StatusEvent[compartmentMessage.EquippedEventBody]:
			compartmentConsumer.HandleCompartmentEquippedEvent(env.logger, env.ctx, e)
		case compartmentMessage.StatusEvent[compartmentMessage.ErrorEventBody]:
			compartmentConsumer.HandleCompartmentErrorEvent(env.logger, env.ctx, e)
		}
	}
}

// cleanup closes the event channel and waits for goroutines to finish
func (env *E2ETestEnvironment) cleanup() {
	close(env.eventChannel)
	env.wg.Wait()
}

// TestCreateAndEquipAsset_E2E_SuccessFlow tests the complete success flow
func TestCreateAndEquipAsset_E2E_SuccessFlow(t *testing.T) {
	env := setupE2EEnvironment(t)
	defer env.cleanup()

	// Start event processing
	env.wg.Add(1)
	go func() {
		defer env.wg.Done()
		env.processKafkaEvents()
	}()

	transactionId := uuid.New()
	characterId := uint32(12345)
	templateId := uint32(1302000)
	assetId := uint32(67890)

	// Setup compartment processor mock
	env.compP.RequestCreateAndEquipAssetFunc = func(txnId uuid.UUID, payload compartment.CreateAndEquipAssetPayload) error {
		assert.Equal(t, transactionId, txnId)
		assert.Equal(t, characterId, payload.CharacterId)
		assert.Equal(t, templateId, payload.Item.TemplateId)
		assert.Equal(t, uint32(1), payload.Item.Quantity)

		// Simulate successful asset creation - this would trigger a CREATED event
		go func() {
			time.Sleep(10 * time.Millisecond) // Simulate processing time
			env.simulateKafkaEvent(compartmentMessage.StatusEvent[compartmentMessage.CreatedStatusEventBody]{
				TransactionId: transactionId,
				Type:          compartmentMessage.StatusEventTypeCreated,
				CharacterId:   characterId,
				Body: compartmentMessage.CreatedStatusEventBody{
					Type:     1,
					Capacity: 100,
				},
			})
		}()

		return nil
	}

	// Setup equipment processor mock for the auto-generated equip step
	env.compP.RequestEquipAssetFunc = func(txnId uuid.UUID, charId uint32, invType byte, source int16, dest int16) error {
		assert.Equal(t, transactionId, txnId)
		assert.Equal(t, characterId, charId)
		assert.Equal(t, byte(1), invType)
		assert.Equal(t, int16(5), source)
		assert.Equal(t, int16(-1), dest)

		// Simulate successful equipment - this would trigger an EQUIPPED event
		go func() {
			time.Sleep(10 * time.Millisecond) // Simulate processing time
			env.simulateKafkaEvent(compartmentMessage.StatusEvent[compartmentMessage.EquippedEventBody]{
				TransactionId: transactionId,
				Type:          compartmentMessage.StatusEventTypeEquipped,
				CharacterId:   characterId,
				Body: compartmentMessage.EquippedEventBody{
					Source:      5,
					Destination: -1,
				},
			})
		}()

		return nil
	}

	// Create initial saga
	initialSaga := saga.Saga{
		TransactionId: transactionId,
		SagaType:      saga.InventoryTransaction,
		InitiatedBy:   "e2e-test",
		Steps: []saga.Step[any]{
			{
				StepId: "create-and-equip-step",
				Status: saga.Pending,
				Action: saga.CreateAndEquipAsset,
				Payload: saga.CreateAndEquipAssetPayload{
					CharacterId: characterId,
					Item: saga.ItemPayload{
						TemplateId: templateId,
						Quantity:   1,
					},
				},
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
			},
		},
	}

	// Store saga in cache
	saga.GetCache().Put(env.tenant.Id(), initialSaga)

	// Phase 1: Execute CreateAndEquipAsset step
	err := env.sagaProcessor.Step(transactionId)
	assert.NoError(t, err, "CreateAndEquipAsset step should succeed")

	// Wait for CREATED event processing
	time.Sleep(50 * time.Millisecond)

	// Phase 2: Complete the CreateAndEquipAsset step (simulating CREATED event)
	err = env.sagaProcessor.StepCompleted(transactionId, true)
	assert.NoError(t, err, "CreateAndEquipAsset step completion should succeed")

	// Verify that an auto-equip step was added
	updatedSaga, err := env.sagaProcessor.GetById(transactionId)
	assert.NoError(t, err, "Should be able to retrieve updated saga")
	assert.Equal(t, 2, len(updatedSaga.Steps), "Should have 2 steps after auto-equip step addition")

	// Verify the auto-equip step
	equipStep := updatedSaga.Steps[1]
	assert.Equal(t, saga.EquipAsset, equipStep.Action, "Second step should be EquipAsset")
	assert.Equal(t, saga.Pending, equipStep.Status, "Auto-equip step should be pending")

	equipPayload, ok := equipStep.Payload.(saga.EquipAssetPayload)
	assert.True(t, ok, "Auto-equip step should have EquipAssetPayload")
	assert.Equal(t, characterId, equipPayload.CharacterId, "Character ID should match")
	assert.Equal(t, uint32(1), equipPayload.InventoryType, "Inventory type should match")
	assert.Equal(t, int16(5), equipPayload.Source, "Source slot should be 5")
	assert.Equal(t, int16(-1), equipPayload.Destination, "Destination slot should be -1")

	// Phase 3: Execute the auto-equip step
	err = env.sagaProcessor.Step(transactionId)
	assert.NoError(t, err, "EquipAsset step should succeed")

	// Wait for EQUIPPED event processing
	time.Sleep(50 * time.Millisecond)

	// Phase 4: Complete the EquipAsset step (simulating EQUIPPED event)
	err = env.sagaProcessor.StepCompleted(transactionId, true)
	assert.NoError(t, err, "EquipAsset step completion should succeed")

	// Verify final saga state
	finalSaga, err := env.sagaProcessor.GetById(transactionId)
	assert.NoError(t, err, "Should be able to retrieve final saga")
	assert.Equal(t, 2, len(finalSaga.Steps), "Should still have 2 steps")

	// Verify both steps are completed
	for i, step := range finalSaga.Steps {
		assert.Equal(t, saga.Completed, step.Status, "Step %d should be completed", i)
	}

	// Verify mock calls
	assert.NotNil(t, env.compP.RequestCreateAndEquipAssetFunc, "RequestCreateAndEquipAsset should have been called")
	assert.NotNil(t, env.compP.RequestEquipAssetFunc, "RequestEquipAsset should have been called")
}

// TestCreateAndEquipAsset_E2E_CreationFailure tests the creation failure flow
func TestCreateAndEquipAsset_E2E_CreationFailure(t *testing.T) {
	env := setupE2EEnvironment(t)
	defer env.cleanup()

	// Start event processing
	env.wg.Add(1)
	go func() {
		defer env.wg.Done()
		env.processKafkaEvents()
	}()

	transactionId := uuid.New()
	characterId := uint32(12345)
	templateId := uint32(0) // Invalid template ID

	// Setup compartment processor mock to fail
	env.compP.RequestCreateAndEquipAssetFunc = func(txnId uuid.UUID, payload compartment.CreateAndEquipAssetPayload) error {
		assert.Equal(t, transactionId, txnId)
		assert.Equal(t, characterId, payload.CharacterId)
		assert.Equal(t, templateId, payload.Item.TemplateId)

		// Simulate asset creation failure - this would trigger a CREATION_FAILED event
		go func() {
			time.Sleep(10 * time.Millisecond) // Simulate processing time
			env.simulateKafkaEvent(compartmentMessage.StatusEvent[compartmentMessage.CreationFailedStatusEventBody]{
				TransactionId: transactionId,
				Type:          compartmentMessage.StatusEventTypeCreationFailed,
				CharacterId:   characterId,
				Body: compartmentMessage.CreationFailedStatusEventBody{
					ErrorCode: "INVALID_TEMPLATE_ID",
					Message:   "Template ID 0 is not valid",
				},
			})
		}()

		return errors.New("invalid templateId")
	}

	// Create initial saga
	initialSaga := saga.Saga{
		TransactionId: transactionId,
		SagaType:      saga.InventoryTransaction,
		InitiatedBy:   "e2e-test-failure",
		Steps: []saga.Step[any]{
			{
				StepId: "create-and-equip-step",
				Status: saga.Pending,
				Action: saga.CreateAndEquipAsset,
				Payload: saga.CreateAndEquipAssetPayload{
					CharacterId: characterId,
					Item: saga.ItemPayload{
						TemplateId: templateId,
						Quantity:   1,
					},
				},
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
			},
		},
	}

	// Store saga in cache
	saga.GetCache().Put(env.tenant.Id(), initialSaga)

	// Phase 1: Execute CreateAndEquipAsset step (should fail)
	err := env.sagaProcessor.Step(transactionId)
	assert.Error(t, err, "CreateAndEquipAsset step should fail")
	assert.Contains(t, err.Error(), "invalid templateId", "Error should contain templateId message")

	// Wait for CREATION_FAILED event processing
	time.Sleep(50 * time.Millisecond)

	// Phase 2: Complete the CreateAndEquipAsset step with failure (simulating CREATION_FAILED event)
	err = env.sagaProcessor.StepCompleted(transactionId, false)
	assert.NoError(t, err, "StepCompleted should succeed even for failed steps")

	// Verify final saga state
	finalSaga, err := env.sagaProcessor.GetById(transactionId)
	assert.NoError(t, err, "Should be able to retrieve final saga")
	assert.Equal(t, 1, len(finalSaga.Steps), "Should still have only 1 step")

	// Verify the step failed
	assert.Equal(t, saga.Failed, finalSaga.Steps[0].Status, "Step should be marked as failed")

	// Verify no auto-equip step was added
	for _, step := range finalSaga.Steps {
		assert.NotEqual(t, saga.EquipAsset, step.Action, "Should not have auto-equip step on creation failure")
	}

	// Verify mock calls
	assert.NotNil(t, env.compP.RequestCreateAndEquipAssetFunc, "RequestCreateAndEquipAsset should have been called")
	assert.Nil(t, env.compP.RequestEquipAssetFunc, "RequestEquipAsset should not have been called")
}

// TestCreateAndEquipAsset_E2E_EquipmentFailure tests the equipment failure flow
func TestCreateAndEquipAsset_E2E_EquipmentFailure(t *testing.T) {
	env := setupE2EEnvironment(t)
	defer env.cleanup()

	// Start event processing
	env.wg.Add(1)
	go func() {
		defer env.wg.Done()
		env.processKafkaEvents()
	}()

	transactionId := uuid.New()
	characterId := uint32(12345)
	templateId := uint32(1302000)
	assetId := uint32(67890)

	// Setup compartment processor mock - creation succeeds
	env.compP.RequestCreateAndEquipAssetFunc = func(txnId uuid.UUID, payload compartment.CreateAndEquipAssetPayload) error {
		assert.Equal(t, transactionId, txnId)

		// Simulate successful asset creation
		go func() {
			time.Sleep(10 * time.Millisecond)
			env.simulateKafkaEvent(compartmentMessage.StatusEvent[compartmentMessage.CreatedStatusEventBody]{
				TransactionId: transactionId,
				Type:          compartmentMessage.StatusEventTypeCreated,
				CharacterId:   characterId,
				Body: compartmentMessage.CreatedStatusEventBody{
					Type:     1,
					Capacity: 100,
				},
			})
		}()

		return nil
	}

	// Setup equipment processor mock - equipment fails
	env.compP.RequestEquipAssetFunc = func(txnId uuid.UUID, charId uint32, invType byte, source int16, dest int16) error {
		assert.Equal(t, transactionId, txnId)

		// Simulate equipment failure
		go func() {
			time.Sleep(10 * time.Millisecond)
			env.simulateKafkaEvent(compartmentMessage.StatusEvent[compartmentMessage.ErrorEventBody]{
				TransactionId: transactionId,
				Type:          compartmentMessage.StatusEventTypeError,
				CharacterId:   characterId,
				Body: compartmentMessage.ErrorEventBody{
					ErrorCode: "EQUIPMENT_SLOT_OCCUPIED",
				},
			})
		}()

		return errors.New("equipment slot occupied")
	}

	// Create initial saga
	initialSaga := saga.Saga{
		TransactionId: transactionId,
		SagaType:      saga.InventoryTransaction,
		InitiatedBy:   "e2e-test-equip-failure",
		Steps: []saga.Step[any]{
			{
				StepId: "create-and-equip-step",
				Status: saga.Pending,
				Action: saga.CreateAndEquipAsset,
				Payload: saga.CreateAndEquipAssetPayload{
					CharacterId: characterId,
					Item: saga.ItemPayload{
						TemplateId: templateId,
						Quantity:   1,
					},
				},
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
			},
		},
	}

	// Store saga in cache
	saga.GetCache().Put(env.tenant.Id(), initialSaga)

	// Phase 1: Execute CreateAndEquipAsset step (should succeed)
	err := env.sagaProcessor.Step(transactionId)
	assert.NoError(t, err, "CreateAndEquipAsset step should succeed")

	// Wait for CREATED event processing
	time.Sleep(50 * time.Millisecond)

	// Phase 2: Complete the CreateAndEquipAsset step (simulating CREATED event)
	err = env.sagaProcessor.StepCompleted(transactionId, true)
	assert.NoError(t, err, "CreateAndEquipAsset step completion should succeed")

	// Verify auto-equip step was added
	updatedSaga, err := env.sagaProcessor.GetById(transactionId)
	assert.NoError(t, err, "Should be able to retrieve updated saga")
	assert.Equal(t, 2, len(updatedSaga.Steps), "Should have 2 steps after auto-equip step addition")

	// Phase 3: Execute the auto-equip step (should fail)
	err = env.sagaProcessor.Step(transactionId)
	assert.Error(t, err, "EquipAsset step should fail")
	assert.Contains(t, err.Error(), "equipment slot occupied", "Error should contain equipment message")

	// Wait for ERROR event processing
	time.Sleep(50 * time.Millisecond)

	// Phase 4: Complete the EquipAsset step with failure (simulating ERROR event)
	err = env.sagaProcessor.StepCompleted(transactionId, false)
	assert.NoError(t, err, "StepCompleted should succeed even for failed steps")

	// Verify final saga state
	finalSaga, err := env.sagaProcessor.GetById(transactionId)
	assert.NoError(t, err, "Should be able to retrieve final saga")
	assert.Equal(t, 2, len(finalSaga.Steps), "Should have 2 steps")

	// Verify step statuses
	assert.Equal(t, saga.Completed, finalSaga.Steps[0].Status, "CreateAndEquipAsset step should be completed")
	assert.Equal(t, saga.Failed, finalSaga.Steps[1].Status, "EquipAsset step should be failed")

	// Verify the saga is in a failing state
	assert.True(t, finalSaga.Failing(), "Saga should be in failing state")

	// Verify both mock calls were made
	assert.NotNil(t, env.compP.RequestCreateAndEquipAssetFunc, "RequestCreateAndEquipAsset should have been called")
	assert.NotNil(t, env.compP.RequestEquipAssetFunc, "RequestEquipAsset should have been called")
}

// TestCreateAndEquipAsset_E2E_ConcurrentExecution tests concurrent execution scenarios
func TestCreateAndEquipAsset_E2E_ConcurrentExecution(t *testing.T) {
	env := setupE2EEnvironment(t)
	defer env.cleanup()

	// Start event processing
	env.wg.Add(1)
	go func() {
		defer env.wg.Done()
		env.processKafkaEvents()
	}()

	// Create multiple sagas to test concurrent execution
	numSagas := 5
	transactionIds := make([]uuid.UUID, numSagas)
	for i := 0; i < numSagas; i++ {
		transactionIds[i] = uuid.New()
	}

	// Track completion
	completedSagas := make(map[uuid.UUID]bool)
	var mutex sync.Mutex

	// Setup compartment processor mock
	env.compP.RequestCreateAndEquipAssetFunc = func(txnId uuid.UUID, payload compartment.CreateAndEquipAssetPayload) error {
		// Simulate successful asset creation for all
		go func() {
			time.Sleep(time.Duration(10+txnId.ID()%20) * time.Millisecond) // Random delay
			env.simulateKafkaEvent(compartmentMessage.StatusEvent[compartmentMessage.CreatedStatusEventBody]{
				TransactionId: txnId,
				Type:          compartmentMessage.StatusEventTypeCreated,
				CharacterId:   payload.CharacterId,
				Body: compartmentMessage.CreatedStatusEventBody{
					Type:     1,
					AssetId:  uint32(67890 + txnId.ID()%1000),
					Quantity: 1,
				},
			})
		}()
		return nil
	}

	env.compP.RequestEquipAssetFunc = func(txnId uuid.UUID, charId uint32, invType byte, source int16, dest int16) error {
		// Simulate successful equipment for all
		go func() {
			time.Sleep(time.Duration(10+txnId.ID()%20) * time.Millisecond) // Random delay
			env.simulateKafkaEvent(compartmentMessage.StatusEvent[compartmentMessage.EquippedEventBody]{
				TransactionId: txnId,
				Type:          compartmentMessage.StatusEventTypeEquipped,
				CharacterId:   charId,
				Body: compartmentMessage.EquippedEventBody{
					InventoryType: int(invType),
					Source:        int(source),
					Destination:   int(dest),
				},
			})
		}()
		return nil
	}

	// Create and execute sagas concurrently
	var wg sync.WaitGroup
	for i, txnId := range transactionIds {
		wg.Add(1)
		go func(index int, transactionId uuid.UUID) {
			defer wg.Done()

			characterId := uint32(12345 + index)
			templateId := uint32(1302000 + index)

			// Create initial saga
			initialSaga := saga.Saga{
				TransactionId: transactionId,
				SagaType:      saga.InventoryTransaction,
				InitiatedBy:   fmt.Sprintf("e2e-concurrent-test-%d", index),
				Steps: []saga.Step[any]{
					{
						StepId: fmt.Sprintf("create-and-equip-step-%d", index),
						Status: saga.Pending,
						Action: saga.CreateAndEquipAsset,
						Payload: saga.CreateAndEquipAssetPayload{
							CharacterId: characterId,
							Item: saga.ItemPayload{
								TemplateId: templateId,
								Quantity:   1,
							},
						},
						CreatedAt: time.Now(),
						UpdatedAt: time.Now(),
					},
				},
			}

			// Store saga in cache
			saga.GetCache().Put(env.tenant.Id(), initialSaga)

			// Execute CreateAndEquipAsset step
			err := env.sagaProcessor.Step(transactionId)
			assert.NoError(t, err, "CreateAndEquipAsset step %d should succeed", index)

			// Wait for processing
			time.Sleep(100 * time.Millisecond)

			// Complete the CreateAndEquipAsset step
			err = env.sagaProcessor.StepCompleted(transactionId, true)
			assert.NoError(t, err, "CreateAndEquipAsset step %d completion should succeed", index)

			// Wait for auto-equip step addition
			time.Sleep(50 * time.Millisecond)

			// Execute the auto-equip step
			err = env.sagaProcessor.Step(transactionId)
			assert.NoError(t, err, "EquipAsset step %d should succeed", index)

			// Wait for processing
			time.Sleep(100 * time.Millisecond)

			// Complete the EquipAsset step
			err = env.sagaProcessor.StepCompleted(transactionId, true)
			assert.NoError(t, err, "EquipAsset step %d completion should succeed", index)

			// Mark as completed
			mutex.Lock()
			completedSagas[transactionId] = true
			mutex.Unlock()
		}(i, txnId)
	}

	// Wait for all sagas to complete
	wg.Wait()

	// Verify all sagas completed
	mutex.Lock()
	defer mutex.Unlock()
	assert.Equal(t, numSagas, len(completedSagas), "All sagas should have completed")

	// Verify final states
	for _, txnId := range transactionIds {
		finalSaga, err := env.sagaProcessor.GetById(txnId)
		assert.NoError(t, err, "Should be able to retrieve final saga %s", txnId)
		assert.Equal(t, 2, len(finalSaga.Steps), "Saga %s should have 2 steps", txnId)

		// Verify both steps are completed
		for i, step := range finalSaga.Steps {
			assert.Equal(t, saga.Completed, step.Status, "Step %d of saga %s should be completed", i, txnId)
		}
	}
}

// TestCreateAndEquipAsset_E2E_KafkaEventOrdering tests event ordering scenarios
func TestCreateAndEquipAsset_E2E_KafkaEventOrdering(t *testing.T) {
	env := setupE2EEnvironment(t)
	defer env.cleanup()

	// Start event processing
	env.wg.Add(1)
	go func() {
		defer env.wg.Done()
		env.processKafkaEvents()
	}()

	transactionId := uuid.New()
	characterId := uint32(12345)
	templateId := uint32(1302000)

	// Track event order
	var eventOrder []string
	var orderMutex sync.Mutex

	// Setup compartment processor mock
	env.compP.RequestCreateAndEquipAssetFunc = func(txnId uuid.UUID, payload compartment.CreateAndEquipAssetPayload) error {
		orderMutex.Lock()
		eventOrder = append(eventOrder, "RequestCreateAndEquipAsset")
		orderMutex.Unlock()

		// Simulate successful asset creation
		go func() {
			time.Sleep(10 * time.Millisecond)
			orderMutex.Lock()
			eventOrder = append(eventOrder, "CreatedEvent")
			orderMutex.Unlock()
			
			env.simulateKafkaEvent(compartmentMessage.StatusEvent[compartmentMessage.CreatedStatusEventBody]{
				TransactionId: transactionId,
				Type:          compartmentMessage.StatusEventTypeCreated,
				CharacterId:   characterId,
				Body: compartmentMessage.CreatedStatusEventBody{
					Type:     1,
					AssetId:  67890,
					Quantity: 1,
				},
			})
		}()

		return nil
	}

	env.compP.RequestEquipAssetFunc = func(txnId uuid.UUID, charId uint32, invType byte, source int16, dest int16) error {
		orderMutex.Lock()
		eventOrder = append(eventOrder, "RequestEquipAsset")
		orderMutex.Unlock()

		// Simulate successful equipment
		go func() {
			time.Sleep(10 * time.Millisecond)
			orderMutex.Lock()
			eventOrder = append(eventOrder, "EquippedEvent")
			orderMutex.Unlock()
			
			env.simulateKafkaEvent(compartmentMessage.StatusEvent[compartmentMessage.EquippedEventBody]{
				TransactionId: transactionId,
				Type:          compartmentMessage.StatusEventTypeEquipped,
				CharacterId:   characterId,
				Body: compartmentMessage.EquippedEventBody{
					Source:      5,
					Destination: -1,
				},
			})
		}()

		return nil
	}

	// Create and execute saga
	initialSaga := saga.Saga{
		TransactionId: transactionId,
		SagaType:      saga.InventoryTransaction,
		InitiatedBy:   "e2e-ordering-test",
		Steps: []saga.Step[any]{
			{
				StepId: "create-and-equip-step",
				Status: saga.Pending,
				Action: saga.CreateAndEquipAsset,
				Payload: saga.CreateAndEquipAssetPayload{
					CharacterId: characterId,
					Item: saga.ItemPayload{
						TemplateId: templateId,
						Quantity:   1,
					},
				},
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
			},
		},
	}

	saga.GetCache().Put(env.tenant.Id(), initialSaga)

	// Execute the full flow
	err := env.sagaProcessor.Step(transactionId)
	assert.NoError(t, err)

	time.Sleep(50 * time.Millisecond)
	err = env.sagaProcessor.StepCompleted(transactionId, true)
	assert.NoError(t, err)

	time.Sleep(50 * time.Millisecond)
	err = env.sagaProcessor.Step(transactionId)
	assert.NoError(t, err)

	time.Sleep(50 * time.Millisecond)
	err = env.sagaProcessor.StepCompleted(transactionId, true)
	assert.NoError(t, err)

	// Verify event order
	time.Sleep(100 * time.Millisecond)
	orderMutex.Lock()
	defer orderMutex.Unlock()

	expectedOrder := []string{
		"RequestCreateAndEquipAsset",
		"CreatedEvent",
		"RequestEquipAsset",
		"EquippedEvent",
	}

	assert.Equal(t, expectedOrder, eventOrder, "Events should occur in the correct order")
}