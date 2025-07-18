package conversation

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/Chronicle20/atlas-constants/field"
	"github.com/Chronicle20/atlas-constants/world"
	"github.com/Chronicle20/atlas-tenant"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// MockOperationExecutor is a mock implementation of the OperationExecutor interface
type MockOperationExecutor struct {
	mock.Mock
}

func (m *MockOperationExecutor) ExecuteOperation(field field.Model, characterId uint32, operation OperationModel) error {
	args := m.Called(field, characterId, operation)
	return args.Error(0)
}

func (m *MockOperationExecutor) ExecuteOperations(field field.Model, characterId uint32, operations []OperationModel) error {
	args := m.Called(field, characterId, operations)
	return args.Error(0)
}

// MockEvaluator is a mock implementation of the Evaluator interface
type MockEvaluator struct {
	mock.Mock
}

func (m *MockEvaluator) EvaluateCondition(characterId uint32, condition ConditionModel) (bool, error) {
	args := m.Called(characterId, condition)
	return args.Bool(0), args.Error(1)
}

// Helper function to create a test field
func createTestField() field.Model {
	return field.NewBuilder(world.Id(1), 1, 100000).Build()
}

// Helper function to create a test conversation context
func createTestConversationContext(characterId uint32, npcId uint32, currentState string) ConversationContext {
	conversation := createTestConversation(npcId)
	return ConversationContext{
		field:        createTestField(),
		characterId:  characterId,
		npcId:        npcId,
		currentState: currentState,
		conversation: conversation,
		context:      make(map[string]string),
	}
}

// Helper function to create a test conversation
func createTestConversation(npcId uint32) Model {
	// Create a generic action state with operations
	operation1 := OperationModel{
		operationType: "award_item",
		params: map[string]string{
			"itemId":   "4001126",
			"quantity": "1",
		},
	}
	
	operation2 := OperationModel{
		operationType: "award_mesos",
		params: map[string]string{
			"amount": "1000",
		},
	}

	outcome := OutcomeModel{
		nextState: "success_state",
		conditions: []ConditionModel{},
	}

	genericAction := GenericActionModel{
		operations: []OperationModel{operation1, operation2},
		outcomes:   []OutcomeModel{outcome},
	}

	state := StateModel{
		id:            "test_state",
		stateType:     GenericActionType,
		genericAction: &genericAction,
	}

	return Model{
		id:         uuid.New(),
		npcId:      npcId,
		startState: "test_state",
		states:     []StateModel{state},
		createdAt:  time.Now(),
		updatedAt:  time.Now(),
	}
}

// Helper function to create a test tenant
func createTestTenant() tenant.Model {
	// Create a zero-value tenant for testing
	return tenant.Model{}
}

// Helper function to create a test processor with mocked dependencies
func createTestProcessor(t *testing.T, executor OperationExecutor, evaluator Evaluator, tenant tenant.Model) *ProcessorImpl {
	logger := logrus.New()
	logger.SetLevel(logrus.DebugLevel)
	
	ctx := context.Background()
	
	return &ProcessorImpl{
		l:         logger,
		ctx:       ctx,
		t:         tenant,
		db:        nil, // Not needed for these tests
		evaluator: evaluator,
		executor:  executor,
	}
}

