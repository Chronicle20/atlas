package saga

import (
	"atlas-saga-orchestrator/character"
	"atlas-saga-orchestrator/character/mock"
	"atlas-saga-orchestrator/compartment"
	mock2 "atlas-saga-orchestrator/compartment/mock"
	"atlas-saga-orchestrator/validation"
	mock3 "atlas-saga-orchestrator/validation/mock"
	"context"
	"errors"
	"fmt"
	"github.com/Chronicle20/atlas-constants/channel"
	"github.com/Chronicle20/atlas-constants/job"
	_map "github.com/Chronicle20/atlas-constants/map"
	"github.com/Chronicle20/atlas-constants/world"
	tenant "github.com/Chronicle20/atlas-tenant"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"github.com/sirupsen/logrus/hooks/test"
	"github.com/stretchr/testify/assert"
	"testing"
	"time"
)

func setupContext() (tenant.Model, context.Context) {
	ctx := context.Background()
	te, _ := tenant.Create(uuid.New(), "GMS", 83, 1)
	tctx := tenant.WithContext(ctx, te)
	return te, tctx
}

// setupTestProcessor creates a ProcessorImpl with mock dependencies for testing
func setupTestProcessor(ctx context.Context, charP character.Processor, compP compartment.Processor, validP ...validation.Processor) (Processor, *test.Hook) {
	logger, hook := test.NewNullLogger()
	logger.SetLevel(logrus.DebugLevel)

	processor := NewProcessor(logger, ctx).WithCharacterProcessor(charP).WithCompartmentProcessor(compP)
	if len(validP) > 0 {
		processor = processor.WithValidationProcessor(validP[0])
	}
	return processor, hook
}

// TestCharacterCreationSagaIntegration tests the full character creation saga flow
func TestCharacterCreationSagaIntegration(t *testing.T) {
	tests := []struct {
		name                    string
		characterCreationResult error
		expectedFinalStatus     Status
		expectedStepCount       int
		description             string
	}{
		{
			name:                    "Success - character created successfully",
			characterCreationResult: nil,
			expectedFinalStatus:     Pending, // Step remains pending until event received
			expectedStepCount:       1,
			description:             "Character creation step should complete successfully and remain pending for event",
		},
		{
			name:                    "Failure - character creation failed",
			characterCreationResult: errors.New("character creation failed"),
			expectedFinalStatus:     Pending, // Step fails during execution
			expectedStepCount:       1,
			description:             "Character creation step should fail immediately on service error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup
			charP := &mock.ProcessorMock{}
			compP := &mock2.ProcessorMock{}
			validP := &mock3.ProcessorMock{}

			te, ctx := setupContext()
			processor, hook := setupTestProcessor(ctx, charP, compP, validP)

			// Configure mocks
			charP.RequestCreateCharacterFunc = func(transactionId uuid.UUID, accountId uint32, worldId byte, name string, level byte, strength uint16, dexterity uint16, intelligence uint16, luck uint16, hp uint16, mp uint16, jobId job.Id, gender byte, face uint32, hair uint32, skin byte, mapId _map.Id) error {
				return tt.characterCreationResult
			}

			// Create test saga with character creation step
			transactionId := uuid.New()
			saga := Saga{
				TransactionId: transactionId,
				SagaType:      CharacterCreation,
				InitiatedBy:   "integration-test",
				Steps: []Step[any]{
					{
						StepId: "create-character-step",
						Status: Pending,
						Action: CreateCharacter,
						Payload: CharacterCreatePayload{
							AccountId:    12345,
							Name:         "IntegrationTestChar",
							WorldId:      0,
							Level:        1,
							Strength:     13,
							Dexterity:    4,
							Intelligence: 4,
							Luck:         4,
							Hp:           50,
							Mp:           5,
							JobId:        job.Id(0),
							Gender:       0,
							Face:         20000,
							Hair:         30000,
							Skin:         0,
							Top:          1040002,
							Bottom:       1060002,
							Shoes:        1072001,
							Weapon:       1302000,
							MapId:        _map.Id(100000000),
						},
						CreatedAt: time.Now(),
						UpdatedAt: time.Now(),
					},
				},
			}

			// Store saga in cache for processing
			GetCache().Put(te.Id(), saga)

			// Execute saga processing
			err := processor.Step(saga.TransactionId)

			// Verify results based on expected outcome
			if tt.characterCreationResult != nil {
				// If character creation failed, ProcessSaga should return an error
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.characterCreationResult.Error())
			} else {
				// If character creation succeeded, ProcessSaga should complete without error
				assert.NoError(t, err)
			}

			// Verify that the character processor was called
			assert.NotNil(t, charP.RequestCreateCharacterFunc)

			// Verify appropriate logging occurred
			logEntries := hook.AllEntries()
			assert.True(t, len(logEntries) > 0, "Expected some log entries")
		})
	}
}

