package saga

import (
	charmock "atlas-saga-orchestrator/character/mock"
	compmock "atlas-saga-orchestrator/compartment/mock"
	"context"
	"testing"
	"time"

	"github.com/Chronicle20/atlas/libs/atlas-constants/channel"
	"github.com/Chronicle20/atlas/libs/atlas-constants/job"
	_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"
	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"github.com/sirupsen/logrus/hooks/test"
	"github.com/stretchr/testify/assert"
)

// TestPetEvolutionCompensationRefundsResources verifies that when a PetEvolution
// saga fails on evolve_pet, the reverse-walk refunds the already-completed
// resource steps: the destroyed Rock of Evolution (DestroyAsset → RequestCreateItem)
// and the deducted mesos (AwardMesos → AwardMesosAndEmit with the inverse sign so
// the player nets back to even).
//
// The consume award_mesos step uses a NEGATIVE amount to deduct the cost
// (npc op builds `amount = -cost`). The refund therefore re-credits the player
// with -payload.Amount (a positive credit). Asserting the refund amount is
// positive guards against a double-charge/double-refund sign bug.
//
// DispatchPetEvolutionRollbacks is exercised directly (mirroring
// TestPresetCompensation) to avoid the EmitSagaFailed Kafka path; no broker
// runs in the test environment.
func TestPetEvolutionCompensationRefundsResources(t *testing.T) {
	logger, _ := test.NewNullLogger()
	logger.SetLevel(logrus.DebugLevel)

	ctx := context.Background()
	te, _ := tenant.Create(uuid.New(), "GMS", 83, 1)
	tctx := tenant.WithContext(ctx, te)

	const (
		testCharId   = uint32(77001)
		rockId       = uint32(5380000)
		mesosCost    = int32(50000)
		testWorldId  = world.Id(0)
		testChannel  = channel.Id(1)
	)

	// Spy compartment processor capturing Rock re-creation.
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

	// Spy character processor capturing the mesos refund.
	var awardMesosCalls int
	var lastMesosAmount int32
	charP := &charmock.ProcessorMock{
		AwardMesosAndEmitFunc: func(_ uuid.UUID, _ channel.Model, _ uint32, _ uint32, _ string, amount int32, _ bool) error {
			awardMesosCalls++
			lastMesosAmount = amount
			return nil
		},
	}

	// Build the PetEvolution saga: completed destroy_item (Rock) + completed
	// award_mesos (negative cost) + FAILED evolve_pet.
	transactionId := uuid.New()
	s, err := NewBuilder().
		SetTransactionId(transactionId).
		SetSagaType(PetEvolution).
		SetInitiatedBy("pet-evolution-compensation-test").
		AddStep("destroy_item", Completed, DestroyAsset, DestroyAssetPayload{
			CharacterId: testCharId,
			TemplateId:  rockId,
			Quantity:    1,
			RemoveAll:   false,
		}).
		AddStep("award_mesos", Completed, AwardMesos, AwardMesosPayload{
			CharacterId: testCharId,
			WorldId:     testWorldId,
			ChannelId:   testChannel,
			ActorId:     testCharId,
			ActorType:   "SYSTEM",
			Amount:      -mesosCost, // npc deducts the cost as a negative amount
			ShowEffect:  false,
		}).
		AddStep("evolve_pet", Failed, EvolvePet, EvolvePetPayload{
			CharacterId: testCharId,
			PetId:       12345,
		}).
		Build()
	assert.NoError(t, err, "saga build should not fail")

	compensator := NewCompensator(logger, tctx).
		WithCharacterProcessor(charP).
		WithCompartmentProcessor(compP)

	compensator.DispatchPetEvolutionRollbacks(s)

	// Rock refunded exactly once, with the right template + quantity.
	assert.Equal(t, 1, len(createItemCalls), "Rock should be refunded exactly once")
	if len(createItemCalls) == 1 {
		assert.Equal(t, testCharId, createItemCalls[0].CharacterId, "refund must target the test character")
		assert.Equal(t, rockId, createItemCalls[0].TemplateId, "refunded item must be the Rock of Evolution")
		assert.Equal(t, uint32(1), createItemCalls[0].Quantity, "refunded Rock quantity must be 1")
	}

	// Mesos refunded exactly once, as a POSITIVE credit netting the player back.
	assert.Equal(t, 1, awardMesosCalls, "mesos should be refunded exactly once")
	assert.Equal(t, mesosCost, lastMesosAmount, "refund must re-credit +cost so the player nets to even")
	assert.Greater(t, lastMesosAmount, int32(0), "refund amount must be a positive credit")
}