// Test operation execution failure scenarios
func TestProcessGenericActionState_OperationExecutionFailure(t *testing.T) {
	tests := []struct {
		name           string
		failingOpIndex int
		expectedError  string
	}{
		{
			name:           "First operation fails",
			failingOpIndex: 0,
			expectedError:  "operation failed",
		},
		{
			name:           "Second operation fails",
			failingOpIndex: 1,
			expectedError:  "second operation failed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup mocks
			mockExecutor := new(MockOperationExecutor)
			mockEvaluator := new(MockEvaluator)
			
			characterId := uint32(12345)
			npcId := uint32(9001)
			
			// Setup conversation context
			ctx := createTestConversationContext(characterId, npcId, "test_state")
			
			// Store the context in the registry
			tenant := createTestTenant()
			GetRegistry().SetContext(tenant, characterId, ctx)
			
			// Verify context is stored
			storedCtx, err := GetRegistry().GetPreviousContext(tenant, characterId)
			require.NoError(t, err)
			require.Equal(t, characterId, storedCtx.CharacterId())
			
			// Get the state from the conversation
			state, err := ctx.Conversation().FindState("test_state")
			require.NoError(t, err)
			require.NotNil(t, state.GenericAction())
			
			operations := state.GenericAction().Operations()
			require.Len(t, operations, 2)
			
			// Mock operation execution - first operations succeed, then one fails
			for i, op := range operations {
				if i == tt.failingOpIndex {
					mockExecutor.On("ExecuteOperation", ctx.Field(), characterId, op).Return(errors.New(tt.expectedError))
				} else if i < tt.failingOpIndex {
					mockExecutor.On("ExecuteOperation", ctx.Field(), characterId, op).Return(nil)
				}
				// Operations after the failing one should not be called
			}
			
			// Create processor
			processor := createTestProcessor(t, mockExecutor, mockEvaluator, tenant)
			
			// Execute the test
			nextState, err := processor.processGenericActionState(ctx, state)
			
			// Assert operation execution failure
			assert.Error(t, err)
			assert.Contains(t, err.Error(), tt.expectedError)
			assert.Empty(t, nextState)
			
			// Verify that the conversation context was cleared from registry
			_, err = GetRegistry().GetPreviousContext(tenant, characterId)
			assert.Error(t, err)
			assert.Contains(t, err.Error(), "unable to previous context")
			
			// Verify all expected calls were made
			mockExecutor.AssertExpectations(t)
			mockEvaluator.AssertExpectations(t)
		})
	}
}

// Test operation execution failure with context references
func TestProcessGenericActionState_OperationExecutionFailure_WithContextReferences(t *testing.T) {
	// Setup mocks
	mockExecutor := new(MockOperationExecutor)
	mockEvaluator := new(MockEvaluator)
	
	characterId := uint32(12345)
	npcId := uint32(9001)
	
	// Create a more complex conversation with context references
	operation := OperationModel{
		operationType: "award_item",
		params: map[string]string{
			"itemId":   "context.selectedItem",
			"quantity": "context.quantity",
		},
	}

	outcome := OutcomeModel{
		nextState: "success_state",
		conditions: []ConditionModel{},
	}

	genericAction := GenericActionModel{
		operations: []OperationModel{operation},
		outcomes:   []OutcomeModel{outcome},
	}

	state := StateModel{
		id:            "test_state",
		stateType:     GenericActionType,
		genericAction: &genericAction,
	}

	conversation := Model{
		id:         uuid.New(),
		npcId:      npcId,
		startState: "test_state",
		states:     []StateModel{state},
		createdAt:  time.Now(),
		updatedAt:  time.Now(),
	}

	// Create context with some values
	ctx := ConversationContext{
		field:        createTestField(),
		characterId:  characterId,
		npcId:        npcId,
		currentState: "test_state",
		conversation: conversation,
		context: map[string]string{
			"selectedItem": "4001126",
			"quantity":     "5",
		},
	}
	
	// Store the context in the registry
	tenant := createTestTenant()
	GetRegistry().SetContext(tenant, characterId, ctx)
	
	// Mock operation execution failure
	mockExecutor.On("ExecuteOperation", ctx.Field(), characterId, operation).Return(errors.New("context operation failed"))
	
	// Create processor
	processor := createTestProcessor(t, mockExecutor, mockEvaluator, tenant)
	
	// Execute the test
	nextState, err := processor.processGenericActionState(ctx, state)
	
	// Assert operation execution failure
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "context operation failed")
	assert.Empty(t, nextState)
	
	// Verify that the conversation context was cleared from registry
	_, err = GetRegistry().GetPreviousContext(tenant, characterId)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unable to previous context")
	
	// Verify all expected calls were made
	mockExecutor.AssertExpectations(t)
}