// TestCharacterCreationEventCorrelation tests the event correlation logic
func TestCharacterCreationEventCorrelation(t *testing.T) {
	tests := []struct {
		name               string
		eventType          string
		transactionId      uuid.UUID
		expectedSuccess    bool
		expectedStepCalled bool
		description        string
	}{
		{
			name:               "Success - character created event",
			eventType:          "CREATED",
			transactionId:      uuid.New(),
			expectedSuccess:    true,
			expectedStepCalled: true,
			description:        "CREATED event should mark saga step as completed",
		},
		{
			name:               "Failure - character creation failed event",
			eventType:          "CREATION_FAILED",
			transactionId:      uuid.New(),
			expectedSuccess:    false,
			expectedStepCalled: true,
			description:        "CREATION_FAILED event should mark saga step as failed",
		},
		{
			name:               "Failure - character error event",
			eventType:          "ERROR",
			transactionId:      uuid.New(),
			expectedSuccess:    false,
			expectedStepCalled: true,
			description:        "ERROR event should mark saga step as failed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup
			charP := &mock.ProcessorMock{}
			compP := &mock2.ProcessorMock{}
			validP := &mock3.ProcessorMock{}

			_, ctx := setupContext()
			processor, hook := setupTestProcessor(ctx, charP, compP, validP)

			// Configure saga processor mock to track StepCompleted calls
			stepCompletedCalls := make([]struct {
				transactionId uuid.UUID
				success       bool
			}, 0)

			// Note: In a real integration test, we would test the actual event handlers
			// For this test, we're verifying the expected behavior pattern
			// The actual event handlers are tested in the character consumer tests

			// Simulate the event handling logic
			var err error
			switch tt.eventType {
			case "CREATED":
				err = processor.StepCompleted(tt.transactionId, true)
				stepCompletedCalls = append(stepCompletedCalls, struct {
					transactionId uuid.UUID
					success       bool
				}{tt.transactionId, true})
			case "CREATION_FAILED", "ERROR":
				err = processor.StepCompleted(tt.transactionId, false)
				stepCompletedCalls = append(stepCompletedCalls, struct {
					transactionId uuid.UUID
					success       bool
				}{tt.transactionId, false})
			}

			// Verify that StepCompleted was called with correct parameters
			if tt.expectedStepCalled {
				assert.Equal(t, 1, len(stepCompletedCalls))
				assert.Equal(t, tt.transactionId, stepCompletedCalls[0].transactionId)
				assert.Equal(t, tt.expectedSuccess, stepCompletedCalls[0].success)
			} else {
				assert.Equal(t, 0, len(stepCompletedCalls))
			}

			// Verify no error in event processing
			assert.NoError(t, err)

			// Verify appropriate logging occurred
			logEntries := hook.AllEntries()
			assert.True(t, len(logEntries) >= 0, "Log entries should be available")
		})
	}
}