// TestPetEvolutionCompensationDefaultsDestroyQuantity verifies that when a
// DestroyAssetPayload carries Quantity 0 (which can happen if the saga step
// was recorded without an explicit quantity), DispatchPetEvolutionRollbacks
// defaults the refund quantity to 1 rather than issuing a zero-quantity
// RequestCreateItem.
func TestPetEvolutionCompensationDefaultsDestroyQuantity(t *testing.T) {
	logger, _ := test.NewNullLogger()
	logger.SetLevel(logrus.DebugLevel)

	ctx := context.Background()
	te, _ := tenant.Create(uuid.New(), "GMS", 83, 1)
	tctx := tenant.WithContext(ctx, te)

	const (
		testCharId  = uint32(77002)
		rockId      = uint32(5380000)
		testWorldId = world.Id(0)
		testChannel = channel.Id(1)
	)

	// Spy compartment processor capturing Rock re-creation.
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

	// Character processor is not exercised by this path but must be non-nil.
	charP := &charmock.ProcessorMock{
		AwardMesosAndEmitFunc: func(_ uuid.UUID, _ channel.Model, _ uint32, _ uint32, _ string, _ int32, _ bool) error {
			return nil
		},
	}

	// Build a PetEvolution saga with a COMPLETED destroy_item step whose
	// Quantity is 0 (the edge case), plus a FAILED evolve_pet step.
	transactionId := uuid.New()
	s, err := NewBuilder().
		SetTransactionId(transactionId).
		SetSagaType(PetEvolution).
		SetInitiatedBy("pet-evolution-default-qty-test").
		AddStep("destroy_item", Completed, DestroyAsset, DestroyAssetPayload{
			CharacterId: testCharId,
			TemplateId:  rockId,
			Quantity:    0, // zero triggers the default-to-1 branch
			RemoveAll:   false,
		}).
		AddStep("evolve_pet", Failed, EvolvePet, EvolvePetPayload{
			CharacterId: testCharId,
			PetId:       99999,
		}).
		Build()
	assert.NoError(t, err, "saga build should not fail")

	compensator := NewCompensator(logger, tctx).
		WithCharacterProcessor(charP).
		WithCompartmentProcessor(compP)

	compensator.DispatchPetEvolutionRollbacks(s)

	// Rock must be refunded exactly once with quantity 1 (the zero→1 default).
	assert.Equal(t, 1, len(createItemCalls), "Rock should be refunded exactly once")
	if len(createItemCalls) == 1 {
		assert.Equal(t, testCharId, createItemCalls[0].CharacterId, "refund must target the test character")
		assert.Equal(t, rockId, createItemCalls[0].TemplateId, "refunded item must be the Rock of Evolution")
		assert.Equal(t, uint32(1), createItemCalls[0].Quantity, "zero payload quantity must default to 1")
	}
}

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

			// Determine payload for step
			var stepPayload any = tt.payload
			if tt.errorContains == "invalid payload for CreateCharacter compensation" {
				stepPayload = "invalid-payload"
			}

			// Build the saga using the builder pattern
			saga, err := NewBuilder().
				SetTransactionId(transactionId).
				SetSagaType(CharacterCreation).
				SetInitiatedBy("compensation-test").
				AddStep("create-character-step", Failed, CreateCharacter, stepPayload).
				Build()
			assert.NoError(t, err)

			// Get the step for passing to compensator
			step, ok := saga.StepAt(0)
			assert.True(t, ok)

			// Execute
			compErr := NewCompensator(logger, tctx).compensateCreateCharacter(saga, step)

			// Verify
			if tt.expectError {
				assert.Error(t, compErr)
				if tt.errorContains != "" {
					assert.Contains(t, compErr.Error(), tt.errorContains)
				}
			} else {
				assert.NoError(t, compErr)
			}
		})
	}
}