// Test operation execution failure with saga operations
func TestProcessGenericActionState_SagaOperationExecutionFailure(t *testing.T) {
	// Setup mocks
	mockExecutor := new(MockOperationExecutor)
	mockEvaluator := new(MockEvaluator)
	
	characterId := uint32(12345)
	npcId := uint32(9001)
	
	// Create a conversation with saga operations (non-local operations)
	operation1 := OperationModel{
		operationType: "award_item",
		params: map[string]string{
			"itemId":   "4001126",
			"quantity": "1",
		},
	}
	
	operation2 := OperationModel{
		operationType: "warp_to_map",
		params: map[string]string{
			"mapId":    "100000",
			"portalId": "0",
		},
	}

	outcome := OutcomeModel{
		nextState: "success_state",
		conditions: []ConditionModel{},
	}

	genericAction := GenericActionModel{
		operations: []OperationModel{operation1, operation2},
		outcomes:   []OutcomeModel{outcome},
	}

	state := StateModel{
		id:            "test_state",
		stateType:     GenericActionType,
		genericAction: &genericAction,
	}

	conversation := Model{
		id:         uuid.New(),
		npcId:      npcId,
		startState: "test_state",
		states:     []StateModel{state},
		createdAt:  time.Now(),
		updatedAt:  time.Now(),
	}

	ctx := ConversationContext{
		field:        createTestField(),
		characterId:  characterId,
		npcId:        npcId,
		currentState: "test_state",
		conversation: conversation,
		context:      make(map[string]string),
	}
	
	// Store the context in the registry
	tenant := createTestTenant()
	GetRegistry().SetContext(tenant, characterId, ctx)
	
	// Mock saga execution failure (e.g., saga orchestrator is down)
	mockExecutor.On("ExecuteOperation", ctx.Field(), characterId, operation1).Return(nil)
	mockExecutor.On("ExecuteOperation", ctx.Field(), characterId, operation2).Return(errors.New("saga orchestrator communication failed"))
	
	// Create processor
	processor := createTestProcessor(t, mockExecutor, mockEvaluator, tenant)
	
	// Execute the test
	nextState, err := processor.processGenericActionState(ctx, state)
	
	// Assert saga operation failure
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "saga orchestrator communication failed")
	assert.Empty(t, nextState)
	
	// Verify that the conversation context was cleared from registry
	_, err = GetRegistry().GetPreviousContext(tenant, characterId)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unable to previous context")
	
	// Verify all expected calls were made
	mockExecutor.AssertExpectations(t)
}

// Test operation execution failure with timeout scenarios
func TestProcessGenericActionState_OperationExecutionTimeout(t *testing.T) {
	// Setup mocks
	mockExecutor := new(MockOperationExecutor)
	mockEvaluator := new(MockEvaluator)
	
	characterId := uint32(12345)
	npcId := uint32(9001)
	
	// Setup conversation context
	ctx := createTestConversationContext(characterId, npcId, "test_state")
	
	// Store the context in the registry
	tenant := createTestTenant()
	GetRegistry().SetContext(tenant, characterId, ctx)
	
	// Get the state from the conversation
	state, err := ctx.Conversation().FindState("test_state")
	require.NoError(t, err)
	require.NotNil(t, state.GenericAction())
	
	operations := state.GenericAction().Operations()
	require.Len(t, operations, 2)
	
	// Mock operation execution timeout
	mockExecutor.On("ExecuteOperation", ctx.Field(), characterId, operations[0]).Return(errors.New("operation timeout: context deadline exceeded"))
	
	// Create processor
	processor := createTestProcessor(t, mockExecutor, mockEvaluator, tenant)
	
	// Execute the test
	nextState, err := processor.processGenericActionState(ctx, state)
	
	// Assert timeout failure
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "operation timeout")
	assert.Empty(t, nextState)
	
	// Verify that the conversation context was cleared from registry
	_, err = GetRegistry().GetPreviousContext(tenant, characterId)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unable to previous context")
	
	// Verify all expected calls were made
	mockExecutor.AssertExpectations(t)
}

