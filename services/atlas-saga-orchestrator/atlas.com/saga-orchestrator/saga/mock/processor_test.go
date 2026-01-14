package mock

import (
	"atlas-saga-orchestrator/saga"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

// TestProcessorMockImplementsInterface verifies that ProcessorMock implements saga.Processor
func TestProcessorMockImplementsInterface(t *testing.T) {
	var _ saga.Processor = &ProcessorMock{}
}

// TestProcessorMockDefaultBehavior tests that default mock behavior returns nil/empty values
func TestProcessorMockDefaultBehavior(t *testing.T) {
	mock := &ProcessorMock{}

	// Test default returns
	sagas, err := mock.GetAll()
	assert.NoError(t, err)
	assert.Nil(t, sagas)

	s, err := mock.GetById(uuid.New())
	assert.NoError(t, err)
	assert.Equal(t, saga.Saga{}, s)

	err = mock.Put(saga.Saga{})
	assert.NoError(t, err)

	err = mock.MarkFurthestCompletedStepFailed(uuid.New())
	assert.NoError(t, err)

	err = mock.MarkEarliestPendingStep(uuid.New(), saga.Pending)
	assert.NoError(t, err)

	err = mock.MarkEarliestPendingStepCompleted(uuid.New())
	assert.NoError(t, err)

	err = mock.StepCompleted(uuid.New(), true)
	assert.NoError(t, err)

	err = mock.AddStep(uuid.New(), saga.Step[any]{})
	assert.NoError(t, err)

	err = mock.AddStepAfterCurrent(uuid.New(), saga.Step[any]{})
	assert.NoError(t, err)

	err = mock.Step(uuid.New())
	assert.NoError(t, err)
}

// TestProcessorMockCustomBehavior tests that custom mock functions are called
func TestProcessorMockCustomBehavior(t *testing.T) {
	transactionId := uuid.New()
	expectedError := assert.AnError

	mock := &ProcessorMock{
		GetByIdFunc: func(tid uuid.UUID) (saga.Saga, error) {
			assert.Equal(t, transactionId, tid)
			return saga.Saga{}, expectedError
		},
		StepCompletedFunc: func(tid uuid.UUID, success bool) error {
			assert.Equal(t, transactionId, tid)
			assert.True(t, success)
			return nil
		},
	}

	_, err := mock.GetById(transactionId)
	assert.Equal(t, expectedError, err)

	err = mock.StepCompleted(transactionId, true)
	assert.NoError(t, err)
}

// TestProcessorMockWithChaining tests that With* methods return the mock for chaining
func TestProcessorMockWithChaining(t *testing.T) {
	mock := &ProcessorMock{}

	result := mock.WithCharacterProcessor(nil)
	assert.Equal(t, mock, result)

	result = mock.WithCompartmentProcessor(nil)
	assert.Equal(t, mock, result)

	result = mock.WithSkillProcessor(nil)
	assert.Equal(t, mock, result)

	result = mock.WithValidationProcessor(nil)
	assert.Equal(t, mock, result)

	result = mock.WithGuildProcessor(nil)
	assert.Equal(t, mock, result)

	result = mock.WithInviteProcessor(nil)
	assert.Equal(t, mock, result)
}