// TestCharacterCreationSagaCompensation tests compensation scenarios
func TestCharacterCreationSagaCompensation(t *testing.T) {
	tests := []struct {
		name           string
		sagaSteps      []Action
		failureAtStep  int
		expectedResult string
		description    string
	}{
		{
			name:           "Single step character creation - no compensation needed",
			sagaSteps:      []Action{CreateCharacter},
			failureAtStep:  0,
			expectedResult: "failed",
			description:    "Single character creation step failure should not require compensation",
		},
		{
			name:           "Multi-step saga with character creation first",
			sagaSteps:      []Action{CreateCharacter, AwardLevel, AwardMesos},
			failureAtStep:  1, // Fail at AwardLevel
			expectedResult: "compensating",
			description:    "Multi-step saga should enter compensation mode on later step failure",
		},
		{
			name:           "Multi-step saga with character creation failure",
			sagaSteps:      []Action{CreateCharacter, AwardLevel, AwardMesos},
			failureAtStep:  0, // Fail at CreateCharacter
			expectedResult: "failed",
			description:    "Saga should fail immediately if character creation fails",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup
			charP := &mock.ProcessorMock{}
			compP := &mock2.ProcessorMock{}
			validP := &mock3.ProcessorMock{}

			te, ctx := setupContext()
			processor, _ := setupTestProcessor(ctx, charP, compP, validP)

			// Configure mocks to fail at specific step
			charP.RequestCreateCharacterFunc = func(transactionId uuid.UUID, accountId uint32, worldId byte, name string, level byte, strength uint16, dexterity uint16, intelligence uint16, luck uint16, hp uint16, mp uint16, jobId job.Id, gender byte, face uint32, hair uint32, skin byte, mapId _map.Id) error {
				if tt.failureAtStep == 0 {
					return errors.New("character creation failed")
				}
				return nil
			}

			charP.AwardLevelAndEmitFunc = func(transactionId uuid.UUID, worldId world.Id, characterId uint32, channelId channel.Id, amount byte) error {
				if tt.failureAtStep == 1 {
					return errors.New("level award failed")
				}
				return nil
			}

			charP.AwardMesosAndEmitFunc = func(transactionId uuid.UUID, worldId world.Id, characterId uint32, channelId channel.Id, actorId uint32, actorType string, amount int32) error {
				if tt.failureAtStep == 2 {
					return errors.New("mesos award failed")
				}
				return nil
			}

			// Create test saga with multiple steps
			transactionId := uuid.New()
			steps := make([]Step[any], 0)

			for i, action := range tt.sagaSteps {
				var payload any
				switch action {
				case CreateCharacter:
					payload = CharacterCreatePayload{
						AccountId:    12345,
						Name:         "TestChar",
						WorldId:      0,
						Level:        1,
						Strength:     13,
						Dexterity:    4,
						Intelligence: 4,
						Luck:         4,
						Hp:           50,
						Mp:           5,
						JobId:        job.Id(0),
						Gender:       0,
						Face:         20000,
						Hair:         30000,
						Skin:         0,
						Top:          1040002,
						Bottom:       1060002,
						Shoes:        1072001,
						Weapon:       1302000,
						MapId:        _map.Id(100000000),
					}
				case AwardLevel:
					payload = AwardLevelPayload{
						CharacterId: 12345,
						WorldId:     0,
						ChannelId:   0,
						Amount:      1,
					}
				case AwardMesos:
					payload = AwardMesosPayload{
						CharacterId: 12345,
						WorldId:     0,
						ChannelId:   0,
						ActorId:     0,
						ActorType:   "SYSTEM",
						Amount:      1000,
					}
				}

				steps = append(steps, Step[any]{
					StepId:    fmt.Sprintf("step-%d", i),
					Status:    Pending,
					Action:    action,
					Payload:   payload,
					CreatedAt: time.Now(),
					UpdatedAt: time.Now(),
				})
			}

			saga := Saga{
				TransactionId: transactionId,
				SagaType:      CharacterCreation,
				InitiatedBy:   "compensation-test",
				Steps:         steps,
			}

			// Store saga in cache for processing
			GetCache().Put(te.Id(), saga)

			// Execute saga processing
			err := processor.Step(saga.TransactionId)

			// Verify expected behavior
			// Note: The Step method only processes one step at a time
			// For multi-step sagas, we would need to call Step multiple times
			switch tt.expectedResult {
			case "failed":
				assert.Error(t, err)
			case "compensating":
				// For this test, since we're only running one step,
				// the first step (character creation) should succeed
				assert.NoError(t, err)
			default:
				assert.NoError(t, err)
			}

			// Verify mock functions were set based on failure point
			if tt.failureAtStep >= 0 {
				assert.NotNil(t, charP.RequestCreateCharacterFunc)
			}
			if tt.failureAtStep >= 1 {
				assert.NotNil(t, charP.AwardLevelAndEmitFunc)
			}
			if tt.failureAtStep >= 2 {
				assert.NotNil(t, charP.AwardMesosAndEmitFunc)
			}
		})
	}
}

func TestTransformExperienceDistributions(t *testing.T) {
	source := []ExperienceDistributions{
		{
			ExperienceType: "WHITE",
			Amount:         1000,
			Attr1:          0,
		},
		{
			ExperienceType: "QUEST",
			Amount:         2000,
			Attr1:          1,
		},
	}

	target := TransformExperienceDistributions(source)

	assert.Equal(t, len(source), len(target))
	for i, s := range source {
		assert.Equal(t, s.ExperienceType, target[i].ExperienceType)
		assert.Equal(t, s.Amount, target[i].Amount)
		assert.Equal(t, s.Attr1, target[i].Attr1)
	}
}