// Test multiple operation execution failures
func TestProcessGenericActionState_MultipleOperationExecutionFailures(t *testing.T) {
	// Setup mocks
	mockExecutor := new(MockOperationExecutor)
	mockEvaluator := new(MockEvaluator)
	
	characterId := uint32(12345)
	npcId := uint32(9001)
	
	// Create conversation with multiple operations
	operations := []OperationModel{
		{
			operationType: "award_item",
			params: map[string]string{
				"itemId":   "4001126",
				"quantity": "1",
			},
		},
		{
			operationType: "award_mesos",
			params: map[string]string{
				"amount": "1000",
			},
		},
		{
			operationType: "award_exp",
			params: map[string]string{
				"amount": "100",
			},
		},
	}

	outcome := OutcomeModel{
		nextState: "success_state",
		conditions: []ConditionModel{},
	}

	genericAction := GenericActionModel{
		operations: operations,
		outcomes:   []OutcomeModel{outcome},
	}

	state := StateModel{
		id:            "test_state",
		stateType:     GenericActionType,
		genericAction: &genericAction,
	}

	conversation := Model{
		id:         uuid.New(),
		npcId:      npcId,
		startState: "test_state",
		states:     []StateModel{state},
		createdAt:  time.Now(),
		updatedAt:  time.Now(),
	}

	ctx := ConversationContext{
		field:        createTestField(),
		characterId:  characterId,
		npcId:        npcId,
		currentState: "test_state",
		conversation: conversation,
		context:      make(map[string]string),
	}
	
	// Store the context in the registry
	tenant := createTestTenant()
	GetRegistry().SetContext(tenant, characterId, ctx)
	
	// Mock first operation succeeds, second fails
	mockExecutor.On("ExecuteOperation", ctx.Field(), characterId, operations[0]).Return(nil)
	mockExecutor.On("ExecuteOperation", ctx.Field(), characterId, operations[1]).Return(errors.New("insufficient funds"))
	// Third operation should not be called since second failed
	
	// Create processor
	processor := createTestProcessor(t, mockExecutor, mockEvaluator, tenant)
	
	// Execute the test
	nextState, err := processor.processGenericActionState(ctx, state)
	
	// Assert operation execution failure
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "insufficient funds")
	assert.Empty(t, nextState)
	
	// Verify that the conversation context was cleared from registry
	_, err = GetRegistry().GetPreviousContext(tenant, characterId)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unable to previous context")
	
	// Verify all expected calls were made
	mockExecutor.AssertExpectations(t)
}

// Test operation execution failure with panic recovery
func TestProcessGenericActionState_PanicRecovery(t *testing.T) {
	// Setup mocks
	mockExecutor := new(MockOperationExecutor)
	mockEvaluator := new(MockEvaluator)
	
	characterId := uint32(12345)
	npcId := uint32(9001)
	
	// Setup conversation context
	ctx := createTestConversationContext(characterId, npcId, "test_state")
	
	// Store the context in the registry
	tenant := createTestTenant()
	GetRegistry().SetContext(tenant, characterId, ctx)
	
	// Get the state from the conversation
	state, err := ctx.Conversation().FindState("test_state")
	require.NoError(t, err)
	require.NotNil(t, state.GenericAction())
	
	operations := state.GenericAction().Operations()
	require.Len(t, operations, 2)
	
	// Mock operation execution panic
	mockExecutor.On("ExecuteOperation", ctx.Field(), characterId, operations[0]).Run(func(args mock.Arguments) {
		panic("unexpected panic during operation execution")
	}).Return(nil)
	
	// Create processor
	processor := createTestProcessor(t, mockExecutor, mockEvaluator, tenant)
	
	// Execute the test - should not panic due to defer recover
	nextState, err := processor.processGenericActionState(ctx, state)
	
	// The panic should be recovered and not propagate
	// processGenericActionState should return normally
	assert.NoError(t, err) // Note: with current implementation, panic is logged but doesn't return error
	assert.Empty(t, nextState)
	
	// Verify that the conversation context was cleared from registry due to panic recovery
	_, err = GetRegistry().GetPreviousContext(tenant, characterId)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unable to previous context")
	
	// Verify the expected call was made
	mockExecutor.AssertExpectations(t)
}

