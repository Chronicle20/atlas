package saga

import (
	"context"
	"github.com/Chronicle20/atlas-constants/job"
	_map "github.com/Chronicle20/atlas-constants/map"
	tenant "github.com/Chronicle20/atlas-tenant"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"github.com/sirupsen/logrus/hooks/test"
	"github.com/stretchr/testify/assert"
	"testing"
	"time"
)

// TestCompensateCreateCharacter tests the compensateCreateCharacter function
func TestCompensateCreateCharacter(t *testing.T) {
	tests := []struct {
		name          string
		payload       CharacterCreatePayload
		expectError   bool
		errorContains string
	}{
		{
			name: "Success case - valid character creation payload",
			payload: CharacterCreatePayload{
				AccountId:    12345,
				Name:         "TestCharacter",
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
			expectError: false,
		},
		{
			name:          "Error case - invalid payload type",
			payload:       CharacterCreatePayload{}, // This will be replaced with invalid payload
			expectError:   true,
			errorContains: "invalid payload for CreateCharacter compensation",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup
			logger, _ := test.NewNullLogger()
			logger.SetLevel(logrus.DebugLevel)

			ctx := context.Background()
			te, _ := tenant.Create(uuid.New(), "GMS", 83, 1)
			tctx := tenant.WithContext(ctx, te)

			// Create test saga with failed step
			transactionId := uuid.New()
			saga := Saga{
				TransactionId: transactionId,
				SagaType:      CharacterCreation,
				InitiatedBy:   "compensation-test",
				Steps: []Step[any]{
					{
						StepId:    "create-character-step",
						Status:    Failed,
						Action:    CreateCharacter,
						Payload:   tt.payload,
						CreatedAt: time.Now(),
						UpdatedAt: time.Now(),
					},
				},
			}

			// For the invalid payload test, replace with invalid payload
			if tt.errorContains == "invalid payload for CreateCharacter compensation" {
				saga.Steps[0].Payload = "invalid-payload"
			}

			// Execute
			err := NewCompensator(logger, tctx).compensateCreateCharacter(saga, saga.Steps[0])

			// Verify
			if tt.expectError {
				assert.Error(t, err)
				if tt.errorContains != "" {
					assert.Contains(t, err.Error(), tt.errorContains)
				}
			} else {
				assert.NoError(t, err)
				// Verify that the step status was reset to pending
				assert.Equal(t, Pending, saga.Steps[0].Status)
			}
		})
	}
}
