package saga

import (
	cashshopmock "atlas-saga-orchestrator/cashshop/mock"
	charmock "atlas-saga-orchestrator/character/mock"
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
	"github.com/Chronicle20/atlas/libs/atlas-constants/job"
	_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"
	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
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
		testCharId  = uint32(77001)
		rockId      = uint32(5380000)
		mesosCost   = int32(50000)
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

// TestCashItemUseCompensationRefundsConsumedItems verifies that when a
// cash-item-use saga (ItemTagUse/SealingLockUse/IncubatorUse) fails, the
// reverse-walk re-creates every consumed item (DestroyAsset/
// DestroyAssetFromSlot → RequestCreateItem) and destroys every awarded result
// (AwardAsset → RequestDestroyItem), mirroring
// TestPetEvolutionCompensationRefundsResources / DispatchPetEvolutionRollbacks
// (Task 10).
//
// DispatchCashItemUseRollbacks is exercised directly (mirroring
// TestPetEvolutionCompensationRefundsResources) to avoid the EmitSagaFailed
// Kafka path.
func TestCashItemUseCompensationRefundsConsumedItems(t *testing.T) {
	logger, _ := test.NewNullLogger()
	logger.SetLevel(logrus.DebugLevel)

	ctx := context.Background()
	te, _ := tenant.Create(uuid.New(), "GMS", 83, 1)
	tctx := tenant.WithContext(ctx, te)

	const (
		testCharId   = uint32(88001)
		sealItemId   = uint32(5390000)
		resultItemId = uint32(5390001)
	)

	type createCall struct {
		CharacterId uint32
		TemplateId  uint32
		Quantity    uint32
	}
	var createItemCalls []createCall
	type destroyCall struct {
		CharacterId uint32
		TemplateId  uint32
		Quantity    uint32
		RemoveAll   bool
	}
	var destroyItemCalls []destroyCall
	compP := &compmock.ProcessorMock{
		RequestCreateItemFunc: func(_ uuid.UUID, characterId uint32, templateId uint32, quantity uint32, _ time.Time) error {
			createItemCalls = append(createItemCalls, createCall{
				CharacterId: characterId,
				TemplateId:  templateId,
				Quantity:    quantity,
			})
			return nil
		},
		RequestDestroyItemFunc: func(_ uuid.UUID, characterId uint32, templateId uint32, quantity uint32, removeAll bool) error {
			destroyItemCalls = append(destroyItemCalls, destroyCall{
				CharacterId: characterId,
				TemplateId:  templateId,
				Quantity:    quantity,
				RemoveAll:   removeAll,
			})
			return nil
		},
	}

	transactionId := uuid.New()
	s, err := NewBuilder().
		SetTransactionId(transactionId).
		SetSagaType(IncubatorUse).
		SetInitiatedBy("cash-item-use-compensation-test").
		AddStep("destroy_seal_item", Completed, DestroyAssetFromSlot, DestroyAssetFromSlotPayload{
			CharacterId:   testCharId,
			InventoryType: 4,
			Slot:          3,
			Quantity:      1,
			TemplateId:    sealItemId,
		}).
		AddStep("award_result_item", Completed, AwardAsset, AwardItemActionPayload{
			CharacterId: testCharId,
			Item: ItemPayload{
				TemplateId: resultItemId,
				Quantity:   1,
			},
		}).
		AddStep("incubator_result", Failed, IncubatorResult, IncubatorResultPayload{
			CharacterId: testCharId,
			ItemId:      resultItemId,
			Count:       1,
		}).
		Build()
	assert.NoError(t, err, "saga build should not fail")

	compensator := NewCompensator(logger, tctx).
		WithCompartmentProcessor(compP)

	compensator.DispatchCashItemUseRollbacks(s)

	assert.Equal(t, 1, len(createItemCalls), "the consumed seal item should be re-created exactly once")
	if len(createItemCalls) == 1 {
		assert.Equal(t, testCharId, createItemCalls[0].CharacterId)
		assert.Equal(t, sealItemId, createItemCalls[0].TemplateId, "re-created item must be the consumed seal, not the result")
		assert.Equal(t, uint32(1), createItemCalls[0].Quantity)
	}

	assert.Equal(t, 1, len(destroyItemCalls), "the awarded result item should be destroyed exactly once")
	if len(destroyItemCalls) == 1 {
		assert.Equal(t, testCharId, destroyItemCalls[0].CharacterId)
		assert.Equal(t, resultItemId, destroyItemCalls[0].TemplateId)
		assert.Equal(t, uint32(1), destroyItemCalls[0].Quantity)
	}
}

// TestCashItemUseCompensationSkipsSlotDestroyWithoutTemplateId verifies that a
// DestroyAssetFromSlot step whose payload has no TemplateId (an older/degraded
// record) is skipped rather than issuing a zero-templateId RequestCreateItem.
func TestCashItemUseCompensationSkipsSlotDestroyWithoutTemplateId(t *testing.T) {
	logger, _ := test.NewNullLogger()
	logger.SetLevel(logrus.DebugLevel)

	ctx := context.Background()
	te, _ := tenant.Create(uuid.New(), "GMS", 83, 1)
	tctx := tenant.WithContext(ctx, te)

	const testCharId = uint32(88002)

	var createItemCalls int
	compP := &compmock.ProcessorMock{
		RequestCreateItemFunc: func(_ uuid.UUID, _ uint32, _ uint32, _ uint32, _ time.Time) error {
			createItemCalls++
			return nil
		},
	}

	transactionId := uuid.New()
	s, err := NewBuilder().
		SetTransactionId(transactionId).
		SetSagaType(ItemTagUse).
		SetInitiatedBy("cash-item-use-compensation-test").
		AddStep("destroy_tag_item", Completed, DestroyAssetFromSlot, DestroyAssetFromSlotPayload{
			CharacterId:   testCharId,
			InventoryType: 4,
			Slot:          3,
			Quantity:      1,
			// TemplateId intentionally omitted (zero value)
		}).
		AddStep("incubator_result", Failed, IncubatorResult, IncubatorResultPayload{
			CharacterId: testCharId,
		}).
		Build()
	assert.NoError(t, err, "saga build should not fail")

	compensator := NewCompensator(logger, tctx).
		WithCompartmentProcessor(compP)

	compensator.DispatchCashItemUseRollbacks(s)

	assert.Equal(t, 0, createItemCalls, "a slot-destroy without a templateId must not be re-created")
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

func lateStepTestCtx(t *testing.T) context.Context {
	t.Helper()
	tm, err := tenant.Create(uuid.New(), "GMS", 83, 1)
	require.NoError(t, err)
	return tenant.WithContext(context.Background(), tm)
}

// The task-102 shape: a late-successful AwardCurrency step dispatches exactly
// one negated wallet credit, and a duplicate delivery dispatches nothing.
func TestCompensateLateStep_AwardCurrency_NegatedOnceOnly(t *testing.T) {
	ResetCache()
	logger, _ := test.NewNullLogger()
	ctx := lateStepTestCtx(t)

	s, err := NewBuilder().
		SetSagaType(InventoryTransaction).
		SetInitiatedBy("test").
		AddStep("award_currency_seller", Pending, AwardCurrency, AwardCurrencyPayload{CharacterId: 1, AccountId: 42, CurrencyType: 2, Amount: 100}).
		Build()
	require.NoError(t, err)
	require.NoError(t, GetCache().Put(ctx, s))

	var calls []int32
	cs := &cashshopmock.ProcessorMock{
		AwardCurrencyAndEmitFunc: func(txId uuid.UUID, accountId uint32, currencyType uint32, amount int32) error {
			assert.Equal(t, s.TransactionId(), txId)
			assert.Equal(t, uint32(42), accountId)
			assert.Equal(t, uint32(2), currencyType)
			calls = append(calls, amount)
			return nil
		},
	}
	c := NewCompensator(logger, ctx).WithCashshopProcessor(cs)

	step, _ := s.GetCurrentStep()
	compensated, err := c.CompensateLateStep(s, step)
	require.NoError(t, err)
	assert.True(t, compensated)
	require.Len(t, calls, 1)
	assert.Equal(t, int32(-100), calls[0], "inverse must negate the amount")

	// Duplicate delivery: marker already claimed — no second dispatch.
	fresh, ok := GetCache().GetById(ctx, s.TransactionId())
	require.True(t, ok)
	freshStep, _ := fresh.GetCurrentStep()
	assert.True(t, freshStep.LateCompensated())
	compensated, err = c.CompensateLateStep(fresh, freshStep)
	require.NoError(t, err)
	assert.False(t, compensated)
	assert.Len(t, calls, 1)
}

func TestCompensateLateStep_AwardAsset_DestroysItem(t *testing.T) {
	ResetCache()
	logger, _ := test.NewNullLogger()
	ctx := lateStepTestCtx(t)

	s, err := NewBuilder().
		SetSagaType(InventoryTransaction).
		SetInitiatedBy("test").
		AddStep("award_item", Pending, AwardAsset, AwardItemActionPayload{CharacterId: 7, Item: ItemPayload{TemplateId: 2000000, Quantity: 3}}).
		Build()
	require.NoError(t, err)
	require.NoError(t, GetCache().Put(ctx, s))

	destroyed := 0
	cp := &compmock.ProcessorMock{
		RequestDestroyItemFunc: func(txId uuid.UUID, characterId uint32, templateId uint32, quantity uint32, removeAll bool) error {
			destroyed++
			assert.Equal(t, uint32(7), characterId)
			assert.Equal(t, uint32(2000000), templateId)
			assert.Equal(t, uint32(3), quantity)
			return nil
		},
	}
	c := NewCompensator(logger, ctx).WithCompartmentProcessor(cp)

	step, _ := s.GetCurrentStep()
	compensated, err := c.CompensateLateStep(s, step)
	require.NoError(t, err)
	assert.True(t, compensated)
	assert.Equal(t, 1, destroyed)
}

// Non-compensable action: absorb-only with a late_effect_unrecoverable WARN.
func TestCompensateLateStep_NonCompensable_WarnsNoDispatch(t *testing.T) {
	ResetCache()
	logger, hook := test.NewNullLogger()
	ctx := lateStepTestCtx(t)

	s, err := NewBuilder().
		SetSagaType(InventoryTransaction).
		SetInitiatedBy("test").
		AddStep("change_hair", Pending, ChangeHair, ChangeHairPayload{CharacterId: 1}).
		Build()
	require.NoError(t, err)
	require.NoError(t, GetCache().Put(ctx, s))

	c := NewCompensator(logger, ctx)
	step, _ := s.GetCurrentStep()
	compensated, err := c.CompensateLateStep(s, step)
	require.NoError(t, err)
	assert.False(t, compensated)

	var warned bool
	for _, e := range hook.AllEntries() {
		if e.Level == logrus.WarnLevel && e.Data["reason"] == "late_effect_unrecoverable" {
			warned = true
		}
	}
	assert.True(t, warned, "expected late_effect_unrecoverable WARN")

	// Marker must NOT be claimed for a non-compensable step.
	fresh, _ := GetCache().GetById(ctx, s.TransactionId())
	freshStep, _ := fresh.GetCurrentStep()
	assert.False(t, freshStep.LateCompensated())
}

// DestroyAsset with RemoveAll=true carries no recoverable quantity in its
// step payload — the destroyed amount is not "whatever Quantity says" but
// "everything the player had". Recreating a fabricated quantity would
// silently under-refund, so this must absorb-only (like the non-compensable
// case) rather than dispatch a bogus recreate.
func TestCompensateLateStep_DestroyAssetRemoveAll_Unrecoverable(t *testing.T) {
	ResetCache()
	logger, hook := test.NewNullLogger()
	ctx := lateStepTestCtx(t)

	s, err := NewBuilder().
		SetSagaType(InventoryTransaction).
		SetInitiatedBy("test").
		AddStep("destroy_item", Pending, DestroyAsset, DestroyAssetPayload{
			CharacterId: 7,
			TemplateId:  2000000,
			Quantity:    0,
			RemoveAll:   true,
		}).
		Build()
	require.NoError(t, err)
	require.NoError(t, GetCache().Put(ctx, s))

	var createCalls, destroyCalls int
	cp := &compmock.ProcessorMock{
		RequestCreateItemFunc: func(_ uuid.UUID, _ uint32, _ uint32, _ uint32, _ time.Time) error {
			createCalls++
			return nil
		},
		RequestDestroyItemFunc: func(_ uuid.UUID, _ uint32, _ uint32, _ uint32, _ bool) error {
			destroyCalls++
			return nil
		},
	}
	c := NewCompensator(logger, ctx).WithCompartmentProcessor(cp)

	step, _ := s.GetCurrentStep()
	compensated, err := c.CompensateLateStep(s, step)
	require.NoError(t, err)
	assert.False(t, compensated)
	assert.Equal(t, 0, createCalls, "RemoveAll destroy must not dispatch a fabricated recreate")
	assert.Equal(t, 0, destroyCalls)

	var warned bool
	for _, e := range hook.AllEntries() {
		if e.Level == logrus.WarnLevel && e.Data["reason"] == "late_effect_unrecoverable" {
			warned = true
		}
	}
	assert.True(t, warned, "expected late_effect_unrecoverable WARN")

	// Marker must NOT be claimed for an unrecoverable RemoveAll destroy.
	fresh, ok := GetCache().GetById(ctx, s.TransactionId())
	require.True(t, ok)
	freshStep, _ := fresh.GetCurrentStep()
	assert.False(t, freshStep.LateCompensated())
}

// TestCompensateLateStep_ReleaseFromMtsHolding_RestoresHolding pins the MTS
// take-home late inverse (task-102): a ReleaseFromMtsHolding that soft-deleted
// the holding but landed late after the saga terminated is rolled back by
// RestoreMtsHolding on the same holding id, so the item stays in MTS custody
// (recoverable) instead of being orphaned.
func TestCompensateLateStep_ReleaseFromMtsHolding_RestoresHolding(t *testing.T) {
	ResetCache()
	logger, _ := test.NewNullLogger()
	ctx := lateStepTestCtx(t)

	holdingId := uuid.New()
	s, err := NewBuilder().
		SetSagaType(MtsOperation).
		SetInitiatedBy("test").
		AddStep("release_from_mts_holding", Pending, ReleaseFromMtsHolding, ReleaseFromMtsHoldingPayload{HoldingId: holdingId}).
		Build()
	require.NoError(t, err)
	require.NoError(t, GetCache().Put(ctx, s))

	mtsMockP := &mtsTestMtsMock{}
	c := NewCompensator(logger, ctx).WithMtsProcessor(mtsMockP)

	step, _ := s.GetCurrentStep()
	compensated, err := c.CompensateLateStep(s, step)
	require.NoError(t, err)
	assert.True(t, compensated)
	assert.Equal(t, 1, mtsMockP.restoreCalls, "late ReleaseFromMtsHolding must dispatch exactly one RestoreMtsHolding")
	assert.Equal(t, holdingId, mtsMockP.restoreHoldingId, "restore must target the same holding")

	// Duplicate delivery: marker claimed — no second restore.
	fresh, ok := GetCache().GetById(ctx, s.TransactionId())
	require.True(t, ok)
	freshStep, _ := fresh.GetCurrentStep()
	assert.True(t, freshStep.LateCompensated())
	compensated, err = c.CompensateLateStep(fresh, freshStep)
	require.NoError(t, err)
	assert.False(t, compensated)
	assert.Equal(t, 1, mtsMockP.restoreCalls)
}

// TestCompensateLateStep_AcceptToMtsListing_RemovesListing pins the late inverse
// of a spurious list-accept: the duplicate listing is removed by RemoveMtsListing.
func TestCompensateLateStep_AcceptToMtsListing_RemovesListing(t *testing.T) {
	ResetCache()
	logger, _ := test.NewNullLogger()
	ctx := lateStepTestCtx(t)

	listingId := uuid.New()
	s, err := NewBuilder().
		SetSagaType(MtsOperation).
		SetInitiatedBy("test").
		AddStep("accept_to_mts_listing", Pending, AcceptToMtsListing, AcceptToMtsListingPayload{ListingId: listingId}).
		Build()
	require.NoError(t, err)
	require.NoError(t, GetCache().Put(ctx, s))

	mtsMockP := &mtsTestMtsMock{}
	c := NewCompensator(logger, ctx).WithMtsProcessor(mtsMockP)

	step, _ := s.GetCurrentStep()
	compensated, err := c.CompensateLateStep(s, step)
	require.NoError(t, err)
	assert.True(t, compensated)
	assert.Equal(t, 1, mtsMockP.removeListingCalls)
	assert.Equal(t, listingId, mtsMockP.removeListingId)
}

// TestCompensateLateStep_MtsMove_RestoresListing pins the late inverse of a buy
// settlement-move: RestoreListingFromHolding is dispatched with (listingId,
// buyerId) so the buyer holding is removed and the listing returns to active.
func TestCompensateLateStep_MtsMove_RestoresListing(t *testing.T) {
	ResetCache()
	logger, _ := test.NewNullLogger()
	ctx := lateStepTestCtx(t)

	listingId := uuid.New()
	s, err := NewBuilder().
		SetSagaType(MtsOperation).
		SetInitiatedBy("test").
		AddStep("mts_move_listing_to_holding", Pending, MtsMoveListingToHolding, MtsMoveListingToHoldingPayload{ListingId: listingId, BuyerId: 4242}).
		Build()
	require.NoError(t, err)
	require.NoError(t, GetCache().Put(ctx, s))

	mtsMockP := &mtsTestMtsMock{}
	c := NewCompensator(logger, ctx).WithMtsProcessor(mtsMockP)

	step, _ := s.GetCurrentStep()
	compensated, err := c.CompensateLateStep(s, step)
	require.NoError(t, err)
	assert.True(t, compensated)
	assert.Equal(t, 1, mtsMockP.restoreListingCalls)
	assert.Equal(t, listingId, mtsMockP.restoreListingId)
	assert.Equal(t, uint32(4242), mtsMockP.restoreListingBuyerId)
}