// Test condition evaluation failure scenarios
func TestProcessGenericActionState_ConditionEvaluationFailure(t *testing.T) {
	tests := []struct {
		name              string
		conditionType     string
		operator          string
		value             string
		itemId            uint32
		expectedError     string
		setupOperations   bool
	}{
		{
			name:              "Level condition evaluation fails",
			conditionType:     "level",
			operator:          ">=",
			value:             "10",
			itemId:            0,
			expectedError:     "failed to validate level condition",
			setupOperations:   true,
		},
		{
			name:              "Item condition evaluation fails",
			conditionType:     "item",
			operator:          ">=",
			value:             "1",
			itemId:            4001126,
			expectedError:     "failed to validate item condition",
			setupOperations:   true,
		},
		{
			name:              "Mesos condition evaluation fails",
			conditionType:     "mesos",
			operator:          ">=",
			value:             "1000",
			itemId:            0,
			expectedError:     "failed to validate mesos condition",
			setupOperations:   true,
		},
		{
			name:              "Quest condition evaluation fails",
			conditionType:     "quest",
			operator:          "==",
			value:             "completed",
			itemId:            0,
			expectedError:     "failed to validate quest condition",
			setupOperations:   true,
		},
		{
			name:              "Condition evaluation fails without operations",
			conditionType:     "level",
			operator:          ">=",
			value:             "50",
			itemId:            0,
			expectedError:     "validation service unavailable",
			setupOperations:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup mocks
			mockExecutor := new(MockOperationExecutor)
			mockEvaluator := new(MockEvaluator)
			
			characterId := uint32(12345)
			npcId := uint32(9001)
			
			// Create a conversation with operations and condition-based outcomes
			operations := []OperationModel{}
			if tt.setupOperations {
				operations = []OperationModel{
					{
						operationType: "award_item",
						params: map[string]string{
							"itemId":   "4001126",
							"quantity": "1",
						},
					},
					{
						operationType: "award_mesos",
						params: map[string]string{
							"amount": "1000",
						},
					},
				}
			}

			// Create condition that will fail evaluation
			condition := ConditionModel{
				conditionType: tt.conditionType,
				operator:      tt.operator,
				value:         tt.value,
				itemId:        tt.itemId,
			}

			outcome := OutcomeModel{
				nextState:    "success_state",
				failureState: "failure_state",
				conditions:   []ConditionModel{condition},
			}

			genericAction := GenericActionModel{
				operations: operations,
				outcomes:   []OutcomeModel{outcome},
			}

			state := StateModel{
				id:            "test_state",
				stateType:     GenericActionType,
				genericAction: &genericAction,
			}

			conversation := Model{
				id:         uuid.New(),
				npcId:      npcId,
				startState: "test_state",
				states:     []StateModel{state},
				createdAt:  time.Now(),
				updatedAt:  time.Now(),
			}

			ctx := ConversationContext{
				field:        createTestField(),
				characterId:  characterId,
				npcId:        npcId,
				currentState: "test_state",
				conversation: conversation,
				context:      make(map[string]string),
			}
			
			// Store the context in the registry
			tenant := createTestTenant()
			GetRegistry().SetContext(tenant, characterId, ctx)
			
			// Mock operations to succeed if present
			if tt.setupOperations {
				for _, op := range operations {
					mockExecutor.On("ExecuteOperation", ctx.Field(), characterId, op).Return(nil)
				}
			}
			
			// Mock condition evaluation to fail
			mockEvaluator.On("EvaluateCondition", characterId, condition).Return(false, errors.New(tt.expectedError))
			
			// Create processor
			processor := createTestProcessor(t, mockExecutor, mockEvaluator, tenant)
			
			// Execute the test
			nextState, err := processor.processGenericActionState(ctx, state)
			
			// Assert condition evaluation failure
			assert.Error(t, err)
			assert.Contains(t, err.Error(), tt.expectedError)
			assert.Empty(t, nextState)
			
			// Verify that the conversation context was cleared from registry
			_, err = GetRegistry().GetPreviousContext(tenant, characterId)
			assert.Error(t, err)
			assert.Contains(t, err.Error(), "unable to previous context")
			
			// Verify all expected calls were made
			mockExecutor.AssertExpectations(t)
			mockEvaluator.AssertExpectations(t)
		})
	}
}

// Test condition evaluation failure with multiple conditions
func TestProcessGenericActionState_MultipleConditionEvaluationFailure(t *testing.T) {
	// Setup mocks
	mockExecutor := new(MockOperationExecutor)
	mockEvaluator := new(MockEvaluator)
	
	characterId := uint32(12345)
	npcId := uint32(9001)
	
	// Create operations that will succeed
	operations := []OperationModel{
		{
			operationType: "award_item",
			params: map[string]string{
				"itemId":   "4001126",
				"quantity": "1",
			},
		},
	}

	// Create multiple conditions - first will fail
	condition1 := ConditionModel{
		conditionType: "level",
		operator:      ">=",
		value:         "10",
		itemId:        0,
	}
	
	condition2 := ConditionModel{
		conditionType: "item",
		operator:      ">=",
		value:         "1",
		itemId:        4001126,
	}

	// Create outcome with multiple conditions
	outcome := OutcomeModel{
		nextState:    "success_state",
		failureState: "failure_state",
		conditions:   []ConditionModel{condition1, condition2},
	}

	genericAction := GenericActionModel{
		operations: operations,
		outcomes:   []OutcomeModel{outcome},
	}

	state := StateModel{
		id:            "test_state",
		stateType:     GenericActionType,
		genericAction: &genericAction,
	}

	conversation := Model{
		id:         uuid.New(),
		npcId:      npcId,
		startState: "test_state",
		states:     []StateModel{state},
		createdAt:  time.Now(),
		updatedAt:  time.Now(),
	}

	ctx := ConversationContext{
		field:        createTestField(),
		characterId:  characterId,
		npcId:        npcId,
		currentState: "test_state",
		conversation: conversation,
		context:      make(map[string]string),
	}
	
	// Store the context in the registry
	tenant := createTestTenant()
	GetRegistry().SetContext(tenant, characterId, ctx)
	
	// Mock operations to succeed
	for _, op := range operations {
		mockExecutor.On("ExecuteOperation", ctx.Field(), characterId, op).Return(nil)
	}
	
	// Mock first condition evaluation to fail (only first condition is evaluated based on code)
	mockEvaluator.On("EvaluateCondition", characterId, condition1).Return(false, errors.New("character level too low"))
	
	// Create processor
	processor := createTestProcessor(t, mockExecutor, mockEvaluator, tenant)
	
	// Execute the test
	nextState, err := processor.processGenericActionState(ctx, state)
	
	// Assert condition evaluation failure
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "character level too low")
	assert.Empty(t, nextState)
	
	// Verify that the conversation context was cleared from registry
	_, err = GetRegistry().GetPreviousContext(tenant, characterId)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unable to previous context")
	
	// Verify all expected calls were made
	mockExecutor.AssertExpectations(t)
	mockEvaluator.AssertExpectations(t)
}

// Test condition evaluation failure with timeout scenarios
func TestProcessGenericActionState_ConditionEvaluationTimeout(t *testing.T) {
	// Setup mocks
	mockExecutor := new(MockOperationExecutor)
	mockEvaluator := new(MockEvaluator)
	
	characterId := uint32(12345)
	npcId := uint32(9001)
	
	// Create operations that will succeed
	operations := []OperationModel{
		{
			operationType: "award_exp",
			params: map[string]string{
				"amount": "100",
			},
		},
	}

	// Create condition that will timeout during evaluation
	condition := ConditionModel{
		conditionType: "level",
		operator:      ">=",
		value:         "10",
		itemId:        0,
	}

	outcome := OutcomeModel{
		nextState:    "success_state",
		failureState: "failure_state",
		conditions:   []ConditionModel{condition},
	}

	genericAction := GenericActionModel{
		operations: operations,
		outcomes:   []OutcomeModel{outcome},
	}

	state := StateModel{
		id:            "test_state",
		stateType:     GenericActionType,
		genericAction: &genericAction,
	}

	conversation := Model{
		id:         uuid.New(),
		npcId:      npcId,
		startState: "test_state",
		states:     []StateModel{state},
		createdAt:  time.Now(),
		updatedAt:  time.Now(),
	}

	ctx := ConversationContext{
		field:        createTestField(),
		characterId:  characterId,
		npcId:        npcId,
		currentState: "test_state",
		conversation: conversation,
		context:      make(map[string]string),
	}
	
	// Store the context in the registry
	tenant := createTestTenant()
	GetRegistry().SetContext(tenant, characterId, ctx)
	
	// Mock operations to succeed
	for _, op := range operations {
		mockExecutor.On("ExecuteOperation", ctx.Field(), characterId, op).Return(nil)
	}
	
	// Mock condition evaluation to timeout
	mockEvaluator.On("EvaluateCondition", characterId, condition).Return(false, errors.New("condition evaluation timeout: context deadline exceeded"))
	
	// Create processor
	processor := createTestProcessor(t, mockExecutor, mockEvaluator, tenant)
	
	// Execute the test
	nextState, err := processor.processGenericActionState(ctx, state)
	
	// Assert condition evaluation timeout
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "condition evaluation timeout")
	assert.Empty(t, nextState)
	
	// Verify that the conversation context was cleared from registry
	_, err = GetRegistry().GetPreviousContext(tenant, characterId)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unable to previous context")
	
	// Verify all expected calls were made
	mockExecutor.AssertExpectations(t)
	mockEvaluator.AssertExpectations(t)
}

// Test condition evaluation failure with external service errors
func TestProcessGenericActionState_ConditionEvaluationExternalServiceError(t *testing.T) {
	// Setup mocks
	mockExecutor := new(MockOperationExecutor)
	mockEvaluator := new(MockEvaluator)
	
	characterId := uint32(12345)
	npcId := uint32(9001)
	
	// Create operations that will succeed
	operations := []OperationModel{
		{
			operationType: "award_item",
			params: map[string]string{
				"itemId":   "4001126",
				"quantity": "1",
			},
		},
	}

	// Create condition that will fail due to external service error
	condition := ConditionModel{
		conditionType: "quest",
		operator:      "==",
		value:         "completed",
		itemId:        0,
	}

	outcome := OutcomeModel{
		nextState:    "success_state",
		failureState: "failure_state",
		conditions:   []ConditionModel{condition},
	}

	genericAction := GenericActionModel{
		operations: operations,
		outcomes:   []OutcomeModel{outcome},
	}

	state := StateModel{
		id:            "test_state",
		stateType:     GenericActionType,
		genericAction: &genericAction,
	}

	conversation := Model{
		id:         uuid.New(),
		npcId:      npcId,
		startState: "test_state",
		states:     []StateModel{state},
		createdAt:  time.Now(),
		updatedAt:  time.Now(),
	}

	ctx := ConversationContext{
		field:        createTestField(),
		characterId:  characterId,
		npcId:        npcId,
		currentState: "test_state",
		conversation: conversation,
		context:      make(map[string]string),
	}
	
	// Store the context in the registry
	tenant := createTestTenant()
	GetRegistry().SetContext(tenant, characterId, ctx)
	
	// Mock operations to succeed
	for _, op := range operations {
		mockExecutor.On("ExecuteOperation", ctx.Field(), characterId, op).Return(nil)
	}
	
	// Mock condition evaluation to fail with external service error
	mockEvaluator.On("EvaluateCondition", characterId, condition).Return(false, errors.New("quest service unavailable"))
	
	// Create processor
	processor := createTestProcessor(t, mockExecutor, mockEvaluator, tenant)
	
	// Execute the test
	nextState, err := processor.processGenericActionState(ctx, state)
	
	// Assert condition evaluation failure
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "quest service unavailable")
	assert.Empty(t, nextState)
	
	// Verify that the conversation context was cleared from registry
	_, err = GetRegistry().GetPreviousContext(tenant, characterId)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unable to previous context")
	
	// Verify all expected calls were made
	mockExecutor.AssertExpectations(t)
	mockEvaluator.AssertExpectations(t)
}

// Test condition evaluation failure with invalid condition parameters
func TestProcessGenericActionState_ConditionEvaluationInvalidParameters(t *testing.T) {
	// Setup mocks
	mockExecutor := new(MockOperationExecutor)
	mockEvaluator := new(MockEvaluator)
	
	characterId := uint32(12345)
	npcId := uint32(9001)
	
	// Create operations that will succeed
	operations := []OperationModel{
		{
			operationType: "award_mesos",
			params: map[string]string{
				"amount": "500",
			},
		},
	}

	// Create condition with invalid parameters
	condition := ConditionModel{
		conditionType: "item",
		operator:      "invalid_operator",
		value:         "not_a_number",
		itemId:        0, // Invalid item ID for item condition
	}

	outcome := OutcomeModel{
		nextState:    "success_state",
		failureState: "failure_state",
		conditions:   []ConditionModel{condition},
	}

	genericAction := GenericActionModel{
		operations: operations,
		outcomes:   []OutcomeModel{outcome},
	}

	state := StateModel{
		id:            "test_state",
		stateType:     GenericActionType,
		genericAction: &genericAction,
	}

	conversation := Model{
		id:         uuid.New(),
		npcId:      npcId,
		startState: "test_state",
		states:     []StateModel{state},
		createdAt:  time.Now(),
		updatedAt:  time.Now(),
	}

	ctx := ConversationContext{
		field:        createTestField(),
		characterId:  characterId,
		npcId:        npcId,
		currentState: "test_state",
		conversation: conversation,
		context:      make(map[string]string),
	}
	
	// Store the context in the registry
	tenant := createTestTenant()
	GetRegistry().SetContext(tenant, characterId, ctx)
	
	// Mock operations to succeed
	for _, op := range operations {
		mockExecutor.On("ExecuteOperation", ctx.Field(), characterId, op).Return(nil)
	}
	
	// Mock condition evaluation to fail with invalid parameter error
	mockEvaluator.On("EvaluateCondition", characterId, condition).Return(false, errors.New("invalid condition parameters: operator 'invalid_operator' not supported"))
	
	// Create processor
	processor := createTestProcessor(t, mockExecutor, mockEvaluator, tenant)
	
	// Execute the test
	nextState, err := processor.processGenericActionState(ctx, state)
	
	// Assert condition evaluation failure
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid condition parameters")
	assert.Empty(t, nextState)
	
	// Verify that the conversation context was cleared from registry
	_, err = GetRegistry().GetPreviousContext(tenant, characterId)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unable to previous context")
	
	// Verify all expected calls were made
	mockExecutor.AssertExpectations(t)
	mockEvaluator.AssertExpectations(t)
}

// Test condition evaluation failure with context parameter resolution
func TestProcessGenericActionState_ConditionEvaluationContextParameterFailure(t *testing.T) {
	// Setup mocks
	mockExecutor := new(MockOperationExecutor)
	mockEvaluator := new(MockEvaluator)
	
	characterId := uint32(12345)
	npcId := uint32(9001)
	
	// Create operations that will succeed
	operations := []OperationModel{
		{
			operationType: "award_item",
			params: map[string]string{
				"itemId":   "4001126",
				"quantity": "1",
			},
		},
	}

	// Create condition that references context but evaluator fails to resolve
	condition := ConditionModel{
		conditionType: "item",
		operator:      ">=",
		value:         "context.requiredQuantity", // Context parameter
		itemId:        4001126,
	}

	outcome := OutcomeModel{
		nextState:    "success_state",
		failureState: "failure_state",
		conditions:   []ConditionModel{condition},
	}

	genericAction := GenericActionModel{
		operations: operations,
		outcomes:   []OutcomeModel{outcome},
	}

	state := StateModel{
		id:            "test_state",
		stateType:     GenericActionType,
		genericAction: &genericAction,
	}

	conversation := Model{
		id:         uuid.New(),
		npcId:      npcId,
		startState: "test_state",
		states:     []StateModel{state},
		createdAt:  time.Now(),
		updatedAt:  time.Now(),
	}

	ctx := ConversationContext{
		field:        createTestField(),
		characterId:  characterId,
		npcId:        npcId,
		currentState: "test_state",
		conversation: conversation,
		context: map[string]string{
			"requiredQuantity": "5", // Context has the value but evaluator fails
		},
	}
	
	// Store the context in the registry
	tenant := createTestTenant()
	GetRegistry().SetContext(tenant, characterId, ctx)
	
	// Mock operations to succeed
	for _, op := range operations {
		mockExecutor.On("ExecuteOperation", ctx.Field(), characterId, op).Return(nil)
	}
	
	// Mock condition evaluation to fail with context resolution error
	mockEvaluator.On("EvaluateCondition", characterId, condition).Return(false, errors.New("failed to resolve context parameter 'requiredQuantity'"))
	
	// Create processor
	processor := createTestProcessor(t, mockExecutor, mockEvaluator, tenant)
	
	// Execute the test
	nextState, err := processor.processGenericActionState(ctx, state)
	
	// Assert condition evaluation failure
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to resolve context parameter")
	assert.Empty(t, nextState)
	
	// Verify that the conversation context was cleared from registry
	_, err = GetRegistry().GetPreviousContext(tenant, characterId)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unable to previous context")
	
	// Verify all expected calls were made
	mockExecutor.AssertExpectations(t)
	mockEvaluator.AssertExpectations(t)
}